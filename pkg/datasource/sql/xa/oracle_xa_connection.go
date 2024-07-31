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
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

type OracleXAConn struct {
	driver.Conn
}

func (c *OracleXAConn) Commit(xid string, onePhase bool) error {
	var sb strings.Builder
	sb.WriteString("XA COMMIT ")
	sb.WriteString(xid)
	if onePhase {
		sb.WriteString(" ONE PHASE")
	}

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(context.TODO(), sb.String(), nil)
	return err
}

func (c *OracleXAConn) End(xid string, flags int) error {
	var sb strings.Builder
	sb.WriteString("XA END ")
	sb.WriteString(xid)

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(context.TODO(), sb.String(), nil)
	return err
}

func (c *OracleXAConn) Forget(xid string) error {
	// TODO implement me
	panic("implement me")
}

func (c *OracleXAConn) GetTransactionTimeout() time.Duration {
	// TODO implement me
	panic("implement me")
}

func (c *OracleXAConn) IsSameRM(resource XAResource) bool {
	// TODO implement me
	panic("implement me")
}

func (c *OracleXAConn) XAPrepare(xid string) (int, error) {
	var sb strings.Builder
	sb.WriteString("XA PREPARE ")
	sb.WriteString(xid)

	conn, _ := c.Conn.(driver.ExecerContext)
	if _, err := conn.ExecContext(context.TODO(), sb.String(), nil); err != nil {
		return -1, err
	}
	return 0, nil
}

func (c *OracleXAConn) Recover(flag int) []string {
	// TODO implement me
	panic("implement me")
}

func (c *OracleXAConn) Rollback(xid string) error {
	var sb strings.Builder
	sb.WriteString("XA ROLLBACK ")
	sb.WriteString(xid)

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(context.TODO(), sb.String(), nil)
	return err
}

func (c *OracleXAConn) SetTransactionTimeout(duration time.Duration) bool {
	// TODO implement me
	panic("implement me")
}

func (c *OracleXAConn) Start(xid string, flags int) error {
	var sb strings.Builder
	sb.WriteString("XA START")
	sb.WriteString(xid)

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(context.TODO(), sb.String(), nil)
	return err
}
