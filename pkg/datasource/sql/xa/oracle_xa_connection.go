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
	RegisterXAResourceFactory(types.DBTypeOracle, &oracleXAResourceFactory{})
}

type oracleXAResourceFactory struct{}

func (f *oracleXAResourceFactory) CreateXAResource(conn driver.Conn) XAResource {
	return &OracleXAConn{Conn: conn}
}

func (f *oracleXAResourceFactory) CreateErrorClassifier() XAErrorClassifier {
	return &OracleXAErrorClassifier{}
}

// OracleXAErrorClassifier classifies Oracle-specific XA errors.
type OracleXAErrorClassifier struct{}

func (c *OracleXAErrorClassifier) IsAlreadyEnded(err error) bool {
	// TODO: check ORA-24756 (transaction does not exist) / ORA-24761 (rolled back)
	return false
}

// OracleXAConn implements XAResource for Oracle using the DBMS_XA PL/SQL package.
//
// Oracle does NOT support MySQL-style XA SQL statements.
// All operations go through PL/SQL anonymous blocks calling DBMS_XA:
//   - DBMS_XA.XA_START / XA_END / XA_PREPARE / XA_COMMIT / XA_ROLLBACK
//   - XID type: DBMS_XA_XID(formatid NUMBER, gtrid RAW(64), bqual RAW(64))
//   - Recovery: DBA_PENDING_TRANSACTIONS or DBMS_XA.XA_RECOVER
//   - Requires: GRANT EXECUTE ON DBMS_XA TO <user>
type OracleXAConn struct {
	driver.Conn
}

func (c *OracleXAConn) Start(ctx context.Context, xid string, flags int) error {
	log.Infof("xa branch start (oracle), xid %s", xid)
	// TODO: DBMS_XA.XA_START via PL/SQL block, map xid to DBMS_XA_XID
	return fmt.Errorf("Oracle XA start not yet implemented")
}

func (c *OracleXAConn) End(ctx context.Context, xid string, flags int) error {
	log.Infof("xa branch end (oracle), xid %s", xid)
	// TODO: DBMS_XA.XA_END via PL/SQL block
	return fmt.Errorf("Oracle XA end not yet implemented")
}

func (c *OracleXAConn) XAPrepare(ctx context.Context, xid string) error {
	log.Infof("xa branch prepare (oracle), xid %s", xid)
	// TODO: DBMS_XA.XA_PREPARE via PL/SQL block
	return fmt.Errorf("Oracle XA prepare not yet implemented")
}

func (c *OracleXAConn) Commit(ctx context.Context, xid string, onePhase bool) error {
	log.Infof("xa branch commit (oracle), xid %s, onePhase %v", xid, onePhase)
	// TODO: DBMS_XA.XA_COMMIT via PL/SQL block
	return fmt.Errorf("Oracle XA commit not yet implemented")
}

func (c *OracleXAConn) Rollback(ctx context.Context, xid string) error {
	log.Infof("xa branch rollback (oracle), xid %s", xid)
	// TODO: DBMS_XA.XA_ROLLBACK via PL/SQL block
	return fmt.Errorf("Oracle XA rollback not yet implemented")
}

func (c *OracleXAConn) Recover(ctx context.Context, flag int) ([]string, error) {
	if (flag & TMStartRScan) == 0 {
		return nil, nil
	}
	// TODO: SELECT globalid, branchid FROM DBA_PENDING_TRANSACTIONS
	return nil, fmt.Errorf("Oracle XA recover not yet implemented")
}

func (c *OracleXAConn) Forget(ctx context.Context, xid string) error {
	// TODO: DBMS_XA.XA_FORGET via PL/SQL block
	return fmt.Errorf("Oracle XA forget not yet implemented")
}

func (c *OracleXAConn) GetTransactionTimeout() time.Duration { return 0 }

func (c *OracleXAConn) IsSameRM(ctx context.Context, resource XAResource) bool { return false }

func (c *OracleXAConn) SetTransactionTimeout(duration time.Duration) bool { return false }
