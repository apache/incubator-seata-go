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
	"strings"
	"time"

	"seata.apache.org/seata-go/pkg/util/log"
)

type PostgresqlXAConn struct {
	driver.Conn
	tx          driver.Tx
	prepared    bool
	preparedXid string
	timeout     time.Duration
}

func NewPostgresqlXaConn(conn driver.Conn, tx driver.Tx) *PostgresqlXAConn {
	return &PostgresqlXAConn{
		Conn:        conn,
		tx:          tx,
		prepared:    false,
		preparedXid: "",
	}
}

func validateXid(xid string) error {
	if xid == "" {
		return errors.New("xid cannot be empty")
	}
	if len(xid) > 200 {
		return errors.New("xid too long (max 200 characters for PostgreSQL)")
	}
	if strings.ContainsAny(xid, "'\"\\;") {
		return errors.New("xid contains invalid characters")
	}
	if strings.Contains(xid, "--") || strings.Contains(xid, "/*") || strings.Contains(xid, "*/") {
		return errors.New("xid contains SQL comment patterns")
	}
	return nil
}

func (c *PostgresqlXAConn) Commit(ctx context.Context, xid string, onePhase bool) error {
	if err := validateXid(xid); err != nil {
		return fmt.Errorf("invalid xid: %v", err)
	}

	var query string
	if onePhase {
		if c.prepared && c.preparedXid == xid {
			query = fmt.Sprintf("COMMIT PREPARED '%s'", xid)
		} else {
			query = "COMMIT"
		}
	} else {
		query = fmt.Sprintf("COMMIT PREPARED '%s'", xid)
	}

	connExec, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return errors.New("postgresql conn does not support ExecerContext")
	}

	_, err := connExec.ExecContext(ctx, query, nil)
	if err != nil {
		log.Errorf("postgresql xa commit failed, xid %s, err %v", xid, err)
		return err
	}

	if c.preparedXid == xid {
		c.prepared = false
		c.preparedXid = ""
	}
	return nil
}

func (c *PostgresqlXAConn) End(ctx context.Context, xid string, flags int) error {
	if err := validateXid(xid); err != nil {
		return fmt.Errorf("invalid xid: %v", err)
	}

	switch flags {
	case TMSuccess:
		return nil
	case TMSuspend:
		return errors.New("postgresql does not support transaction suspension")
	case TMFail:
		return nil
	default:
		return fmt.Errorf("invalid flags for End: %d", flags)
	}
}

func (c *PostgresqlXAConn) Forget(ctx context.Context, xid string) error {
	if err := validateXid(xid); err != nil {
		return fmt.Errorf("invalid xid: %v", err)
	}

	exists, err := c.checkPreparedTransactionExists(ctx, xid)
	if err != nil {
		return fmt.Errorf("failed to check prepared transaction: %v", err)
	}

	if !exists {
		return nil
	}

	return c.Rollback(ctx, xid)
}

func (c *PostgresqlXAConn) checkPreparedTransactionExists(ctx context.Context, xid string) (bool, error) {
	connQuery, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return false, errors.New("postgresql conn does not support QueryerContext")
	}

	query := "SELECT 1 FROM pg_prepared_xacts WHERE gid = $1"
	rows, err := connQuery.QueryContext(ctx, query, []driver.NamedValue{
		{Value: xid, Ordinal: 1},
	})
	if err != nil {
		return false, err
	}
	defer rows.Close()

	dest := make([]driver.Value, 1)
	err = rows.Next(dest)
	if err == io.EOF {
		return false, nil
	}
	return err == nil, err
}

func (c *PostgresqlXAConn) GetTransactionTimeout() time.Duration {
	return c.timeout
}

func (c *PostgresqlXAConn) IsSameRM(ctx context.Context, resource XAResource) bool {
	pgResource, ok := resource.(*PostgresqlXAConn)
	if !ok {
		return false
	}
	return c.Conn == pgResource.Conn
}

func (c *PostgresqlXAConn) XAPrepare(ctx context.Context, xid string) error {
	if err := validateXid(xid); err != nil {
		return fmt.Errorf("invalid xid: %v", err)
	}

	query := fmt.Sprintf("PREPARE TRANSACTION '%s'", xid)

	txExec, ok := c.tx.(driver.ExecerContext)
	if !ok {
		return errors.New("postgresql tx does not support ExecerContext")
	}

	_, err := txExec.ExecContext(ctx, query, nil)
	if err != nil {
		log.Errorf("postgresql xa prepare failed, xid %s, err %v", xid, err)
		return err
	}

	c.prepared = true
	c.preparedXid = xid
	return nil
}

func (c *PostgresqlXAConn) Recover(ctx context.Context, flag int) ([]string, error) {
	startRscan := (flag & TMStartRScan) > 0
	endRscan := (flag & TMEndRScan) > 0

	if !startRscan && !endRscan && flag != TMNoFlags {
		return nil, fmt.Errorf("invalid recover flag: %d", flag)
	}

	if !startRscan {
		return nil, nil
	}

	connQuery, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, errors.New("postgresql conn does not support QueryerContext")
	}

	query := "SELECT gid FROM pg_prepared_xacts ORDER BY gid"
	rows, err := connQuery.QueryContext(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("postgresql xa recover failed: %v", err)
	}
	defer rows.Close()

	var xids []string
	dest := make([]driver.Value, 1)
	for {
		err = rows.Next(dest)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("recover next failed: %v", err)
		}

		gid, ok := dest[0].(string)
		if !ok {
			return nil, errors.New("invalid gid type in recover result")
		}

		xids = append(xids, gid)
	}

	return xids, nil
}

func (c *PostgresqlXAConn) Rollback(ctx context.Context, xid string) error {
	if err := validateXid(xid); err != nil {
		return fmt.Errorf("invalid xid: %v", err)
	}

	query := fmt.Sprintf("ROLLBACK PREPARED '%s'", xid)

	connExec, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return errors.New("postgresql conn does not support ExecerContext")
	}

	_, err := connExec.ExecContext(ctx, query, nil)
	if err != nil {
		log.Errorf("postgresql xa rollback failed, xid %s, err %v", xid, err)
		return err
	}

	if c.preparedXid == xid {
		c.prepared = false
		c.preparedXid = ""
	}

	return nil
}

func (c *PostgresqlXAConn) SetTransactionTimeout(duration time.Duration) bool {
	if duration <= 0 {
		return false
	}

	connExec, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return false
	}

	timeoutMs := int(duration.Milliseconds())
	query := fmt.Sprintf("SET LOCAL statement_timeout = %d", timeoutMs)

	_, err := connExec.ExecContext(context.Background(), query, nil)
	if err != nil {
		log.Errorf("failed to set postgresql transaction timeout: %v", err)
		return false
	}

	c.timeout = duration
	return true
}

func (c *PostgresqlXAConn) Start(ctx context.Context, xid string, flags int) error {
	if err := validateXid(xid); err != nil {
		return fmt.Errorf("invalid xid: %v", err)
	}

	switch flags {
	case TMJoin:
		return errors.New("postgresql does not support transaction joining")
	case TMResume:
		return errors.New("postgresql does not support transaction resuming")
	case TMNoFlags:
		return nil
	default:
		return fmt.Errorf("invalid flags for Start: %d", flags)
	}
}
