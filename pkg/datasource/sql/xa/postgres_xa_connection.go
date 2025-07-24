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
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"seata.apache.org/seata-go/pkg/util/log"
)

type PostgresqlXAConn struct {
	driver.Conn
}

func NewPostgresqlXaConn(conn driver.Conn) *PostgresqlXAConn {
	return &PostgresqlXAConn{Conn: conn}
}

// Auxiliary function: Split xid into the required (gtrid, bqual, formatID) for PostgreSQL
// The xid format of PostgreSQL is "gtrid, bqual, formatID" (string, string, integer)
func splitXID(xid string) (gtrid, bqual string, formatID int, err error) {
	parts := strings.Split(xid, ",")
	if len(parts) != 3 {
		return "", "", 0, errors.New("invalid xid format for postgresql (expected 'gtrid,bqual,formatID')")
	}
	gtrid = parts[0]
	bqual = parts[1]
	formatID, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid formatID in xid: %v", err)
	}
	return
}

func (c *PostgresqlXAConn) Commit(ctx context.Context, xid string, onePhase bool) error {
	gtrid, bqual, formatID, err := splitXID(xid)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("XA COMMIT '")
	sb.WriteString(gtrid)
	sb.WriteString("', '")
	sb.WriteString(bqual)
	sb.WriteString("', ")
	sb.WriteString(strconv.Itoa(formatID))

	if onePhase {
		sb.WriteString(" ONE PHASE")
	}

	conn, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return errors.New("postgresql conn does not support ExecerContext")
	}
	_, err = conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa branch commit failed, xid %s, err %v", xid, err)
	}
	return err
}

func (c *PostgresqlXAConn) End(ctx context.Context, xid string, flags int) error {
	gtrid, bqual, formatID, err := splitXID(xid)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("XA END '")
	sb.WriteString(gtrid)
	sb.WriteString("', '")
	sb.WriteString(bqual)
	sb.WriteString("', ")
	sb.WriteString(strconv.Itoa(formatID))

	switch flags {
	case TMSuccess:
	case TMSuspend:
		sb.WriteString(" SUSPEND")
	case TMFail:

	default:
		return errors.New("invalid flags for End (support TMSuccess, TMSuspend, TMFail)")
	}

	conn, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return errors.New("postgresql conn does not support ExecerContext")
	}
	_, err = conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa branch end failed, xid %s, err %v", xid, err)
	}
	return err
}

func (c *PostgresqlXAConn) Forget(ctx context.Context, xid string) error {
	gtrid, bqual, formatID, err := splitXID(xid)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("XA FORGET '")
	sb.WriteString(gtrid)
	sb.WriteString("', '")
	sb.WriteString(bqual)
	sb.WriteString("', ")
	sb.WriteString(strconv.Itoa(formatID))

	conn, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return errors.New("postgresql conn does not support ExecerContext")
	}
	_, err = conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa forget failed, xid %s, err %v", xid, err)
	}
	return err
}

func (c *PostgresqlXAConn) GetTransactionTimeout() time.Duration {
	return 0
}

// IsSameRM is called to determine if the resource manager instance represented by the target object
// is the same as the resource manager instance represented by the parameter xares.
func (c *PostgresqlXAConn) IsSameRM(ctx context.Context, resource XAResource) bool {
	// todo:It is supported on the driver. Whether to implement it depends on whether to implement "XA transaction level timeout control"
	return false
}

func (c *PostgresqlXAConn) XAPrepare(ctx context.Context, xid string) error {
	gtrid, bqual, formatID, err := splitXID(xid)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("XA PREPARE '")
	sb.WriteString(gtrid)
	sb.WriteString("', '")
	sb.WriteString(bqual)
	sb.WriteString("', ")
	sb.WriteString(strconv.Itoa(formatID))

	conn, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return errors.New("postgresql conn does not support ExecerContext")
	}
	_, err = conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa prepare failed, xid %s, err %v", xid, err)
	}
	return err
}

// Recover
func (c *PostgresqlXAConn) Recover(ctx context.Context, flag int) ([]string, error) {
	startRscan := (flag & TMStartRScan) > 0
	endRscan := (flag & TMEndRScan) > 0

	if !startRscan && !endRscan && flag != TMNoFlags {
		return nil, errors.New("invalid recover flag for postgresql")
	}
	if !startRscan {
		return nil, nil
	}

	conn, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, errors.New("postgresql conn does not support QueryerContext")
	}

	res, err := conn.QueryContext(ctx, "XA RECOVER FORMATAS TEXT", nil)
	if err != nil {
		return nil, fmt.Errorf("xa recover failed: %v", err)
	}
	defer res.Close()

	var xids []string
	dest := make([]driver.Value, 3)
	for {
		if err = res.Next(dest); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("recover next failed: %v", err)
		}

		formatID, ok := dest[0].(int64)
		if !ok {
			return nil, errors.New("invalid formatID type in recover result")
		}
		gtrid, ok := dest[1].(string)
		if !ok {
			return nil, errors.New("invalid gtrid type in recover result")
		}
		bqual, ok := dest[2].(string)
		if !ok {
			return nil, errors.New("invalid bqual type in recover result")
		}

		xid := fmt.Sprintf("%s,%s,%d", gtrid, bqual, formatID)
		xids = append(xids, xid)
	}

	return xids, nil
}

// Rollback
func (c *PostgresqlXAConn) Rollback(ctx context.Context, xid string) error {
	gtrid, bqual, formatID, err := splitXID(xid)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("XA ROLLBACK '")
	sb.WriteString(gtrid)
	sb.WriteString("', '")
	sb.WriteString(bqual)
	sb.WriteString("', ")
	sb.WriteString(strconv.Itoa(formatID))

	conn, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return errors.New("postgresql conn does not support ExecerContext")
	}
	_, err = conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa rollback failed, xid %s, err %v", xid, err)
	}
	return err
}

func (c *PostgresqlXAConn) SetTransactionTimeout(duration time.Duration) bool {
	return false
}

func (c *PostgresqlXAConn) Start(ctx context.Context, xid string, flags int) error {
	gtrid, bqual, formatID, err := splitXID(xid)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("XA START '")
	sb.WriteString(gtrid)
	sb.WriteString("', '")
	sb.WriteString(bqual)
	sb.WriteString("', ")
	sb.WriteString(strconv.Itoa(formatID))

	switch flags {
	case TMJoin:
		sb.WriteString(" JOIN")
	case TMResume:
		sb.WriteString(" RESUME")
	case TMNoFlags:
	default:
		return errors.New("invalid flags for start (support TMJoin, TMResume, TMNoFlags)")
	}

	conn, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return errors.New("postgresql conn does not support ExecerContext")
	}
	_, err = conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa start failed, xid %s, err %v", xid, err)
	}
	return err
}
