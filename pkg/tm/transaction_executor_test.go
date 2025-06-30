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

package tm_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/tm/transaction/getty"
)

func TestMain(m *testing.M) {
	tm.SetGlobalTransactionManager(&getty.GettyGlobalTransactionManager{})
	code := m.Run()
	os.Exit(code)
}

func TestTransactionExecutorBegin(t *testing.T) {
	type Test struct {
		ctx                context.Context
		gc                 *tm.GtxConfig
		xid                string
		wantHasMock        bool
		wantMockTargetName string
		wantMockFunction   interface{}
		wantHasError       bool
		wantErrorString    string
		wantUseExist       bool
		wantBeginNew       bool
	}

	gts := []Test{
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: tm.NotSupported,
			},
			xid: "123456",
		},
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: tm.Supports,
			},
			xid:          "123456",
			wantUseExist: true,
		},
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: tm.RequiresNew,
			},
			xid:                "123456",
			wantHasMock:        true,
			wantMockTargetName: "Begin",
			wantMockFunction: func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, i time.Duration) error {
				tm.SetXID(ctx, "123456")
				return nil
			},
			wantBeginNew: true,
		},
		// use exist
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: tm.Required,
			},
			xid:          "123456",
			wantUseExist: true,
		},
		//Begin new
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: tm.Required,
			},
			wantHasMock:        true,
			wantMockTargetName: "Begin",
			wantMockFunction: func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, i time.Duration) error {
				tm.SetXID(ctx, "123456")
				return nil
			},
			wantBeginNew: true,
		},
		// has error
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: tm.Never,
			},
			xid:             "123456",
			wantHasError:    true,
			wantErrorString: "existing transaction found for transaction marked with pg 'never', xid = 123456",
		},
		// has not error
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: tm.Never,
			},
		},
		// use exist
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: tm.Mandatory,
			},
			xid:          "123456",
			wantUseExist: true,
		},
		// has error
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: tm.Mandatory,
			},
			wantHasError:    true,
			wantErrorString: "no existing transaction found for transaction marked with pg 'mandatory'",
		},
		// default case
		{
			ctx: context.Background(),
			gc: &tm.GtxConfig{
				Name:        "MockGtxName",
				Propagation: -1,
			},
			wantHasError:    true,
			wantErrorString: "not Supported Propagation:-1",
		},
	}

	for i, v := range gts {
		t.Logf("Case %v: %+v", i, v)
		var stub *gomonkey.Patches
		// set up stub
		if v.wantHasMock {
			stub = gomonkey.ApplyMethod(reflect.TypeOf(tm.GetGlobalTransactionManager()), v.wantMockTargetName, v.wantMockFunction)
		}

		v.ctx = tm.InitSeataContext(v.ctx)
		if v.xid != "" {
			tm.SetXID(v.ctx, v.xid)
		}
		err := tm.Begin(v.ctx, v.gc)

		if v.wantHasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		if v.wantBeginNew {
			assert.Equal(t, tm.Launcher, *tm.GetTxRole(v.ctx))
		}

		if v.wantUseExist {
			assert.Equal(t, tm.Participant, *tm.GetTxRole(v.ctx))
		}

		// rest up stub
		if v.wantHasMock {
			stub.Reset()
		}
	}
}

func TestTransactionExecutorCommit(t *testing.T) {
	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetTxRole(ctx, tm.Launcher)
	tm.SetTxStatus(ctx, message.GlobalStatusBegin)
	tm.SetXID(ctx, "")
	assert.Equal(t, "Commit xid should not be empty", tm.CommitOrRollback(ctx, true).Error())
}

func TestTransactionExecurotRollback(t *testing.T) {
	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetTxRole(ctx, tm.Launcher)
	tm.SetTxStatus(ctx, message.GlobalStatusBegin)
	tm.SetXID(ctx, "")
	errActual := tm.CommitOrRollback(ctx, false)
	assert.Equal(t, "Rollback xid should not be empty", errActual.Error())
}

func TestCommitOrRollback(t *testing.T) {
	type Test struct {
		ctx                context.Context
		tx                 tm.GlobalTransaction
		ok                 bool
		wantHasMock        bool
		wantMockTargetName string
		wantMockFunction   interface{}
		wantHasError       bool
		wantErrorString    string
	}

	gts := []Test{
		{
			ctx: context.Background(),
			tx: tm.GlobalTransaction{
				TxRole: tm.UnKnow,
			},
			wantHasError:    true,
			wantErrorString: "global transaction role is UnKnow.",
		},
		//ok with nil
		{
			ctx: context.Background(),
			tx: tm.GlobalTransaction{
				TxRole: tm.Launcher,
			},
			ok:                 true,
			wantHasMock:        true,
			wantMockTargetName: "Commit",
			wantMockFunction: func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, gtr *tm.GlobalTransaction) error {
				return nil
			},
		},
		//ok with error
		{
			ctx: context.Background(),
			tx: tm.GlobalTransaction{
				TxRole: tm.Launcher,
			},
			ok:                 true,
			wantHasMock:        true,
			wantMockTargetName: "Commit",
			wantMockFunction: func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, gtr *tm.GlobalTransaction) error {
				return errors.New("Mock error")
			},
			wantHasError:    true,
			wantErrorString: "Mock error",
		},
		// false with nil
		{
			ctx: context.Background(),
			tx: tm.GlobalTransaction{
				TxRole: tm.Launcher,
			},
			ok:                 false,
			wantHasMock:        true,
			wantMockTargetName: "Rollback",
			wantMockFunction: func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, gtr *tm.GlobalTransaction) error {
				return nil
			},
		},
		// false with error
		{
			ctx: context.Background(),
			tx: tm.GlobalTransaction{
				TxRole: tm.Launcher,
			},
			ok:                 false,
			wantHasMock:        true,
			wantMockTargetName: "Rollback",
			wantMockFunction: func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, gtr *tm.GlobalTransaction) error {
				return errors.New("Mock error")
			},
			wantHasError:    true,
			wantErrorString: "Mock error",
		},
		{
			ctx: context.Background(),
			tx: tm.GlobalTransaction{
				TxRole: tm.Participant,
			},
		},
	}

	for i, v := range gts {
		t.Logf("Case %v: %+v", i, v)
		v.ctx = tm.InitSeataContext(v.ctx)
		tm.SetTx(v.ctx, &v.tx)
		var stub *gomonkey.Patches
		if v.wantHasMock {
			stub = gomonkey.ApplyMethod(reflect.TypeOf(tm.GetGlobalTransactionManager()), v.wantMockTargetName, v.wantMockFunction)
		}

		err := tm.CommitOrRollback(v.ctx, v.ok)

		if v.wantHasError {
			assert.Equal(t, v.wantErrorString, err.Error())
		} else {
			assert.Nil(t, err)
		}

		if v.wantHasMock {
			stub.Reset()
		}
	}
}

func TestClearTxConf(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())

	tm.SetTx(ctx, &tm.GlobalTransaction{
		Xid:      "123456",
		TxName:   "MockTxName",
		TxStatus: message.GlobalStatusBegin,
		TxRole:   tm.Launcher,
	})

	tm.ClearTxConf(ctx)

	assert.Equal(t, "123456", tm.GetXID(ctx))
	assert.Equal(t, tm.UnKnow, *tm.GetTxRole(ctx))
	assert.Equal(t, message.GlobalStatusUnKnown, *tm.GetTxStatus(ctx))
	assert.Equal(t, "", tm.GetTxName(ctx))
}

func TestUseExistGtx(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "123456")
	tm.UseExistGtx(ctx, &tm.GtxConfig{
		Name: "useExistGtxMock",
	})
	assert.Equal(t, tm.Participant, *tm.GetTxRole(ctx))
	assert.Equal(t, message.GlobalStatusBegin, *tm.GetTxStatus(ctx))
}

func TestBeginNewGtx(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	g := &tm.GtxConfig{
		Name: "beginNewGtxMock",
	}
	// case return nil
	gomonkey.ApplyMethod(reflect.TypeOf(tm.GetGlobalTransactionManager()), "Begin",
		func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, timeout time.Duration) error {
			return nil
		})
	assert.Nil(t, tm.BeginNewGtx(ctx, g))
	assert.Equal(t, tm.Launcher, *tm.GetTxRole(ctx))
	assert.Equal(t, g.Name, tm.GetTxName(ctx))
	assert.Equal(t, message.GlobalStatusBegin, *tm.GetTxStatus(ctx))

	// case return error
	err := errors.New("Mock Error")
	gomonkey.ApplyMethod(reflect.TypeOf(tm.GetGlobalTransactionManager()), "Begin",
		func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, timeout time.Duration) error {
			return err
		})
	assert.Error(t, tm.BeginNewGtx(ctx, g))
}

func TestWithGlobalTx(t *testing.T) {
	callbackError := func(ctx context.Context) error {
		return errors.New("mock callback error")
	}
	callbackNil := func(ctx context.Context) error {
		return nil
	}

	type testCase struct {
		GtxConfig              *tm.GtxConfig
		occurError             bool
		timeoutErr             bool
		errMessage             string
		callbackErr            bool
		mockBeginFunc          interface{}
		mockBeginTarget        interface{}
		mockSecondPhaseFunc    interface{}
		mockSecondPhaseTarget  interface{}
		mockRollbackTargetName string
		mockRollbackFunc       interface{}
		mockTimeoutTarget      interface{}
		mockTimeoutFunc        interface{}
		secondErr              bool
		callback               tm.CallbackWithCtx
	}

	gts := []testCase{
		// case TxName is nil
		{
			GtxConfig:  &tm.GtxConfig{},
			occurError: true,
			errMessage: "global transaction name is required.",
		},

		// case tm.GtxConfig is nil
		{
			occurError: true,
			errMessage: "global transaction config info is required.",
		},

		// case mock begin return error
		{
			GtxConfig: &tm.GtxConfig{
				Name: "Mocktm.GtxConfig",
			},
			occurError:      true,
			errMessage:      "mock begin",
			mockBeginTarget: tm.Begin,
			mockBeginFunc: func(ctx context.Context, gc *tm.GtxConfig) error {
				return errors.New("mock begin")
			},
		},

		// case callback return error
		{
			GtxConfig: &tm.GtxConfig{
				Name: "Mocktm.GtxConfig",
			},
			callbackErr:     true,
			callback:        callbackError,
			mockBeginTarget: tm.Begin,
			mockBeginFunc: func(ctx context.Context, gc *tm.GtxConfig) error {
				return nil
			},
		},

		// case callback return nil
		{
			GtxConfig: &tm.GtxConfig{
				Name: "Mocktm.GtxConfig",
			},
			mockBeginTarget: tm.Begin,
			mockBeginFunc: func(ctx context.Context, gc *tm.GtxConfig) error {
				tm.SetXID(ctx, "123456")
				return nil
			},
			callback:              callbackNil,
			mockSecondPhaseTarget: tm.CommitOrRollback,
			mockSecondPhaseFunc: func(ctx context.Context, s bool) error {
				return nil
			},
		},

		// case second mock error
		{
			GtxConfig: &tm.GtxConfig{
				Name: "Mocktm.GtxConfig",
			},
			mockBeginTarget: tm.Begin,
			mockBeginFunc: func(ctx context.Context, gc *tm.GtxConfig) error {
				tm.SetXID(ctx, "123456")
				return nil
			},
			callback:              callbackNil,
			mockSecondPhaseTarget: tm.CommitOrRollback,
			mockSecondPhaseFunc: func(ctx context.Context, s bool) error {
				return errors.New("second error mock")
			},
			secondErr: true,
		},

		// case tm detected a timeout and executed rollback successfully.
		{
			GtxConfig: &tm.GtxConfig{
				Name:    "MockGtxConfig",
				Timeout: time.Second * 30,
			},
			timeoutErr:      false,
			mockBeginTarget: tm.Begin,
			mockBeginFunc: func(ctx context.Context, gc *tm.GtxConfig) error {
				tm.SetXID(ctx, "123456")
				tm.SetTxRole(ctx, tm.Launcher)
				return nil
			},
			callback:          callbackNil,
			mockTimeoutTarget: tm.IsTimeout,
			mockTimeoutFunc: func(ctx context.Context) bool {
				return true
			},
			mockRollbackTargetName: "Rollback",
			mockRollbackFunc: func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, gtr *tm.GlobalTransaction) error {
				return nil
			},
		},
		// tm detected a timeout but rollback threw an exception.
		{
			GtxConfig: &tm.GtxConfig{
				Name:    "MockGtxConfig",
				Timeout: time.Second * 30,
			},
			timeoutErr:      true,
			errMessage:      "tm detected a timeout but rollback threw an exception",
			mockBeginTarget: tm.Begin,
			mockBeginFunc: func(ctx context.Context, gc *tm.GtxConfig) error {
				tm.SetXID(ctx, "123456")
				tm.SetTxRole(ctx, tm.Launcher)
				return nil
			},
			callback:          callbackNil,
			mockTimeoutTarget: tm.IsTimeout,
			mockTimeoutFunc: func(ctx context.Context) bool {
				return true
			},
			mockRollbackTargetName: "Rollback",
			mockRollbackFunc: func(_ *getty.GettyGlobalTransactionManager, ctx context.Context, gtr *tm.GlobalTransaction) error {
				return fmt.Errorf("tm detected a timeout but rollback threw an exception")
			},
		},
	}

	for i, v := range gts {
		t.Logf("Case %v: %+v", i, v)
		var beginStub *gomonkey.Patches
		var secondStub *gomonkey.Patches
		var timeoutStub *gomonkey.Patches
		var rollbackStub *gomonkey.Patches
		if v.mockBeginTarget != nil {
			beginStub = gomonkey.ApplyFunc(v.mockBeginTarget, v.mockBeginFunc)
		}
		if v.mockSecondPhaseTarget != nil {
			secondStub = gomonkey.ApplyFunc(v.mockSecondPhaseTarget, v.mockSecondPhaseFunc)
		}
		if v.mockTimeoutTarget != nil {
			timeoutStub = gomonkey.ApplyFunc(v.mockTimeoutTarget, v.mockTimeoutFunc)
		}
		if v.mockRollbackTargetName != "" {
			rollbackStub = gomonkey.ApplyMethod(reflect.TypeOf(tm.GetGlobalTransactionManager()), v.mockRollbackTargetName, v.mockRollbackFunc)
		}

		ctx := context.Background()
		err := tm.WithGlobalTx(ctx, v.GtxConfig, v.callback)

		if v.occurError {
			assert.Equal(t, v.errMessage, err.Error())
		}

		if v.callbackErr {
			assert.NotNil(t, err)
		}

		if v.secondErr {
			assert.NotNil(t, err)
		}

		if v.timeoutErr {
			assert.Regexp(t, v.errMessage, err.Error())
		}

		if v.mockBeginTarget != nil {
			beginStub.Reset()
		}

		if v.mockSecondPhaseTarget != nil {
			secondStub.Reset()
		}

		if v.mockTimeoutTarget != nil {
			timeoutStub.Reset()
		}

		if v.mockRollbackTargetName != "" {
			rollbackStub.Reset()
		}

	}
}
