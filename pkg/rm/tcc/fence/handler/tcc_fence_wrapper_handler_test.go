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
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/store/db/model"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
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
	assert.Equal(t, testDuration, cleanInterval)

	// Reset to default for other tests
	cleanInterval = 5 * time.Minute
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

func TestPushCleanChannel_FullQueue(t *testing.T) {
	log.Init()

	handler := &tccFenceWrapperHandler{
		logQueue: make(chan *model.FenceLogIdentity, 1),
	}

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
	log.Init()

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

	// Start the goroutine
	go handler.traversalCleanChannel(db)

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

	// Give it time to finish
	time.Sleep(100 * time.Millisecond)

	// Verify expectations
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTraversalCleanChannel_RemainingBatch(t *testing.T) {
	log.Init()

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

	// Start the goroutine
	go handler.traversalCleanChannel(db)

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

	// Give it time to finish
	time.Sleep(100 * time.Millisecond)

	// Verify expectations
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTraversalCleanChannel_DeleteError(t *testing.T) {
	log.Init()

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

	// Start the goroutine
	go handler.traversalCleanChannel(db)

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

	// Give it time to finish
	time.Sleep(100 * time.Millisecond)
}

func TestInitLogCleanChannel(t *testing.T) {
	// Wait a bit to ensure previous tests' goroutines have finished
	time.Sleep(300 * time.Millisecond)

	log.Init()

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

	// Verify db was attempted to be set (sql.Open returns non-nil DB even with empty DSN)
	handler.dbMutex.RLock()
	defer handler.dbMutex.RUnlock()
	// DB object is created but may not be functional
	assert.NotNil(t, handler.db)
}

func TestConstants(t *testing.T) {
	// Verify constants are set as expected
	assert.Equal(t, 500, maxQueueSize)
	assert.Equal(t, 5, channelDelete)
	assert.Equal(t, 24*time.Hour, cleanExpired)
	assert.Equal(t, 5*time.Minute, cleanInterval)
}
