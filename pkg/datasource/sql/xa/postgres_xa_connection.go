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
	"fmt"
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
}

func (c *PostgresXAConn) Start(ctx context.Context, xid string, flags int) error {
	log.Infof("xa branch start (postgres no-op), xid %s", xid)
	return nil
}

func (c *PostgresXAConn) End(ctx context.Context, xid string, flags int) error {
	log.Infof("xa branch end (postgres no-op), xid %s", xid)
	return nil
}

func (c *PostgresXAConn) XAPrepare(ctx context.Context, xid string) error {
	// TODO: PREPARE TRANSACTION 'xid'
	return fmt.Errorf("PostgreSQL XA prepare not yet implemented")
}

func (c *PostgresXAConn) Commit(ctx context.Context, xid string, onePhase bool) error {
	// TODO: COMMIT PREPARED 'xid' (onePhase=true → regular commit by upper layer)
	return fmt.Errorf("PostgreSQL XA commit not yet implemented")
}

func (c *PostgresXAConn) Rollback(ctx context.Context, xid string) error {
	// TODO: ROLLBACK PREPARED 'xid'
	return fmt.Errorf("PostgreSQL XA rollback not yet implemented")
}

func (c *PostgresXAConn) Recover(ctx context.Context, flag int) ([]string, error) {
	if (flag & TMStartRScan) == 0 {
		return nil, nil
	}
	// TODO: SELECT gid FROM pg_prepared_xacts WHERE database = current_database()
	return nil, fmt.Errorf("PostgreSQL XA recover not yet implemented")
}

func (c *PostgresXAConn) Forget(ctx context.Context, xid string) error {
	return fmt.Errorf("PostgreSQL does not support XA forget")
}

func (c *PostgresXAConn) GetTransactionTimeout() time.Duration { return 0 }

func (c *PostgresXAConn) IsSameRM(ctx context.Context, resource XAResource) bool { return false }

func (c *PostgresXAConn) SetTransactionTimeout(duration time.Duration) bool { return false }
