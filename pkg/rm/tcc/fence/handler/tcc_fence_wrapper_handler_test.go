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

package handler

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/v2/pkg/rm/tcc/fence/store/db/model"
	"seata.apache.org/seata-go/v2/pkg/tm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

// mockTCCFenceStore is a mock implementation of TCCFenceStore for testing
type mockTCCFenceStore struct {
	mu                               sync.RWMutex
	insertFunc                       func(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error
	queryFunc                        func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error)
	updateFunc                       func(tx *sql.Tx, xid string, branchId int64, oldStatus enum.FenceStatus, newStatus enum.FenceStatus) error
	deleteFunc                       func(tx *sql.Tx, xid string, branchId int64) error
	deleteMultipleFunc               func(tx *sql.Tx, identity []model.FenceLogIdentity) error
	deleteTCCFenceDOByMdfDateFunc    func(tx *sql.Tx, datetime time.Time, limit int32) (int64, error)
	queryTCCFenceLogIdentityByMdDate func(tx *sql.Tx, datetime time.Time) ([]model.FenceLogIdentity, error)
}

func (m *mockTCCFenceStore) InsertTCCFenceDO(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.insertFunc != nil {
		return m.insertFunc(tx, tccFenceDo)
	}
	return nil
}

func (m *mockTCCFenceStore) QueryTCCFenceDO(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.queryFunc != nil {
		return m.queryFunc(tx, xid, branchId)
	}
	return nil, nil
}

func (m *mockTCCFenceStore) UpdateTCCFenceDO(tx *sql.Tx, xid string, branchId int64, oldStatus enum.FenceStatus, newStatus enum.FenceStatus) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.updateFunc != nil {
		return m.updateFunc(tx, xid, branchId, oldStatus, newStatus)
	}
	return nil
}

func (m *mockTCCFenceStore) DeleteTCCFenceDO(tx *sql.Tx, xid string, branchId int64) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.deleteFunc != nil {
		return m.deleteFunc(tx, xid, branchId)
	}
	return nil
}

func (m *mockTCCFenceStore) DeleteMultipleTCCFenceLogIdentity(tx *sql.Tx, identity []model.FenceLogIdentity) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.deleteMultipleFunc != nil {
		return m.deleteMultipleFunc(tx, identity)
	}
	return nil
}

func (m *mockTCCFenceStore) DeleteTCCFenceDOByMdfDate(tx *sql.Tx, datetime time.Time, limit int32) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.deleteTCCFenceDOByMdfDateFunc != nil {
		return m.deleteTCCFenceDOByMdfDateFunc(tx, datetime, limit)
	}
	return 0, nil
}

func (m *mockTCCFenceStore) QueryTCCFenceLogIdentityByMdDate(tx *sql.Tx, datetime time.Time) ([]model.FenceLogIdentity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.queryTCCFenceLogIdentityByMdDate != nil {
		return m.queryTCCFenceLogIdentityByMdDate(tx, datetime)
	}
	return nil, nil
}

func (m *mockTCCFenceStore) SetLogTableName(logTable string) {
	// No-op for mock
}

func createTestContext(xid string, branchId int64, actionName string) context.Context {
	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	bac := &tm.BusinessActionContext{
		Xid:        xid,
		BranchId:   branchId,
		ActionName: actionName,
	}
	tm.SetBusinessActionContext(ctx, bac)
	return ctx
}

func TestGetFenceHandler(t *testing.T) {
	log.Init()

	handler1 := GetFenceHandler()
	assert.NotNil(t, handler1)

	handler2 := GetFenceHandler()
	assert.Equal(t, handler1, handler2, "GetFenceHandler should return the same instance")
}

func TestInitCleanPeriod(t *testing.T) {
	log.Init()

	handler := GetFenceHandler()
	testDuration := 10 * time.Minute

	handler.InitCleanPeriod(testDuration)
	assert.Equal(t, testDuration, currentCleanInterval())

	// Reset to default for other tests
	cleanIntervalNanos.Store(int64(5 * time.Minute))
}

func TestPrepareFence_Success(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		insertFunc: func(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error {
			assert.Equal(t, "test-xid", tccFenceDo.Xid)
			assert.Equal(t, int64(123), tccFenceDo.BranchId)
			assert.Equal(t, "test-action", tccFenceDo.ActionName)
			assert.Equal(t, enum.StatusTried, tccFenceDo.Status)
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.PrepareFence(ctx, tx)
	assert.NoError(t, err)
}

func TestPrepareFence_DuplicateEntry(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mysqlErr := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
	mockDao := &mockTCCFenceStore{
		insertFunc: func(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error {
			return mysqlErr
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
		logQueue:    make(chan *model.FenceLogIdentity, maxQueueSize),
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.PrepareFence(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert tcc fence record errors")
}

func TestPrepareFence_InsertError(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		insertFunc: func(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error {
			return errors.New("database error")
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.PrepareFence(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert tcc fence record errors")
}

func TestCommitFence_Success(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return &model.TCCFenceDO{
				Xid:      xid,
				BranchId: branchId,
				Status:   enum.StatusTried,
			}, nil
		},
		updateFunc: func(tx *sql.Tx, xid string, branchId int64, oldStatus enum.FenceStatus, newStatus enum.FenceStatus) error {
			assert.Equal(t, enum.StatusTried, oldStatus)
			assert.Equal(t, enum.StatusCommitted, newStatus)
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.CommitFence(ctx, tx)
	assert.NoError(t, err)
}

func TestCommitFence_QueryError(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return nil, errors.New("query error")
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.CommitFence(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commit fence method failed")
}

func TestCommitFence_RecordNotExists(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return nil, nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.CommitFence(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tcc fence record not exists")
}

func TestCommitFence_AlreadyCommitted(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return &model.TCCFenceDO{
				Xid:      xid,
				BranchId: branchId,
				Status:   enum.StatusCommitted,
			}, nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.CommitFence(ctx, tx)
	assert.NoError(t, err)
}

func TestCommitFence_UnexpectedStatus_Rollbacked(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return &model.TCCFenceDO{
				Xid:      xid,
				BranchId: branchId,
				Status:   enum.StatusRollbacked,
			}, nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.CommitFence(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch transaction status is unexpected")
}

func TestCommitFence_UnexpectedStatus_Suspended(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return &model.TCCFenceDO{
				Xid:      xid,
				BranchId: branchId,
				Status:   enum.StatusSuspended,
			}, nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.CommitFence(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch transaction status is unexpected")
}

func TestRollbackFence_Success(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return &model.TCCFenceDO{
				Xid:      xid,
				BranchId: branchId,
				Status:   enum.StatusTried,
			}, nil
		},
		updateFunc: func(tx *sql.Tx, xid string, branchId int64, oldStatus enum.FenceStatus, newStatus enum.FenceStatus) error {
			assert.Equal(t, enum.StatusTried, oldStatus)
			assert.Equal(t, enum.StatusRollbacked, newStatus)
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.RollbackFence(ctx, tx)
	assert.NoError(t, err)
}

func TestRollbackFence_QueryError(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return nil, errors.New("query error")
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.RollbackFence(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rollback fence method failed")
}

func TestRollbackFence_RecordNotExists_InsertSuspended(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return nil, nil
		},
		insertFunc: func(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error {
			assert.Equal(t, enum.StatusSuspended, tccFenceDo.Status)
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.RollbackFence(ctx, tx)
	assert.NoError(t, err)
}

func TestRollbackFence_RecordNotExists_InsertError(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return nil, nil
		},
		insertFunc: func(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error {
			return errors.New("insert error")
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.RollbackFence(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert tcc fence record errors")
}

func TestRollbackFence_AlreadyRollbacked(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return &model.TCCFenceDO{
				Xid:      xid,
				BranchId: branchId,
				Status:   enum.StatusRollbacked,
			}, nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.RollbackFence(ctx, tx)
	assert.NoError(t, err)
}

func TestRollbackFence_AlreadySuspended(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return &model.TCCFenceDO{
				Xid:      xid,
				BranchId: branchId,
				Status:   enum.StatusSuspended,
			}, nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.RollbackFence(ctx, tx)
	assert.NoError(t, err)
}

func TestRollbackFence_UnexpectedStatus_Committed(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		queryFunc: func(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
			return &model.TCCFenceDO{
				Xid:      xid,
				BranchId: branchId,
				Status:   enum.StatusCommitted,
			}, nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	ctx := createTestContext("test-xid", 123, "test-action")
	err = handler.RollbackFence(ctx, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch transaction status is unexpected")
}

func TestInsertTCCFenceLog(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		insertFunc: func(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error {
			assert.Equal(t, "test-xid", tccFenceDo.Xid)
			assert.Equal(t, int64(456), tccFenceDo.BranchId)
			assert.Equal(t, "test-action", tccFenceDo.ActionName)
			assert.Equal(t, enum.StatusTried, tccFenceDo.Status)
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	err = handler.insertTCCFenceLog(tx, "test-xid", 456, "test-action", enum.StatusTried)
	assert.NoError(t, err)
}

func TestUpdateFenceStatus(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mockDao := &mockTCCFenceStore{
		updateFunc: func(tx *sql.Tx, xid string, branchId int64, oldStatus enum.FenceStatus, newStatus enum.FenceStatus) error {
			assert.Equal(t, "test-xid", xid)
			assert.Equal(t, int64(789), branchId)
			assert.Equal(t, enum.StatusTried, oldStatus)
			assert.Equal(t, enum.StatusCommitted, newStatus)
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	err = handler.updateFenceStatus(tx, "test-xid", 789, enum.StatusCommitted)
	assert.NoError(t, err)
}

func TestDeleteBatchFence_Success(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	batch := []model.FenceLogIdentity{
		{Xid: "xid1", BranchId: 1},
		{Xid: "xid2", BranchId: 2},
	}

	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			assert.Equal(t, batch, identity)
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	err = handler.deleteBatchFence(tx, batch)
	assert.NoError(t, err)
}

func TestDeleteBatchFence_Error(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	batch := []model.FenceLogIdentity{
		{Xid: "xid1", BranchId: 1},
	}

	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			return errors.New("delete error")
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
	}

	err = handler.deleteBatchFence(tx, batch)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete batch fence log failed")
}

func TestPushCleanChannel(t *testing.T) {
	log.Init()

	handler := &tccFenceWrapperHandler{
		logQueue: make(chan *model.FenceLogIdentity, maxQueueSize),
	}

	handler.pushCleanChannel("test-xid", 123)

	select {
	case fli := <-handler.logQueue:
		assert.Equal(t, "test-xid", fli.Xid)
		assert.Equal(t, int64(123), fli.BranchId)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected to receive from logQueue")
	}
}

func TestEnqueueFenceLogIdentities_PreserveDistinctValues(t *testing.T) {
	log.Init()

	handler := &tccFenceWrapperHandler{
		logQueue: make(chan *model.FenceLogIdentity, 3),
	}
	identities := []model.FenceLogIdentity{
		{Xid: "xid1", BranchId: 1},
		{Xid: "xid2", BranchId: 2},
		{Xid: "xid3", BranchId: 3},
	}

	handler.enqueueFenceLogIdentities(identities)

	consumed := []model.FenceLogIdentity{
		*<-handler.logQueue,
		*<-handler.logQueue,
		*<-handler.logQueue,
	}

	assert.Equal(t, identities, consumed)
}

func TestPushCleanChannel_FullQueue(t *testing.T) {
	log.Init()

	handler := &tccFenceWrapperHandler{
		logQueue: make(chan *model.FenceLogIdentity, 1),
	}

	// Second pushCleanChannel starts drainCacheTask; stop it before the next test mutates cleanInterval.
	defer handler.DestroyLogCleanChannel()
	// Fill the queue
	handler.pushCleanChannel("xid1", 1)

	// This should add to cache instead
	handler.pushCleanChannel("xid2", 2)

	// Verify queue has first item
	select {
	case fli := <-handler.logQueue:
		assert.Equal(t, "xid1", fli.Xid)
	default:
		t.Fatal("Expected queue to have item")
	}

	// Verify cache has second item
	assert.Equal(t, 1, handler.logCache.Len())
}

func TestDestroyLogCleanChannel(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	mock.ExpectClose()

	handler := &tccFenceWrapperHandler{
		logQueue: make(chan *model.FenceLogIdentity, maxQueueSize),
		db:       db,
	}

	handler.DestroyLogCleanChannel()

	// Verify channel is closed
	_, ok := <-handler.logQueue
	assert.False(t, ok, "logQueue should be closed")

	// Verify db is nil
	assert.Nil(t, handler.db)
}

func TestDestroyLogCleanChannel_OnlyOnce(t *testing.T) {
	log.Init()

	handler := &tccFenceWrapperHandler{
		logQueue: make(chan *model.FenceLogIdentity, maxQueueSize),
	}

	// Call multiple times
	handler.DestroyLogCleanChannel()
	handler.DestroyLogCleanChannel()
	handler.DestroyLogCleanChannel()

	// Should only close once without panic
	assert.Nil(t, handler.db)
}

func TestTraversalCleanChannel(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Expect Begin and Commit for batch delete
	mock.ExpectBegin()
	mock.ExpectCommit()

	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			// Verify we're deleting the expected batch
			assert.Equal(t, channelDelete, len(identity))
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
		logQueue:    make(chan *model.FenceLogIdentity, maxQueueSize),
	}

	// Use WaitGroup to ensure goroutine completes
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the goroutine
	go func() {
		defer wg.Done()
		handler.traversalCleanChannel(db)
	}()

	// Push exactly channelDelete items to trigger batch delete
	for i := 0; i < channelDelete; i++ {
		handler.logQueue <- &model.FenceLogIdentity{
			Xid:      "test-xid",
			BranchId: int64(i),
		}
	}

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	// Close the channel to stop the goroutine
	close(handler.logQueue)

	// Wait for goroutine to finish
	wg.Wait()

	// Verify expectations
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTraversalCleanChannel_RemainingBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Expect Begin and Commit for final batch
	mock.ExpectBegin()
	mock.ExpectCommit()

	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			// Verify we're deleting the remaining items
			assert.Equal(t, 3, len(identity))
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
		logQueue:    make(chan *model.FenceLogIdentity, maxQueueSize),
	}

	// Use WaitGroup to ensure goroutine completes
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the goroutine
	go func() {
		defer wg.Done()
		handler.traversalCleanChannel(db)
	}()

	// Push less than channelDelete items
	for i := 0; i < 3; i++ {
		handler.logQueue <- &model.FenceLogIdentity{
			Xid:      "test-xid",
			BranchId: int64(i),
		}
	}

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	// Close the channel to stop the goroutine and trigger final batch
	close(handler.logQueue)

	// Wait for goroutine to finish
	wg.Wait()

	// Verify expectations
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTraversalCleanChannel_DeleteError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Expect Begin but delete will fail
	mock.ExpectBegin()

	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			return errors.New("delete failed")
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
		logQueue:    make(chan *model.FenceLogIdentity, maxQueueSize),
	}

	// Use WaitGroup to ensure goroutine completes
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the goroutine
	go func() {
		defer wg.Done()
		handler.traversalCleanChannel(db)
	}()

	// Push channelDelete items to trigger batch delete
	for i := 0; i < channelDelete; i++ {
		handler.logQueue <- &model.FenceLogIdentity{
			Xid:      "test-xid",
			BranchId: int64(i),
		}
	}

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	// Close the channel
	close(handler.logQueue)

	// Wait for goroutine to finish
	wg.Wait()
}

func TestInitLogCleanChannel(t *testing.T) {
	// Create a test DSN for sqlmock
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer db.Close()

	// We can't easily test the actual goroutines, but we can verify
	// the function doesn't panic and initializes the handler
	handler := &tccFenceWrapperHandler{
		tccFenceDao: &mockTCCFenceStore{},
	}

	// Test with invalid DSN (won't connect but won't panic)
	handler.InitLogCleanChannel("invalid-dsn")

	// Verify db was attempted to be set
	handler.dbMutex.RLock()
	dbSet := handler.db != nil
	handler.dbMutex.RUnlock()

	// With invalid DSN, db should still be nil
	assert.False(t, dbSet)

	// Clean up
	if err := mock.ExpectationsWereMet(); err != nil {
		// It's okay if expectations aren't met with invalid DSN
		t.Logf("Expected behavior with invalid DSN: %v", err)
	}
}

func TestInitLogCleanChannel_EmptyDSN(t *testing.T) {
	// Wait a bit to ensure previous tests' goroutines have finished
	time.Sleep(300 * time.Millisecond)

	log.Init()

	handler := &tccFenceWrapperHandler{
		tccFenceDao: &mockTCCFenceStore{},
	}

	// Test with empty DSN - sql.Open still returns a non-nil DB even with empty DSN
	// but it may not be usable
	handler.InitLogCleanChannel("")
	defer handler.DestroyLogCleanChannel()

	// Verify db was attempted to be set (sql.Open returns non-nil DB even with empty DSN)
	handler.dbMutex.RLock()
	// DB object is created but may not be functional
	assert.NotNil(t, handler.db)
	handler.dbMutex.RUnlock()
}

func TestConstants(t *testing.T) {
	// Verify constants are set as expected
	assert.Equal(t, 500, maxQueueSize)
	assert.Equal(t, 5, channelDelete)
	assert.Equal(t, 24*time.Hour, cleanExpired)
	assert.Equal(t, 5*time.Minute, currentCleanInterval())
}

func TestDrainCacheTask(t *testing.T) {
	log.Init()

	oldInterval := currentCleanInterval()
	cleanIntervalNanos.Store(int64(20 * time.Millisecond))
	defer cleanIntervalNanos.Store(int64(oldInterval))

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// drainCacheTask should execute one delete batch with Begin + Commit.
	mock.ExpectBegin()
	mock.ExpectCommit()

	deleted := make(chan struct{}, 1)

	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			for _, it := range identity {
				if it.Xid == "xid-cache" && it.BranchId == 2 {
					select {
					case deleted <- struct{}{}:
					default:
					}
				}
			}
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
		logQueue:    make(chan *model.FenceLogIdentity, 1), // Intentionally small queue to trigger the default branch.
		db:          db,
	}

	// Fill the queue first.
	handler.pushCleanChannel("xid-queue", 1)
	// Push one more item so it falls back to cache and starts drainCacheTask via logCacheOnce.Do(...).
	handler.pushCleanChannel("xid-cache", 2)

	time.Sleep(100 * time.Millisecond)
	assert.NotNil(t, handler.stopDrainCache)

	select {
	case <-deleted:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("expected drainCacheTask to delete cached identity")
	}

	assert.NoError(t, mock.ExpectationsWereMet())

	// Cleanup to avoid goroutine leaks.
	handler.DestroyLogCleanChannel()
}

func TestDrainCacheTask_DBNil(t *testing.T) {
	log.Init()

	oldInterval := currentCleanInterval()
	cleanIntervalNanos.Store(int64(20 * time.Millisecond))
	defer cleanIntervalNanos.Store(int64(oldInterval))

	handler := &tccFenceWrapperHandler{
		tccFenceDao:    &mockTCCFenceStore{},
		logQueue:       make(chan *model.FenceLogIdentity, 1),
		stopDrainCache: make(chan struct{}),
		// Keep db as nil intentionally.
	}

	handler.cacheMutex.Lock()
	handler.logCache.PushBack(&fenceLogCacheEntry{identity: model.FenceLogIdentity{Xid: "xid-nil", BranchId: 9}})
	handler.cacheMutex.Unlock()

	go handler.drainCacheTask()

	time.Sleep(100 * time.Millisecond)

	handler.cacheMutex.Lock()
	cacheLen := handler.logCache.Len()
	handler.cacheMutex.Unlock()

	assert.Equal(t, 1, cacheLen)

	handler.DestroyLogCleanChannel()
}

func TestDrainCacheTask_DeleteError_RequeuesToCache(t *testing.T) {
	log.Init()

	oldInterval := currentCleanInterval()
	cleanIntervalNanos.Store(int64(20 * time.Millisecond))
	defer cleanIntervalNanos.Store(int64(oldInterval))

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Each drain tick: Begin -> deleteBatchFence (error) -> Rollback. Allow several ticks before Destroy stops the goroutine.
	for i := 0; i < 10; i++ {
		mock.ExpectBegin()
		mock.ExpectRollback()
	}

	var deleteCalls atomic.Int32
	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			deleteCalls.Add(1)
			return errors.New("simulated delete failure")
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
		logQueue:    make(chan *model.FenceLogIdentity, 1),
		db:          db,
	}

	handler.pushCleanChannel("xid-queue", 1)
	handler.pushCleanChannel("xid-cache", 2)

	// Require both in one poll: (1) Len()>=1 alone is true before the first drain (second push left an item).
	// (2) deleteCalls>=1 alone can race the next ticker tick (cache emptied again before we read Len).
	assert.Eventually(t, func() bool {
		if deleteCalls.Load() < 1 {
			return false
		}
		handler.cacheMutex.Lock()
		n := handler.logCache.Len()
		handler.cacheMutex.Unlock()
		return n >= 1
	}, 2*time.Second, 10*time.Millisecond,
		"delete must have run and logCache must still hold requeued identities (stable window)")

	handler.DestroyLogCleanChannel()
}

func TestDrainCacheTask_BeginError_RequeuesToCache(t *testing.T) {
	log.Init()

	oldInterval := currentCleanInterval()
	cleanIntervalNanos.Store(int64(20 * time.Millisecond))
	defer cleanIntervalNanos.Store(int64(oldInterval))

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin().WillReturnError(errors.New("simulated begin failure"))
	for i := 0; i < 15; i++ {
		mock.ExpectBegin()
		mock.ExpectCommit()
	}

	var deleteCalls atomic.Int32
	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			deleteCalls.Add(1)
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
		logQueue:    make(chan *model.FenceLogIdentity, 1),
		db:          db,
	}

	handler.pushCleanChannel("xid-queue", 1)
	handler.pushCleanChannel("xid-cache", 2)

	// First tick: Begin fails (requeue). deleteCalls stays 0. Later tick: delete succeeds and cache drains.
	assert.Eventually(t, func() bool {
		if deleteCalls.Load() < 1 {
			return false
		}
		handler.cacheMutex.Lock()
		n := handler.logCache.Len()
		handler.cacheMutex.Unlock()
		return n == 0
	}, 2*time.Second, 10*time.Millisecond,
		"successful delete after begin failure should empty requeued cache")

	handler.DestroyLogCleanChannel()
}

func TestDrainCacheTask_CommitError_RequeuesToCache(t *testing.T) {
	log.Init()

	oldInterval := currentCleanInterval()
	cleanIntervalNanos.Store(int64(20 * time.Millisecond))
	defer cleanIntervalNanos.Store(int64(oldInterval))

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("simulated commit failure"))
	for i := 0; i < 15; i++ {
		mock.ExpectBegin()
		mock.ExpectCommit()
	}

	var deleteCalls atomic.Int32
	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			deleteCalls.Add(1)
			return nil
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
		logQueue:    make(chan *model.FenceLogIdentity, 1),
		db:          db,
	}

	handler.pushCleanChannel("xid-queue", 1)
	handler.pushCleanChannel("xid-cache", 2)

	// First tick: delete ok, Commit fails (requeue), deleteCalls==1 and Len==1. Later tick finishes cleanup.
	assert.Eventually(t, func() bool {
		if deleteCalls.Load() < 1 {
			return false
		}
		handler.cacheMutex.Lock()
		n := handler.logCache.Len()
		handler.cacheMutex.Unlock()
		return n == 0
	}, 2*time.Second, 10*time.Millisecond,
		"successful commit after prior commit failure should empty requeued cache")

	handler.DestroyLogCleanChannel()
}

func TestDrainCacheTask_MaxRetries_DropsFromCache(t *testing.T) {
	log.Init()

	oldInterval := currentCleanInterval()
	cleanIntervalNanos.Store(int64(20 * time.Millisecond))
	defer cleanIntervalNanos.Store(int64(oldInterval))

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	for i := 0; i < 12; i++ {
		mock.ExpectBegin()
		mock.ExpectRollback()
	}

	var deleteCalls atomic.Int32
	mockDao := &mockTCCFenceStore{
		deleteMultipleFunc: func(tx *sql.Tx, identity []model.FenceLogIdentity) error {
			deleteCalls.Add(1)
			return errors.New("always fail delete")
		},
	}

	handler := &tccFenceWrapperHandler{
		tccFenceDao: mockDao,
		logQueue:    make(chan *model.FenceLogIdentity, 1),
		db:          db,
	}

	handler.pushCleanChannel("xid-queue", 1)
	handler.pushCleanChannel("xid-cache", 2)

	// maxFenceLogCacheRetries=3: requeue with counts 1,2,3 then drop on fourth failure (fourth successful Begin + failed delete).
	assert.Eventually(t, func() bool {
		if deleteCalls.Load() < int32(maxFenceLogCacheRetries+1) {
			return false
		}
		handler.cacheMutex.Lock()
		n := handler.logCache.Len()
		handler.cacheMutex.Unlock()
		return n == 0
	}, 4*time.Second, 10*time.Millisecond,
		"after max requeues, failed drain should drop identities from logCache")

	handler.DestroyLogCleanChannel()
}

func TestDrainCacheTask_StopsOnDestroy(t *testing.T) {
	log.Init()

	oldInterval := currentCleanInterval()
	cleanIntervalNanos.Store(int64(20 * time.Millisecond))
	defer cleanIntervalNanos.Store(int64(oldInterval))

	waitDrainGoroutine := func(t *testing.T, wg *sync.WaitGroup) {
		t.Helper()
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("drainCacheTask goroutine did not exit after DestroyLogCleanChannel")
		}
	}

	t.Run("emptyLogCache", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		handler := &tccFenceWrapperHandler{
			tccFenceDao: &mockTCCFenceStore{},
			logQueue:    make(chan *model.FenceLogIdentity, maxQueueSize),
		}
		handler.logCacheOnce.Do(func() {
			handler.stopDrainCache = make(chan struct{})
			go func() {
				defer wg.Done()
				handler.drainCacheTask()
			}()
		})

		time.Sleep(30 * time.Millisecond)

		assert.NotPanics(t, func() {
			handler.DestroyLogCleanChannel()
		})

		waitDrainGoroutine(t, &wg)
	})

	t.Run("withPendingCacheEntries", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()
		for i := 0; i < 10; i++ {
			mock.ExpectBegin()
			mock.ExpectCommit()
		}

		handler := &tccFenceWrapperHandler{
			tccFenceDao: &mockTCCFenceStore{},
			logQueue:    make(chan *model.FenceLogIdentity, maxQueueSize),
			db:          db,
		}
		handler.logCacheOnce.Do(func() {
			handler.stopDrainCache = make(chan struct{})
			go func() {
				defer wg.Done()
				handler.drainCacheTask()
			}()
		})

		handler.cacheMutex.Lock()
		handler.logCache.PushBack(&fenceLogCacheEntry{identity: model.FenceLogIdentity{Xid: "xid-pending", BranchId: 1}})
		handler.cacheMutex.Unlock()

		time.Sleep(50 * time.Millisecond)

		assert.NotPanics(t, func() {
			handler.DestroyLogCleanChannel()
		})

		waitDrainGoroutine(t, &wg)
	})
}
