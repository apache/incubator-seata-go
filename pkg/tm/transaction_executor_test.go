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

package tm

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/protocol/message"
)

func TestTransactionExecutorBegin(t *testing.T) {
	type Test struct {
		ctx                context.Context
		gc                 *GtxConfig
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
			gc: &GtxConfig{
				Name:        "MockGtxName",
				Propagation: NotSupported,
			},
			xid: "123456",
		},
		{
			ctx: context.Background(),
			gc: &GtxConfig{
				Name:        "MockGtxName",
				Propagation: Supports,
			},
			xid:          "123456",
			wantUseExist: true,
		},
		{
			ctx: context.Background(),
			gc: &GtxConfig{
				Name:        "MockGtxName",
				Propagation: RequiresNew,
			},
			xid:                "123456",
			wantHasMock:        true,
			wantMockTargetName: "Begin",
			wantMockFunction: func(_ *GlobalTransactionManager, ctx context.Context, i time.Duration) error {
				SetXID(ctx, "123456")
				return nil
			},
			wantBeginNew: true,
		},
		// use exist
		{
			ctx: context.Background(),
			gc: &GtxConfig{
				Name:        "MockGtxName",
				Propagation: Required,
			},
			xid:          "123456",
			wantUseExist: true,
		},
		//Begin new
		{
			ctx: context.Background(),
			gc: &GtxConfig{
				Name:        "MockGtxName",
				Propagation: Required,
			},
			wantHasMock:        true,
			wantMockTargetName: "Begin",
			wantMockFunction: func(_ *GlobalTransactionManager, ctx context.Context, i time.Duration) error {
				SetXID(ctx, "123456")
				return nil
			},
			wantBeginNew: true,
		},
		// has error
		{
			ctx: context.Background(),
			gc: &GtxConfig{
				Name:        "MockGtxName",
				Propagation: Never,
			},
			xid:             "123456",
			wantHasError:    true,
			wantErrorString: "existing transaction found for transaction marked with pg 'never', xid = 123456",
		},
		// has not error
		{
			ctx: context.Background(),
			gc: &GtxConfig{
				Name:        "MockGtxName",
				Propagation: Never,
			},
		},
		// use exist
		{
			ctx: context.Background(),
			gc: &GtxConfig{
				Name:        "MockGtxName",
				Propagation: Mandatory,
			},
			xid:          "123456",
			wantUseExist: true,
		},
		// has error
		{
			ctx: context.Background(),
			gc: &GtxConfig{
				Name:        "MockGtxName",
				Propagation: Mandatory,
			},
			wantHasError:    true,
			wantErrorString: "no existing transaction found for transaction marked with pg 'mandatory'",
		},
		// default case
		{
			ctx: context.Background(),
			gc: &GtxConfig{
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
			stub = gomonkey.ApplyMethod(reflect.TypeOf(GetGlobalTransactionManager()), v.wantMockTargetName, v.wantMockFunction)
		}

		v.ctx = InitSeataContext(v.ctx)
		if v.xid != "" {
			SetXID(v.ctx, v.xid)
		}
		err := begin(v.ctx, v.gc)

		if v.wantHasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		if v.wantBeginNew {
			assert.Equal(t, Launcher, *GetTxRole(v.ctx))
		}

		if v.wantUseExist {
			assert.Equal(t, Participant, *GetTxRole(v.ctx))
		}

		// rest up stub
		if v.wantHasMock {
			stub.Reset()
		}
	}
}

func TestTransactionExecutorCommit(t *testing.T) {
	ctx := context.Background()
	ctx = InitSeataContext(ctx)
	SetTxRole(ctx, Launcher)
	SetTxStatus(ctx, message.GlobalStatusBegin)
	SetXID(ctx, "")
	assert.Equal(t, "Commit xid should not be empty", commitOrRollback(ctx, true).Error())
}

func TestTransactionExecurotRollback(t *testing.T) {
	ctx := context.Background()
	ctx = InitSeataContext(ctx)
	SetTxRole(ctx, Launcher)
	SetTxStatus(ctx, message.GlobalStatusBegin)
	SetXID(ctx, "")
	errActual := commitOrRollback(ctx, false)
	assert.Equal(t, "Rollback xid should not be empty", errActual.Error())
}

func TestCommitOrRollback(t *testing.T) {
	type Test struct {
		ctx                context.Context
		tx                 GlobalTransaction
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
			tx: GlobalTransaction{
				TxRole: UnKnow,
			},
			wantHasError:    true,
			wantErrorString: "global transaction role is UnKnow.",
		},
		//ok with nil
		{
			ctx: context.Background(),
			tx: GlobalTransaction{
				TxRole: Launcher,
			},
			ok:                 true,
			wantHasMock:        true,
			wantMockTargetName: "Commit",
			wantMockFunction: func(_ *GlobalTransactionManager, ctx context.Context, gtr *GlobalTransaction) error {
				return nil
			},
		},
		//ok with error
		{
			ctx: context.Background(),
			tx: GlobalTransaction{
				TxRole: Launcher,
			},
			ok:                 true,
			wantHasMock:        true,
			wantMockTargetName: "Commit",
			wantMockFunction: func(_ *GlobalTransactionManager, ctx context.Context, gtr *GlobalTransaction) error {
				return errors.New("Mock error")
			},
			wantHasError:    true,
			wantErrorString: "Mock error",
		},
		// false with nil
		{
			ctx: context.Background(),
			tx: GlobalTransaction{
				TxRole: Launcher,
			},
			ok:                 false,
			wantHasMock:        true,
			wantMockTargetName: "Rollback",
			wantMockFunction: func(_ *GlobalTransactionManager, ctx context.Context, gtr *GlobalTransaction) error {
				return nil
			},
		},
		// false with error
		{
			ctx: context.Background(),
			tx: GlobalTransaction{
				TxRole: Launcher,
			},
			ok:                 false,
			wantHasMock:        true,
			wantMockTargetName: "Rollback",
			wantMockFunction: func(_ *GlobalTransactionManager, ctx context.Context, gtr *GlobalTransaction) error {
				return errors.New("Mock error")
			},
			wantHasError:    true,
			wantErrorString: "Mock error",
		},
		{
			ctx: context.Background(),
			tx: GlobalTransaction{
				TxRole: Participant,
			},
		},
	}

	for i, v := range gts {
		t.Logf("Case %v: %+v", i, v)
		v.ctx = InitSeataContext(v.ctx)
		SetTx(v.ctx, &v.tx)
		var stub *gomonkey.Patches
		if v.wantHasMock {
			stub = gomonkey.ApplyMethod(reflect.TypeOf(GetGlobalTransactionManager()), v.wantMockTargetName, v.wantMockFunction)
		}

		err := commitOrRollback(v.ctx, v.ok)

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
	ctx := InitSeataContext(context.Background())

	SetTx(ctx, &GlobalTransaction{
		Xid:      "123456",
		TxName:   "MockTxName",
		TxStatus: message.GlobalStatusBegin,
		TxRole:   Launcher,
	})

	clearTxConf(ctx)

	assert.Equal(t, "123456", GetXID(ctx))
	assert.Equal(t, UnKnow, *GetTxRole(ctx))
	assert.Equal(t, message.GlobalStatusUnKnown, *GetTxStatus(ctx))
	assert.Equal(t, "", GetTxName(ctx))
}

func TestUseExistGtx(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	SetXID(ctx, "123456")
	useExistGtx(ctx, &GtxConfig{
		Name: "useExistGtxMock",
	})
	assert.Equal(t, Participant, *GetTxRole(ctx))
	assert.Equal(t, message.GlobalStatusBegin, *GetTxStatus(ctx))
}

func TestBeginNewGtx(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	g := &GtxConfig{
		Name: "beginNewGtxMock",
	}
	// case return nil
	gomonkey.ApplyMethod(reflect.TypeOf(GetGlobalTransactionManager()), "Begin",
		func(_ *GlobalTransactionManager, ctx context.Context, timeout time.Duration) error {
			return nil
		})
	assert.Nil(t, beginNewGtx(ctx, g))
	assert.Equal(t, Launcher, *GetTxRole(ctx))
	assert.Equal(t, g.Name, GetTxName(ctx))
	assert.Equal(t, message.GlobalStatusBegin, *GetTxStatus(ctx))

	// case return error
	err := errors.New("Mock Exception")
	gomonkey.ApplyMethod(reflect.TypeOf(GetGlobalTransactionManager()), "Begin",
		func(_ *GlobalTransactionManager, ctx context.Context, timeout time.Duration) error {
			return err
		})
	assert.Error(t, beginNewGtx(ctx, g))
}

func TestWithGlobalTx(t *testing.T) {
	callbackError := func(ctx context.Context) error {
		return errors.New("mock callback error")
	}
	callbackNil := func(ctx context.Context) error {
		return nil
	}

	type testCase struct {
		GtxConfig             *GtxConfig
		occurError            bool
		errMessage            string
		callbackErr           bool
		mockBeginFunc         interface{}
		mockBeginTarget       interface{}
		mockSecondPhaseFunc   interface{}
		mockSecondPhaseTarget interface{}
		secondErr             bool
		callback              CallbackWithCtx
	}

	gts := []testCase{
		// case TxName is nil
		{
			GtxConfig:  &GtxConfig{},
			occurError: true,
			errMessage: "global transaction name is required.",
		},

		// case GtxConfig is nil
		{
			occurError: true,
			errMessage: "global transaction config info is required.",
		},

		// case mock begin return error
		{
			GtxConfig: &GtxConfig{
				Name: "MockGtxConfig",
			},
			occurError:      true,
			errMessage:      "mock begin",
			mockBeginTarget: begin,
			mockBeginFunc: func(ctx context.Context, gc *GtxConfig) error {
				return errors.New("mock begin")
			},
		},

		// case callback return error
		{
			GtxConfig: &GtxConfig{
				Name: "MockGtxConfig",
			},
			callbackErr:     true,
			callback:        callbackError,
			mockBeginTarget: begin,
			mockBeginFunc: func(ctx context.Context, gc *GtxConfig) error {
				return nil
			},
		},

		// case callback return nil
		{
			GtxConfig: &GtxConfig{
				Name: "MockGtxConfig",
			},
			mockBeginTarget: begin,
			mockBeginFunc: func(ctx context.Context, gc *GtxConfig) error {
				SetXID(ctx, "123456")
				return nil
			},
			callback:              callbackNil,
			mockSecondPhaseTarget: commitOrRollback,
			mockSecondPhaseFunc: func(ctx context.Context, s bool) error {
				return nil
			},
		},

		// case second mock error
		{
			GtxConfig: &GtxConfig{
				Name: "MockGtxConfig",
			},
			mockBeginTarget: begin,
			mockBeginFunc: func(ctx context.Context, gc *GtxConfig) error {
				SetXID(ctx, "123456")
				return nil
			},
			callback:              callbackNil,
			mockSecondPhaseTarget: commitOrRollback,
			mockSecondPhaseFunc: func(ctx context.Context, s bool) error {
				return errors.New("second error mock")
			},
			secondErr: true,
		},
	}

	for i, v := range gts {
		t.Logf("Case %v: %+v", i, v)
		var beginStub *gomonkey.Patches
		var secondStub *gomonkey.Patches
		if v.mockBeginTarget != nil {
			beginStub = gomonkey.ApplyFunc(v.mockBeginTarget, v.mockBeginFunc)
		}
		if v.mockSecondPhaseTarget != nil {
			secondStub = gomonkey.ApplyFunc(v.mockSecondPhaseTarget, v.mockSecondPhaseFunc)
		}

		ctx := context.Background()
		err := WithGlobalTx(ctx, v.GtxConfig, v.callback)

		if v.occurError {
			assert.Equal(t, v.errMessage, err.Error())
		}

		if v.callbackErr {
			assert.NotNil(t, err)
		}

		if v.secondErr {
			assert.NotNil(t, err)
		}

		if v.mockBeginTarget != nil {
			beginStub.Reset()
		}

		if v.mockSecondPhaseTarget != nil {
			secondStub.Reset()
		}
	}
}
