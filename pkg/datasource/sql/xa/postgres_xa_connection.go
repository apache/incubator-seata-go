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

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

func init() {
	RegisterXAResourceFactory(types.DBTypePostgreSQL, &postgresXAResourceFactory{})
}

type postgresXAResourceFactory struct{}

func (f *postgresXAResourceFactory) CreateXAResource(conn driver.Conn) XAResource {
	return &PostgresXAConn{Conn: conn}
}

func (f *postgresXAResourceFactory) CreateErrorClassifier() XAErrorClassifier {
	return &PostgresXAErrorClassifier{}
}

// PostgresXAErrorClassifier classifies PostgreSQL-specific XA errors.
type PostgresXAErrorClassifier struct{}

func (c *PostgresXAErrorClassifier) IsAlreadyEnded(err error) bool {
	// TODO: check pgconn.PgError SQLSTATE "42704" / "55000"
	return false
}

// PostgresXAConn implements XAResource for PostgreSQL using native 2PC.
//
// Key differences from MySQL XA:
//   - Start/End are no-ops (PostgreSQL uses regular BEGIN, no XA START/END).
//   - XAPrepare → PREPARE TRANSACTION 'xid'
//   - Commit    → COMMIT PREPARED 'xid'
//   - Rollback  → ROLLBACK PREPARED 'xid'
//   - Recover   → SELECT gid FROM pg_prepared_xacts
//   - Requires max_prepared_transactions > 0 in postgresql.conf.
type PostgresXAConn struct {
	driver.Conn
	tx driver.Tx
}

func (c *PostgresXAConn) Start(ctx context.Context, xid string, flags int) error {
	log.Infof("xa branch start (postgres begin), xid %s", xid)

	if flags != TMNoFlags {
		return errors.New("invalid arguments")
	}
	if c.tx != nil {
		return fmt.Errorf("postgres xa transaction already started, xid %s", xid)
	}

	var (
		tx  driver.Tx
		err error
	)
	if conn, ok := c.Conn.(driver.ConnBeginTx); ok {
		tx, err = conn.BeginTx(ctx, driver.TxOptions{})
	} else {
		tx, err = c.Conn.Begin()
	}
	if err != nil {
		log.Errorf("postgres xa branch start failed, xid %s, err %v", xid, err)
		return err
	}

	c.tx = tx
	return nil
}

func (c *PostgresXAConn) End(ctx context.Context, xid string, flags int) error {
	log.Infof("xa branch end (postgres no-op), xid %s", xid)

	switch flags {
	case TMSuccess, TMFail:
		return nil
	default:
		return errors.New("invalid arguments")
	}
}

func (c *PostgresXAConn) XAPrepare(ctx context.Context, xid string) error {
	log.Infof("postgres xa branch prepare, xid %s", xid)

	if c.tx == nil {
		return fmt.Errorf("postgres xa prepare requires active transaction, xid %s", xid)
	}

	query := "PREPARE TRANSACTION " + quotePostgresXID(xid)
	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(ctx, query, nil)
	if err != nil {
		log.Errorf("postgres xa branch prepare failed, xid %s, err %v", xid, err)
		return err
	}

	c.tx = nil
	return nil
}

func (c *PostgresXAConn) Commit(ctx context.Context, xid string, onePhase bool) error {
	if onePhase {
		if c.tx == nil {
			return fmt.Errorf("postgres xa one-phase commit requires active transaction, xid %s", xid)
		}
		if err := c.tx.Commit(); err != nil {
			log.Errorf("postgres xa one-phase commit failed, xid %s, err %v", xid, err)
			return err
		}
		c.tx = nil
		return nil
	}

	if c.tx != nil {
		return fmt.Errorf("postgres xa commit requires prepared transaction, xid %s", xid)
	}

	log.Infof("postgres xa branch commit prepared, xid %s", xid)

	query := "COMMIT PREPARED " + quotePostgresXID(xid)
	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(ctx, query, nil)
	if err != nil {
		log.Errorf("postgres xa branch commit prepared failed, xid %s, err %v", xid, err)
	}
	return err
}

func (c *PostgresXAConn) Rollback(ctx context.Context, xid string) error {
	if c.tx != nil {
		log.Infof("postgres xa branch rollback active transaction, xid %s", xid)
		err := c.tx.Rollback()
		if err != nil {
			log.Errorf("postgres xa branch rollback active transaction failed, xid %s, err %v", xid, err)
			return err
		}
		c.tx = nil
		return nil
	}

	log.Infof("postgres xa branch rollback prepared, xid %s", xid)

	query := "ROLLBACK PREPARED " + quotePostgresXID(xid)
	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(ctx, query, nil)
	if err != nil {
		log.Errorf("postgres xa branch rollback prepared failed, xid %s, err %v", xid, err)
	}
	return err
}

func (c *PostgresXAConn) Recover(ctx context.Context, flag int) ([]string, error) {
	startRscan := (flag & TMStartRScan) > 0
	endRscan := (flag & TMEndRScan) > 0

	if !startRscan && !endRscan && flag != TMNoFlags {
		return nil, errors.New("invalid arguments")
	}
	if !startRscan {
		return nil, nil
	}

	conn := c.Conn.(driver.QueryerContext)
	rows, err := conn.QueryContext(ctx, "SELECT gid FROM pg_prepared_xacts WHERE database = current_database()", nil)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	xids := make([]string, 0)
	dest := make([]driver.Value, 1)
	for {
		if err = rows.Next(dest); err != nil {
			if err == io.EOF {
				return xids, nil
			}
			return nil, err
		}

		switch v := dest[0].(type) {
		case string:
			xids = append(xids, v)
		case []byte:
			xids = append(xids, string(v))
		default:
			return nil, errors.New("the protocol of postgres prepared transaction query is error")
		}
	}
}

func (c *PostgresXAConn) Forget(ctx context.Context, xid string) error {
	return fmt.Errorf("PostgreSQL does not support XA forget")
}

func (c *PostgresXAConn) GetTransactionTimeout() time.Duration { return 0 }

func (c *PostgresXAConn) IsSameRM(ctx context.Context, resource XAResource) bool { return false }

func (c *PostgresXAConn) SetTransactionTimeout(duration time.Duration) bool { return false }

func quotePostgresXID(xid string) string {
	return "'" + strings.ReplaceAll(xid, "'", "''") + "'"
}
