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

package client

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	mysqlundo "seata.apache.org/seata-go/v2/pkg/datasource/sql/undo/mysql"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

func TestProcess_InvalidBodyType(t *testing.T) {
	p := &rmDeleteUndoLogProcessor{}
	err := p.Process(context.Background(), message.RpcMessage{
		Body: "not a UndoLogDeleteRequest",
	})
	assert.Error(t, err)
}

func TestProcess_NonATBranchType_Skipped(t *testing.T) {
	p := &rmDeleteUndoLogProcessor{}
	for _, bt := range []branch.BranchType{branch.BranchTypeXA, branch.BranchTypeTCC, branch.BranchTypeSAGA} {
		err := p.Process(context.Background(), message.RpcMessage{
			Body: message.UndoLogDeleteRequest{
				ResourceId: "any-resource",
				SaveDays:   7,
				BranchType: bt,
			},
		})
		assert.NoError(t, err, "branch type %v should be skipped silently", bt)
	}
}

func TestProcess_ResourceNotFound_ReturnsNil(t *testing.T) {
	p := &rmDeleteUndoLogProcessor{}
	err := p.Process(context.Background(), message.RpcMessage{
		Body: message.UndoLogDeleteRequest{
			ResourceId: "jdbc:mysql://not-registered:3306/db",
			SaveDays:   7,
			BranchType: branch.BranchTypeAT,
		},
	})
	// no AT resource manager / resource not managed by this client is a normal
	// skip (the request is broadcast), so Process returns nil rather than an error
	assert.NoError(t, err, "unmanaged resource should be skipped, not treated as an error")
}

type mockDBResource struct {
	db     *sql.DB
	dbType types.DBType
}

func (m *mockDBResource) GetDB() *sql.DB          { return m.db }
func (m *mockDBResource) GetDbType() types.DBType { return m.dbType }

// fakeATResourceManager is a minimal AT resource manager that only exposes a cached
// resource, so the processor can reach the delete path in tests.
type fakeATResourceManager struct {
	rm.ResourceManager
	cache *sync.Map
}

func (f *fakeATResourceManager) GetBranchType() branch.BranchType { return branch.BranchTypeAT }
func (f *fakeATResourceManager) GetCachedResources() *sync.Map    { return f.cache }

// TestProcess_RealError_Propagates verifies that a real internal failure while
// deleting (here: no undo log manager for the resource's db type) is surfaced by
// Process instead of being swallowed, while unmanaged resources are still skipped.
func TestProcess_RealError_Propagates(t *testing.T) {
	undo.RegisterUndoLogManager(mysqlundo.NewUndoLogManager())

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	cache := &sync.Map{}
	cache.Store("res-real-err", &mockDBResource{db: db, dbType: types.DBTypeMySQL})
	rm.GetRmCacheInstance().RegisterResourceManager(&fakeATResourceManager{cache: cache})

	mock.ExpectQuery("SELECT 1 FROM undo_log LIMIT 1").
		WillReturnError(errors.New("check table failed"))

	p := &rmDeleteUndoLogProcessor{}
	err = p.Process(context.Background(), message.RpcMessage{
		Body: message.UndoLogDeleteRequest{
			ResourceId: "res-real-err",
			SaveDays:   7,
			BranchType: branch.BranchTypeAT,
		},
	})
	assert.Error(t, err, "a real internal failure must propagate through Process")
	assert.Contains(t, err.Error(), "check undo log table")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProcess_EndToEnd_DeletesExpiredUndoLog(t *testing.T) {
	undo.RegisterUndoLogManager(mysqlundo.NewUndoLogManager())

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	const resourceID = "jdbc:mysql://127.0.0.1:3306/seata"
	cache := &sync.Map{}
	cache.Store(resourceID, &mockDBResource{db: db, dbType: types.DBTypeMySQL})
	rm.GetRmCacheInstance().RegisterResourceManager(&fakeATResourceManager{cache: cache})

	mock.ExpectQuery("SELECT 1 FROM undo_log LIMIT 1").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, defaultDeleteBatchSize))
	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 10))

	p := &rmDeleteUndoLogProcessor{}
	err = p.Process(context.Background(), message.RpcMessage{
		Body: message.UndoLogDeleteRequest{
			ResourceId: resourceID,
			SaveDays:   7,
			BranchType: branch.BranchTypeAT,
		},
	})
	assert.NoError(t, err, "end-to-end delete should succeed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProcess_NonPositiveSaveDays_RejectedAtEntry(t *testing.T) {
	p := &rmDeleteUndoLogProcessor{}
	for _, saveDays := range []int16{0, -1, -32768} {
		err := p.Process(context.Background(), message.RpcMessage{
			Body: message.UndoLogDeleteRequest{
				ResourceId: "jdbc:mysql://127.0.0.1:3306/seata-savedays-guard",
				SaveDays:   saveDays,
				BranchType: branch.BranchTypeAT,
			},
		})
		assert.Error(t, err, "saveDays=%d must be rejected", saveDays)
	}
}

func TestProcess_ResourceNotInCache_Skipped(t *testing.T) {
	rm.GetRmCacheInstance().RegisterResourceManager(&fakeATResourceManager{cache: &sync.Map{}})

	p := &rmDeleteUndoLogProcessor{}
	err := p.Process(context.Background(), message.RpcMessage{
		Body: message.UndoLogDeleteRequest{
			ResourceId: "jdbc:mysql://another-client:3306/seata",
			SaveDays:   7,
			BranchType: branch.BranchTypeAT,
		},
	})
	assert.NoError(t, err, "unmanaged resource on a broadcast request should be skipped, not error")
}

// TestProcess_UnsupportedDbType_Skipped verifies that a non-MySQL AT resource is
// skipped before any SQL is issued: the DELETE ... LIMIT cleanup is a MySQL
// dialect extension, and Postgres (which has a registered undo log manager, so
// the earlier manager lookup would not stop it) rejects the statement on every
// TC cleanup cycle. The processor must warn and skip instead.
func TestProcess_UnsupportedDbType_Skipped(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	const resourceID = "jdbc:postgresql://127.0.0.1:5432/seata"
	cache := &sync.Map{}
	cache.Store(resourceID, &mockDBResource{db: db, dbType: types.DBTypePostgreSQL})
	rm.GetRmCacheInstance().RegisterResourceManager(&fakeATResourceManager{cache: cache})

	p := &rmDeleteUndoLogProcessor{}
	err = p.Process(context.Background(), message.RpcMessage{
		Body: message.UndoLogDeleteRequest{
			ResourceId: resourceID,
			SaveDays:   7,
			BranchType: branch.BranchTypeAT,
		},
	})
	assert.NoError(t, err, "non-MySQL resource should be skipped, not error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchDeleteByLogCreated_DeletesRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	before := time.Now().AddDate(0, 0, -7)

	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1000))
	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 50))

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	p := &rmDeleteUndoLogProcessor{}
	err = p.batchDeleteByLogCreated(context.Background(), conn, before)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchDeleteByLogCreated_CustomTableName(t *testing.T) {
	original := undo.UndoConfig.LogTable
	undo.UndoConfig.LogTable = "my_undo_log"
	defer func() { undo.UndoConfig.LogTable = original }()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("DELETE FROM my_undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	p := &rmDeleteUndoLogProcessor{}
	err = p.batchDeleteByLogCreated(context.Background(), conn, time.Now())
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchDeleteByLogCreated_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(driver.ErrBadConn)

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	p := &rmDeleteUndoLogProcessor{}
	err = p.batchDeleteByLogCreated(context.Background(), conn, time.Now())
	assert.Error(t, err)
}

// TestBatchDeleteByLogCreated_RowsAffectedError verifies that an error from
// RowsAffected() (e.g. driver returns -1 affected on a successful exec) is
// surfaced instead of being swallowed. Before the fix, affected=-1 would
// trigger an early `break` (since -1 < batchSize), silently leaving old
// undo_log rows un-deleted while reporting success.
func TestBatchDeleteByLogCreated_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("driver does not support RowsAffected")))

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	p := &rmDeleteUndoLogProcessor{}
	err = p.batchDeleteByLogCreated(context.Background(), conn, time.Now())
	assert.Error(t, err, "RowsAffected error must propagate, not be swallowed")
	assert.Contains(t, err.Error(), "rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func BenchmarkProcess_NonAT(b *testing.B) {
	p := &rmDeleteUndoLogProcessor{}
	msg := message.RpcMessage{
		Body: message.UndoLogDeleteRequest{
			ResourceId: "any",
			SaveDays:   7,
			BranchType: branch.BranchTypeXA,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Process(context.Background(), msg)
	}
}

func TestBatchDeleteByLogCreated_CustomBatchSize(t *testing.T) {
	original := undo.UndoConfig.DeleteBatchSize
	undo.UndoConfig.DeleteBatchSize = 2 // 小批次，方便验证循环
	defer func() { undo.UndoConfig.DeleteBatchSize = original }()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// 第一批删了 2 行（等于 batchSize），继续
	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	// 第二批删了 1 行（小于 batchSize），退出
	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	p := &rmDeleteUndoLogProcessor{}
	err = p.batchDeleteByLogCreated(context.Background(), conn, time.Now())
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchDeleteByLogCreated_RoundLimitStopsDraining(t *testing.T) {
	original := undo.UndoConfig.DeleteBatchSize
	undo.UndoConfig.DeleteBatchSize = 2
	defer func() { undo.UndoConfig.DeleteBatchSize = original }()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	for i := 0; i < maxDeleteBatchRounds; i++ {
		mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))
	}

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	p := &rmDeleteUndoLogProcessor{}
	err = p.batchDeleteByLogCreated(context.Background(), conn, time.Now())
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
