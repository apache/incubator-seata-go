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

type MysqlXAConn struct {
	driver.Conn
}

func NewMysqlXaConn(conn driver.Conn) *MysqlXAConn {
	return &MysqlXAConn{Conn: conn}
}

func (c *MysqlXAConn) Commit(ctx context.Context, xid string, onePhase bool) error {
	log.Infof("xa branch commit, xid %s", xid)

	var sb strings.Builder
	sb.WriteString("XA COMMIT ")
	sb.WriteString("'")
	sb.WriteString(xid)
	sb.WriteString("'")
	if onePhase {
		sb.WriteString(" ONE PHASE")
	}

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa branch commit failed, xid %s, err %v", xid, err)
	}
	return err
}

func (c *MysqlXAConn) End(ctx context.Context, xid string, flags int) error {
	log.Infof("xa branch end, xid %s", xid)

	var sb strings.Builder
	sb.WriteString("XA END ")
	sb.WriteString("'")
	sb.WriteString(xid)
	sb.WriteString("'")

	switch flags {
	case TMSuccess:
		break
	case TMSuspend:
		sb.WriteString(" SUSPEND")
		break
	case TMFail:
		break
	default:
		return errors.New("invalid arguments")
	}

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa branch end failed, xid %s, err %v", xid, err)
	}
	return err
}

func (c *MysqlXAConn) Forget(ctx context.Context, xid string) error {
	// mysql doesn't support this
	return errors.New("mysql doesn't support this")
}

func (c *MysqlXAConn) GetTransactionTimeout() time.Duration {
	return 0
}

// IsSameRM is called to determine if the resource manager instance represented by the target object
// is the same as the resource manager instance represented by the parameter xares.
func (c *MysqlXAConn) IsSameRM(ctx context.Context, xares XAResource) bool {
	// todo: the fn depends on the driver.Conn, but it doesn't support
	return false
}

func (c *MysqlXAConn) XAPrepare(ctx context.Context, xid string) error {
	log.Infof("xa branch prepare, xid %s", xid)

	var sb strings.Builder
	sb.WriteString("XA PREPARE ")
	sb.WriteString("'")
	sb.WriteString(xid)
	sb.WriteString("'")

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa branch prepare failed, xid %s, err %v", xid, err)
	}
	return err
}

// Recover Obtains a list of prepared transaction branches from a resource manager.
// The transaction manager calls this method during recovery to obtain the list of transaction branches
// that are currently in prepared or heuristically completed states.
func (c *MysqlXAConn) Recover(ctx context.Context, flag int) (xids []string, err error) {
	startRscan := (flag & TMStartRScan) > 0
	endRscan := (flag & TMEndRScan) > 0

	if !startRscan && !endRscan && flag != TMNoFlags {
		return nil, errors.New("invalid arguments")
	}

	if !startRscan {
		return nil, nil
	}

	conn := c.Conn.(driver.QueryerContext)
	res, err := conn.QueryContext(ctx, "XA RECOVER", nil)
	if err != nil {
		return nil, err
	}

	dest := make([]driver.Value, 4)
	for true {
		if err = res.Next(dest); err != nil {
			if err == io.EOF {
				return xids, nil
			}
			return nil, err
		}
		gtridAndbqual, ok := dest[3].(string)
		if !ok {
			return nil, errors.New("the protocol of XA RECOVER statement is error")
		}
		fmt.Printf("gtr: %v", gtridAndbqual)

		xids = append(xids, string(gtridAndbqual))
	}
	return xids, err
}

func (c *MysqlXAConn) Rollback(ctx context.Context, xid string) error {
	log.Infof("xa branch rollback, xid %s", xid)

	var sb strings.Builder
	sb.WriteString("XA ROLLBACK ")
	sb.WriteString("'")
	sb.WriteString(xid)
	sb.WriteString("'")

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa branch rollback failed, xid %s, err %v", xid, err)
	}
	return err
}

func (c *MysqlXAConn) SetTransactionTimeout(duration time.Duration) bool {
	return false
}

func (c *MysqlXAConn) Start(ctx context.Context, xid string, flags int) error {
	log.Infof("xa branch start, xid %s", xid)

	var sb strings.Builder
	sb.WriteString("XA START ")
	sb.WriteString("'")
	sb.WriteString(xid)
	sb.WriteString("'")

	switch flags {
	case TMJoin:
		sb.WriteString(" JOIN")
		break
	case TMResume:
		sb.WriteString(" RESUME")
		break
	case TMNoFlags:
		break
	default:
		return errors.New("invalid arguments")
	}

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(ctx, sb.String(), nil)
	if err != nil {
		log.Errorf("xa branch start failed, xid %s, err %v", xid, err)
	}
	return err
}
