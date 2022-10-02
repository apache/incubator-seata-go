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
	"database/sql/driver"
	"sync/atomic"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/mock"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/stretchr/testify/assert"
)

type mockSQLInterceptor struct {
	before func(ctx context.Context, execCtx *types.ExecContext)
	after  func(ctx context.Context, execCtx *types.ExecContext)
}

func (mi *mockSQLInterceptor) Type() types.SQLType {
	return types.SQLTypeUnknown
}

// Before
func (mi *mockSQLInterceptor) Before(ctx context.Context, execCtx *types.ExecContext) error {
	if mi.before != nil {
		mi.before(ctx, execCtx)
	}
	return nil
}

// After
func (mi *mockSQLInterceptor) After(ctx context.Context, execCtx *types.ExecContext) error {
	if mi.after != nil {
		mi.after(ctx, execCtx)
	}
	return nil
}

type mockTxHook struct {
	beforeCommit   func(tx *Tx)
	beforeRollback func(tx *Tx)
}

// BeforeCommit
func (mi *mockTxHook) BeforeCommit(tx *Tx) {
	if mi.beforeCommit != nil {
		mi.beforeCommit(tx)
	}
}

// BeforeRollback
func (mi *mockTxHook) BeforeRollback(tx *Tx) {
	if mi.beforeRollback != nil {
		mi.beforeRollback(tx)
	}
}

func baseMoclConn(mockConn *mock.MockTestDriverConn) {
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)
	mockConn.EXPECT().Exec(gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)
	mockConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
	mockConn.EXPECT().Close().AnyTimes().Return(nil)
}

func TestXAConn_ExecContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(t, ctrl)
	_ = mockMgr

	db, err := sql.Open("seata-xa-mysql", "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	_ = initMockXaConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Commit().AnyTimes().Return(nil)
		mockTx.EXPECT().Rollback().AnyTimes().Return(nil)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().Begin().AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).AnyTimes().Return(mockTx, nil)
		baseMoclConn(mockConn)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	mi := &mockSQLInterceptor{}
	ti := &mockTxHook{}

	exec.CleanCommonHook()
	CleanTxHooks()
	exec.RegisCommonHook(mi)
	RegisterTxHook(ti)

	t.Run("have xid", func(t *testing.T) {
		ctx := tm.InitSeataContext(context.Background())
		tm.SetXID(ctx, uuid.New().String())
		t.Logf("set xid=%s", tm.GetXID(ctx))

		before := func(_ context.Context, execCtx *types.ExecContext) {
			t.Logf("on exec xid=%s", execCtx.TxCtx.XaID)
			assert.Equal(t, tm.GetXID(ctx), execCtx.TxCtx.XaID)
			assert.Equal(t, types.XAMode, execCtx.TxCtx.TransType)
		}
		mi.before = before

		var comitCnt int32
		beforeCommit := func(tx *Tx) {
			atomic.AddInt32(&comitCnt, 1)
			assert.Equal(t, tx.ctx.TransType, types.XAMode)
		}
		ti.beforeCommit = beforeCommit

		conn, err := db.Conn(context.Background())
		assert.NoError(t, err)

		_, err = conn.ExecContext(ctx, "SELECT 1")
		assert.NoError(t, err)
		_, err = db.ExecContext(ctx, "SELECT 1")
		assert.NoError(t, err)

		assert.Equal(t, int32(2), atomic.LoadInt32(&comitCnt))
	})

	t.Run("not xid", func(t *testing.T) {
		before := func(_ context.Context, execCtx *types.ExecContext) {
			assert.Equal(t, "", execCtx.TxCtx.XaID)
			assert.Equal(t, types.Local, execCtx.TxCtx.TransType)
		}
		mi.before = before

		var comitCnt int32
		beforeCommit := func(tx *Tx) {
			atomic.AddInt32(&comitCnt, 1)
		}
		ti.beforeCommit = beforeCommit

		conn, err := db.Conn(context.Background())
		assert.NoError(t, err)

		_, err = conn.ExecContext(context.Background(), "SELECT 1")
		assert.NoError(t, err)
		_, err = db.ExecContext(context.Background(), "SELECT 1")
		assert.NoError(t, err)

		_, err = db.Exec("SELECT 1")
		assert.NoError(t, err)

		assert.Equal(t, int32(0), atomic.LoadInt32(&comitCnt))
	})
}
