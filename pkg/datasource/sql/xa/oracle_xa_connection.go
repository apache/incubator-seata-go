/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package xa

import (
	"context"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

const (
	oracleXAMaxXIDPartSize = 64
	oracleXAFormatID       = 0x53474F
	oracleXAOK             = 0
)

var _ XAResource = (*OracleXAConn)(nil)

func init() {
	RegisterXAResourceFactory(types.DBTypeOracle, &oracleXAResourceFactory{})
}

type oracleXAResourceFactory struct{}

func (f *oracleXAResourceFactory) CreateXAResource(conn driver.Conn) XAResource {
	return NewOracleXaConn(conn)
}

func (f *oracleXAResourceFactory) CreateErrorClassifier() XAErrorClassifier {
	return &OracleXAErrorClassifier{}
}

// OracleXAErrorClassifier recognizes Oracle errors that indicate an XA branch has already ended.
type OracleXAErrorClassifier struct{}

func (c *OracleXAErrorClassifier) IsAlreadyEnded(err error) bool {
	return oracleXAHasErrorCode(err, "24756") || oracleXAHasErrorCode(err, "24761")
}

func (c *OracleXAErrorClassifier) IsAlreadyCommitted(err error) bool {
	return oracleXAHasErrorCode(err, "24756")
}

func (c *OracleXAErrorClassifier) IsAlreadyRollbacked(err error) bool {
	return oracleXAHasErrorCode(err, "24761")
}

func oracleXAHasErrorCode(err error, code string) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "ORA-"+code) ||
		strings.Contains(message, "oer="+code)
}

// OracleXAConn implements Oracle XA lifecycle operations through DBMS_XA.
type OracleXAConn struct {
	driver.Conn
}

type oracleXID struct {
	formatID            int
	globalTransactionID []byte
	branchQualifier     []byte
}

func NewOracleXaConn(conn driver.Conn) *OracleXAConn {
	return &OracleXAConn{Conn: conn}
}

func (c *OracleXAConn) Start(ctx context.Context, xid string, flags int) error {
	xaXID, err := oracleXABranchXid(xid)
	if err != nil {
		return err
	}

	flagExpr, err := oracleXAStartFlag(flags)
	if err != nil {
		return err
	}

	return c.exec(ctx, oracleXAWithXidBlock(
		"XA_START",
		xaXID,
		fmt.Sprintf("DBMS_XA.XA_START(l_xid, %s)", flagExpr),
	))
}

func (c *OracleXAConn) End(ctx context.Context, xid string, flags int) error {
	xaXID, err := oracleXABranchXid(xid)
	if err != nil {
		return err
	}

	flagExpr, err := oracleXAEndFlag(flags)
	if err != nil {
		return err
	}

	return c.exec(ctx, oracleXAWithXidBlock(
		"XA_END",
		xaXID,
		fmt.Sprintf("DBMS_XA.XA_END(l_xid, %s)", flagExpr),
	))
}

func (c *OracleXAConn) XAPrepare(ctx context.Context, xid string) error {
	xaXID, err := oracleXABranchXid(xid)
	if err != nil {
		return err
	}

	ret, err := c.queryInt(ctx, oracleXASelectXidStatement(xaXID, "DBMS_XA.XA_PREPARE"))
	if err != nil {
		return err
	}
	switch ret {
	case oracleXAOK:
		return nil
	case XAReadOnly:
		return ErrXAReadOnly
	default:
		return c.xaError(ctx, "XA_PREPARE", ret)
	}
}

func (c *OracleXAConn) Commit(ctx context.Context, xid string, onePhase bool) error {
	xaXID, err := oracleXABranchXid(xid)
	if err != nil {
		return err
	}

	return c.exec(ctx, oracleXAWithXidBlock(
		"XA_COMMIT",
		xaXID,
		fmt.Sprintf("DBMS_XA.XA_COMMIT(l_xid, %s)", oracleBooleanLiteral(onePhase)),
	))
}

func (c *OracleXAConn) Rollback(ctx context.Context, xid string) error {
	xaXID, err := oracleXABranchXid(xid)
	if err != nil {
		return err
	}

	return c.exec(ctx, oracleXAWithXidBlock("XA_ROLLBACK", xaXID, "DBMS_XA.XA_ROLLBACK(l_xid)"))
}

func (c *OracleXAConn) Recover(ctx context.Context, flag int) ([]string, error) {
	if flag&TMStartRScan == 0 {
		return nil, nil
	}
	return nil, errors.New("oracle xa recover is not supported")
}

func (c *OracleXAConn) Forget(ctx context.Context, xid string) error {
	return errors.New("oracle xa forget is not supported")
}

func (c *OracleXAConn) GetTransactionTimeout() time.Duration {
	return 0
}

func (c *OracleXAConn) IsSameRM(ctx context.Context, resource XAResource) bool {
	other, ok := resource.(*OracleXAConn)
	return ok && c.Conn == other.Conn
}

func (c *OracleXAConn) SetTransactionTimeout(duration time.Duration) bool {
	return false
}

func (c *OracleXAConn) exec(ctx context.Context, query string) error {
	conn, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return driver.ErrSkip
	}

	_, err := conn.ExecContext(ctx, query, nil)
	return err
}

func (c *OracleXAConn) queryInt(ctx context.Context, query string) (int, error) {
	conn, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return 0, driver.ErrSkip
	}

	rows, err := conn.QueryContext(ctx, query, nil)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	dest := make([]driver.Value, len(rows.Columns()))
	if len(dest) == 0 {
		return 0, errors.New("oracle xa query returned no columns")
	}
	if err = rows.Next(dest); err != nil {
		if err == io.EOF {
			return 0, errors.New("oracle xa query returned no rows")
		}
		return 0, err
	}
	return oracleXAInt(dest[0])
}

func (c *OracleXAConn) xaError(ctx context.Context, operation string, ret int) error {
	oer, err := c.queryInt(ctx, "SELECT DBMS_XA.XA_GETLASTOER() FROM DUAL")
	if err != nil {
		return fmt.Errorf("DBMS_XA.%s:ret=%d,oer=<unknown>: %w", operation, ret, err)
	}
	return fmt.Errorf("DBMS_XA.%s:ret=%d,oer=%d", operation, ret, oer)
}

func oracleXABranchXid(xid string) (*oracleXID, error) {
	branchSplitIdx := strings.LastIndex(xid, "-")
	if branchSplitIdx <= 0 || branchSplitIdx == len(xid)-1 {
		return nil, fmt.Errorf("invalid xa branch xid: %s", xid)
	}
	if _, err := strconv.ParseUint(xid[branchSplitIdx+1:], 10, 64); err != nil {
		return nil, fmt.Errorf("invalid xa branch xid: %s", xid)
	}

	xaXID := &oracleXID{
		formatID:            oracleXAFormatID,
		globalTransactionID: []byte(xid[:branchSplitIdx]),
		branchQualifier:     []byte(xid[branchSplitIdx:]),
	}
	if len(xaXID.globalTransactionID) > oracleXAMaxXIDPartSize {
		return nil, fmt.Errorf("oracle xa gtrid exceeds %d bytes", oracleXAMaxXIDPartSize)
	}
	if len(xaXID.branchQualifier) > oracleXAMaxXIDPartSize {
		return nil, fmt.Errorf("oracle xa bqual exceeds %d bytes", oracleXAMaxXIDPartSize)
	}
	return xaXID, nil
}

func oracleXAStartFlag(flags int) (string, error) {
	switch flags {
	case TMNoFlags:
		return "DBMS_XA.TMNOFLAGS", nil
	case TMJoin:
		return "DBMS_XA.TMJOIN", nil
	case TMResume:
		return "DBMS_XA.TMRESUME", nil
	default:
		return "", errors.New("invalid arguments")
	}
}

func oracleXAEndFlag(flags int) (string, error) {
	switch flags {
	case TMSuccess:
		return "DBMS_XA.TMSUCCESS", nil
	case TMFail:
		return "", errors.New("oracle xa end TMFAIL is not supported")
	case TMSuspend:
		return "DBMS_XA.TMSUSPEND", nil
	default:
		return "", errors.New("invalid arguments")
	}
}

func oracleXASelectXidStatement(xid *oracleXID, function string) string {
	return fmt.Sprintf("SELECT %s(DBMS_XA_XID(%d, HEXTORAW('%s'), HEXTORAW('%s'))) FROM DUAL",
		function,
		xid.formatID,
		hex.EncodeToString(xid.globalTransactionID),
		hex.EncodeToString(xid.branchQualifier),
	)
}

func oracleXAInt(value driver.Value) (int, error) {
	switch v := value.(type) {
	case int64:
		return int(v), nil
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(v))
	case []byte:
		return strconv.Atoi(strings.TrimSpace(string(v)))
	default:
		return 0, fmt.Errorf("oracle xa returned unexpected value type %T", value)
	}
}

func oracleXAWithXidBlock(operation string, xid *oracleXID, statement string, allowedReturns ...string) string {
	if len(allowedReturns) == 0 {
		allowedReturns = []string{strconv.Itoa(oracleXAOK)}
	}
	return fmt.Sprintf(`DECLARE
	l_xid DBMS_XA_XID := DBMS_XA_XID(%d, HEXTORAW('%s'), HEXTORAW('%s'));
	l_ret PLS_INTEGER;
BEGIN
	l_ret := %s;
	IF l_ret NOT IN (%s) THEN
		RAISE_APPLICATION_ERROR(-20000, 'DBMS_XA.%s:ret=' || TO_CHAR(l_ret) || ',oer=' || TO_CHAR(DBMS_XA.XA_GETLASTOER()));
	END IF;
END;`,
		xid.formatID,
		hex.EncodeToString(xid.globalTransactionID),
		hex.EncodeToString(xid.branchQualifier),
		statement,
		strings.Join(allowedReturns, ", "),
		operation,
	)
}

func oracleBooleanLiteral(value bool) string {
	if value {
		return "TRUE"
	}
	return "FALSE"
}
