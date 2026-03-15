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
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

type mockXAConnection struct {
	commitErr     error
	rollbackErr   error
	commitCalls   int
	rollbackCalls int
}

func (m *mockXAConnection) Commit(ctx context.Context) error {
	m.commitCalls++
	return m.commitErr
}

func (m *mockXAConnection) Rollback(ctx context.Context) error {
	m.rollbackCalls++
	return m.rollbackErr
}

func TestXATx_commitOnXA_NoGlobalTransaction(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = ""

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
		},
	}

	err := xaTx.commitOnXA()
	assert.NoError(t, err)
}

func TestXATx_commitOnXA_MissingXAConn(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.TransactionMode = types.XAMode

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  nil,
		},
	}

	err := xaTx.commitOnXA()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "xa transaction requires xaConn")
}

func TestXATx_commitOnXA_CommitSuccess_BranchNotRegistered(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.BranchID = 0
	tranCtx.TransactionMode = types.XAMode

	mockConn := &mockXAConnection{}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.commitOnXA()
	assert.NoError(t, err)
	assert.Equal(t, 1, mockConn.commitCalls)
}

func TestXATx_commitOnXA_CommitFailure_BranchNotRegistered(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.BranchID = 0
	tranCtx.TransactionMode = types.XAMode

	mockConn := &mockXAConnection{
		commitErr: errors.New("XA PREPARE failed"),
	}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.commitOnXA()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "XA PREPARE failed")
	assert.Equal(t, 1, mockConn.commitCalls)
}

func TestXATx_commitOnXA_CommitSuccess_BranchRegisteredReportsSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := mock.NewMockDataSourceManager(ctrl)
	mockMgr.SetBranchType(branch.BranchTypeXA)
	rm.GetRmCacheInstance().RegisterResourceManager(mockMgr)
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, param rm.BranchReportParam) error {
			assert.Equal(t, branch.BranchTypeXA, param.BranchType)
			assert.Equal(t, int64(123), param.BranchId)
			assert.EqualValues(t, branch.BranchStatusPhaseoneDone, param.Status)
			assert.Equal(t, "test-xid", param.Xid)
			return nil
		},
	).Times(1)

	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.BranchID = 123
	tranCtx.TransactionMode = types.XAMode

	mockConn := &mockXAConnection{}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.commitOnXA()
	assert.NoError(t, err)
	assert.Equal(t, 1, mockConn.commitCalls)
}

func TestXATx_commitOnXA_CommitFailure_BranchRegisteredReportsFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := mock.NewMockDataSourceManager(ctrl)
	mockMgr.SetBranchType(branch.BranchTypeXA)
	rm.GetRmCacheInstance().RegisterResourceManager(mockMgr)
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, param rm.BranchReportParam) error {
			assert.Equal(t, branch.BranchTypeXA, param.BranchType)
			assert.Equal(t, int64(123), param.BranchId)
			assert.EqualValues(t, branch.BranchStatusPhaseoneFailed, param.Status)
			assert.Equal(t, "test-xid", param.Xid)
			return nil
		},
	).Times(1)

	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.BranchID = 123
	tranCtx.TransactionMode = types.XAMode

	mockConn := &mockXAConnection{
		commitErr: errors.New("XA PREPARE failed"),
	}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.commitOnXA()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "XA PREPARE failed")
	assert.Equal(t, 1, mockConn.commitCalls)
}

func TestXATx_Rollback_NoGlobalTransaction(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = ""

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
		},
	}

	err := xaTx.Rollback()
	assert.NoError(t, err)
}

func TestXATx_Rollback_BranchNotRegistered(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.BranchID = 0
	tranCtx.TransactionMode = types.XAMode

	mockConn := &mockXAConnection{}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.Rollback()
	assert.NoError(t, err)
	assert.Equal(t, 1, mockConn.rollbackCalls)
}

func TestXATx_Rollback_WithXAConn(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.BranchID = 0
	tranCtx.TransactionMode = types.XAMode

	mockConn := &mockXAConnection{}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.Rollback()
	assert.NoError(t, err)
	assert.Equal(t, 1, mockConn.rollbackCalls)
}

func TestXATx_Rollback_XAConnError(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.BranchID = 0
	tranCtx.TransactionMode = types.XAMode

	mockConn := &mockXAConnection{
		rollbackErr: errors.New("XA ROLLBACK failed"),
	}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.Rollback()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "XA ROLLBACK failed")
	assert.Equal(t, 1, mockConn.rollbackCalls)
}

func TestXATx_Rollback_BranchRegisteredReportsFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := mock.NewMockDataSourceManager(ctrl)
	mockMgr.SetBranchType(branch.BranchTypeXA)
	rm.GetRmCacheInstance().RegisterResourceManager(mockMgr)
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, param rm.BranchReportParam) error {
			assert.Equal(t, branch.BranchTypeXA, param.BranchType)
			assert.Equal(t, int64(123), param.BranchId)
			assert.EqualValues(t, branch.BranchStatusPhaseoneFailed, param.Status)
			assert.Equal(t, "test-xid", param.Xid)
			return nil
		},
	).Times(1)

	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.BranchID = 123
	tranCtx.TransactionMode = types.XAMode

	mockConn := &mockXAConnection{}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.Rollback()
	assert.NoError(t, err)
	assert.Equal(t, 1, mockConn.rollbackCalls)
}

func TestXATx_Commit_CallsCommitOnXA(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.BranchID = 0
	tranCtx.TransactionMode = types.XAMode

	mockConn := &mockXAConnection{}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.Commit()
	assert.NoError(t, err)
	assert.Equal(t, 1, mockConn.commitCalls)
}

func TestTx_report_NoBranchID(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.BranchID = 0

	tx := &Tx{
		tranCtx: tranCtx,
	}

	err := tx.report(true)
	assert.NoError(t, err)
}

func TestXAConnection_Interface(t *testing.T) {
	var _ XAConnection = (*mockXAConnection)(nil)
}

func TestWithXAConn(t *testing.T) {
	mockConn := &mockXAConnection{}
	tx := &Tx{}

	opt := withXAConn(mockConn)
	opt(tx)

	assert.Equal(t, mockConn, tx.xaConn)
}

func TestXATx_Commit_BeforeCommitError(t *testing.T) {
	tranCtx := types.NewTxCtx()
	tranCtx.XID = "test-xid"
	tranCtx.TransactionMode = types.XAMode

	hookCalled := false
	RegisterTxHook(&mockTxHook{
		beforeCommit: func(tx *Tx) error {
			hookCalled = true
			return errors.New("hook error")
		},
	})
	defer CleanTxHooks()

	mockConn := &mockXAConnection{}

	xaTx := &XATx{
		tx: &Tx{
			tranCtx: tranCtx,
			xaConn:  mockConn,
		},
	}

	err := xaTx.Commit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hook error")
	assert.True(t, hookCalled)
	assert.Equal(t, 0, mockConn.commitCalls)
}

func TestGetStatus(t *testing.T) {
	assert.NotNil(t, getStatus(true))
	assert.NotNil(t, getStatus(false))
}
