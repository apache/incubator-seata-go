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

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluele/gcache"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

type oracleXAIntegrationManager struct {
	resources sync.Map
	nextID    int64
	lastID    int64
	reports   []rm.BranchReportParam
}

func (m *oracleXAIntegrationManager) BranchCommit(context.Context, rm.BranchResource) (branch.BranchStatus, error) {
	return branch.BranchStatusPhasetwoCommitted, nil
}

func (m *oracleXAIntegrationManager) BranchRollback(context.Context, rm.BranchResource) (branch.BranchStatus, error) {
	return branch.BranchStatusPhasetwoRollbacked, nil
}

func (m *oracleXAIntegrationManager) BranchRegister(_ context.Context, _ rm.BranchRegisterParam) (int64, error) {
	id := atomic.AddInt64(&m.nextID, 1)
	m.lastID = id
	return id, nil
}

func (m *oracleXAIntegrationManager) BranchReport(_ context.Context, param rm.BranchReportParam) error {
	m.reports = append(m.reports, param)
	return nil
}

func (m *oracleXAIntegrationManager) LockQuery(context.Context, rm.LockQueryParam) (bool, error) {
	return true, nil
}

func (m *oracleXAIntegrationManager) RegisterResource(resource rm.Resource) error {
	m.resources.Store(resource.GetResourceId(), resource)
	return nil
}

func (m *oracleXAIntegrationManager) UnregisterResource(rm.Resource) error {
	return nil
}

func (m *oracleXAIntegrationManager) GetCachedResources() *sync.Map {
	return &m.resources
}

func (m *oracleXAIntegrationManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeXA
}

func (m *oracleXAIntegrationManager) CreateTableMetaCache(context.Context, string, types.DBType, *sql.DB) (datasource.TableMetaCache, error) {
	return nil, nil
}

func TestOracleXAIntegration_SeataXAOracleFirstAndSecondPhase(t *testing.T) {
	dsn := os.Getenv("ORACLE_XA_DSN")
	if dsn == "" {
		t.Skip("set ORACLE_XA_DSN to run Oracle XA integration test")
	}

	branchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()
	xaConnTimeout = time.Hour
	t.Cleanup(func() {
		xaConnTimeout = 0
	})

	ctx := context.Background()
	manager := &oracleXAIntegrationManager{}
	rm.GetRmCacheInstance().RegisterResourceManager(manager)

	directDB, err := sql.Open("oracle", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = directDB.Close()
	})
	require.NoError(t, directDB.PingContext(ctx))
	tableName := fmt.Sprintf("SXAPROBE_%d", time.Now().UnixNano()%1_000_000_000)
	t.Cleanup(func() {
		_, _ = directDB.ExecContext(context.Background(), "DROP TABLE "+tableName+" PURGE")
	})
	_, err = directDB.ExecContext(ctx, "CREATE TABLE "+tableName+" (id NUMBER PRIMARY KEY, val VARCHAR2(32))")
	require.NoError(t, err)

	db, err := sql.Open(SeataXAOracleDriver, dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.NoError(t, db.PingContext(ctx))

	globalXID := fmt.Sprintf("oracle-framework-%d", time.Now().UnixNano()%1_000_000_000)
	globalCtx := tm.InitSeataContext(context.Background())
	tm.SetXID(globalCtx, globalXID)

	_, err = db.ExecContext(globalCtx, "INSERT INTO "+tableName+" (id, val) VALUES (10, 'prepared')")
	require.NoError(t, err)
	require.NotEmpty(t, manager.reports)
	require.Equal(t, branch.BranchStatus(branch.BranchStatusPhaseoneDone), manager.reports[len(manager.reports)-1].Status)

	var count int
	require.NoError(t, directDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+tableName+" WHERE id = 10").Scan(&count))
	require.Equal(t, 0, count)

	_, err = db.ExecContext(ctx, "INSERT INTO "+tableName+" (id, val) VALUES (20, 'autocommit')")
	require.NoError(t, err)
	require.NoError(t, directDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM "+tableName+" WHERE id = 20 AND val = 'autocommit'").Scan(&count))
	require.Equal(t, 1, count)

	resourceAny, ok := manager.resources.Load(dsn)
	require.True(t, ok)
	resource := resourceAny.(*DBResource)
	xaID := XaIdBuild(globalXID, uint64(manager.lastID))
	xaConn, err := resource.ConnectionForXA(ctx, xaID)
	require.NoError(t, err)
	require.NoError(t, xaConn.XaCommit(ctx, xaID))
	require.NoError(t, xaConn.Close())

	require.NoError(t, directDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM "+tableName+" WHERE id = 10 AND val = 'prepared'").Scan(&count))
	require.Equal(t, 1, count)
}

func TestOracleXAIntegration_SeataXAOracleRollback(t *testing.T) {
	dsn := os.Getenv("ORACLE_XA_DSN")
	if dsn == "" {
		t.Skip("set ORACLE_XA_DSN to run Oracle XA integration test")
	}

	branchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()
	xaConnTimeout = time.Hour
	t.Cleanup(func() {
		branchStatusCache = nil
		xaConnTimeout = 0
	})

	ctx := context.Background()
	manager := &oracleXAIntegrationManager{}
	rm.GetRmCacheInstance().RegisterResourceManager(manager)

	directDB, err := sql.Open("oracle", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = directDB.Close()
	})
	require.NoError(t, directDB.PingContext(ctx))
	tableName := fmt.Sprintf("SXARB_%d", time.Now().UnixNano()%1_000_000_000)
	t.Cleanup(func() {
		_, _ = directDB.ExecContext(context.Background(), "DROP TABLE "+tableName+" PURGE")
	})
	_, err = directDB.ExecContext(ctx, "CREATE TABLE "+tableName+" (id NUMBER PRIMARY KEY, val VARCHAR2(32))")
	require.NoError(t, err)

	db, err := sql.Open(SeataXAOracleDriver, dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.NoError(t, db.PingContext(ctx))

	globalCtx := tm.InitSeataContext(context.Background())
	tm.SetXID(globalCtx, fmt.Sprintf("oracle-rollback-%d", time.Now().UnixNano()%1_000_000_000))

	tx, err := db.BeginTx(globalCtx, &sql.TxOptions{})
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), "INSERT INTO "+tableName+" (id, val) VALUES (10, 'rolled-back')")
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	var count int
	require.NoError(t, directDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM "+tableName+" WHERE id = 10").Scan(&count))
	require.Equal(t, 0, count)

	_, err = db.ExecContext(ctx, "INSERT INTO "+tableName+" (id, val) VALUES (20, 'autocommit')")
	require.NoError(t, err)
	require.NoError(t, directDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM "+tableName+" WHERE id = 20 AND val = 'autocommit'").Scan(&count))
	require.Equal(t, 1, count)
}

func TestOracleXAIntegration_SeataXAOracleReadOnlySkipsSecondPhaseCommit(t *testing.T) {
	dsn := os.Getenv("ORACLE_XA_DSN")
	if dsn == "" {
		t.Skip("set ORACLE_XA_DSN to run Oracle XA integration test")
	}

	branchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()
	xaConnTimeout = time.Hour
	t.Cleanup(func() {
		branchStatusCache = nil
		xaConnTimeout = 0
	})

	ctx := context.Background()
	manager := &oracleXAIntegrationManager{}
	rm.GetRmCacheInstance().RegisterResourceManager(manager)

	directDB, err := sql.Open("oracle", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = directDB.Close()
	})
	require.NoError(t, directDB.PingContext(ctx))
	tableName := fmt.Sprintf("SXARO_%d", time.Now().UnixNano()%1_000_000_000)
	t.Cleanup(func() {
		_, _ = directDB.ExecContext(context.Background(), "DROP TABLE "+tableName+" PURGE")
	})
	_, err = directDB.ExecContext(ctx, "CREATE TABLE "+tableName+" (id NUMBER PRIMARY KEY, val VARCHAR2(32))")
	require.NoError(t, err)

	db, err := sql.Open(SeataXAOracleDriver, dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.NoError(t, db.PingContext(ctx))

	globalXID := fmt.Sprintf("oracle-readonly-%d", time.Now().UnixNano()%1_000_000_000)
	globalCtx := tm.InitSeataContext(context.Background())
	tm.SetXID(globalCtx, globalXID)

	var value int
	require.NoError(t, db.QueryRowContext(globalCtx, "SELECT 1 FROM DUAL").Scan(&value))
	require.Equal(t, 1, value)
	require.NotEmpty(t, manager.reports)
	require.Equal(t, branch.BranchStatus(branch.BranchStatusPhaseoneDone), manager.reports[len(manager.reports)-1].Status)

	resourceAny, ok := manager.resources.Load(dsn)
	require.True(t, ok)
	resource := resourceAny.(*DBResource)
	xaID := XaIdBuild(globalXID, uint64(manager.lastID))

	xaManager := &XAResourceManager{}
	xaManager.resourceCache.Store(dsn, resource)
	status, err := xaManager.BranchCommit(ctx, rm.BranchResource{
		ResourceId: dsn,
		Xid:        xaID.GetGlobalXid(),
		BranchId:   int64(xaID.GetBranchId()),
	})
	require.NoError(t, err)
	require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitted), status)

	_, err = db.ExecContext(ctx, "INSERT INTO "+tableName+" (id, val) VALUES (10, 'autocommit')")
	require.NoError(t, err)
	require.NoError(t, directDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM "+tableName+" WHERE id = 10 AND val = 'autocommit'").Scan(&value))
	require.Equal(t, 1, value)
}
