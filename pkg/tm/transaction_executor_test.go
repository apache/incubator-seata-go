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
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

type testGlobalTransactionManager struct {
	beginFunc        func(ctx context.Context, timeout time.Duration) error
	commitFunc       func(ctx context.Context, gtr *tm.GlobalTransaction) error
	rollbackFunc     func(ctx context.Context, gtr *tm.GlobalTransaction) error
	globalReportFunc func(ctx context.Context, gtr *tm.GlobalTransaction) (interface{}, error)
}

func (m *testGlobalTransactionManager) reset() {
	m.beginFunc = nil
	m.commitFunc = nil
	m.rollbackFunc = nil
	m.globalReportFunc = nil
}

func (m *testGlobalTransactionManager) Begin(ctx context.Context, timeout time.Duration) error {
	if m.beginFunc != nil {
		return m.beginFunc(ctx, timeout)
	}
	return nil
}

func (m *testGlobalTransactionManager) Commit(ctx context.Context, gtr *tm.GlobalTransaction) error {
	if tm.IsTimeout(ctx) {
		return m.Rollback(ctx, gtr)
	}
	if m.commitFunc != nil {
		return m.commitFunc(ctx, gtr)
	}
	if gtr != nil && gtr.Xid == "" {
		return fmt.Errorf("Commit xid should not be empty")
	}
	return nil
}

func (m *testGlobalTransactionManager) Rollback(ctx context.Context, gtr *tm.GlobalTransaction) error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(ctx, gtr)
	}
	if gtr != nil && gtr.Xid == "" {
		return fmt.Errorf("Rollback xid should not be empty")
	}
	return nil
}

func (m *testGlobalTransactionManager) GlobalReport(ctx context.Context, gtr *tm.GlobalTransaction) (interface{}, error) {
	if m.globalReportFunc != nil {
		return m.globalReportFunc(ctx, gtr)
	}
	return nil, nil
}

var globalTransactionManagerStub = &testGlobalTransactionManager{}

func TestMain(m *testing.M) {
	tm.SetGlobalTransactionManager(globalTransactionManagerStub)
	code := m.Run()
	os.Exit(code)
}

func TestTransactionExecutorBegin(t *testing.T) {
	type Test struct {
		ctx             context.Context
		gc              *tm.GtxConfig
		xid             string
		beginFunc       func(ctx context.Context, timeout time.Duration) error
		wantHasError    bool
		wantErrorString string
		wantUseExist    bool
		wantBeginNew    bool
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
			xid: "123456",
			beginFunc: func(ctx context.Context, i time.Duration) error {
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
			beginFunc: func(ctx context.Context, i time.Duration) error {
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
			wantErrorString: "not supported propagation:-1",
		},
	}

	for i, v := range gts {
		t.Logf("Case %v: %+v", i, v)
		globalTransactionManagerStub.reset()
		if v.beginFunc != nil {
			globalTransactionManagerStub.beginFunc = v.beginFunc
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
		ctx             context.Context
		tx              tm.GlobalTransaction
		ok              bool
		commitFunc      func(ctx context.Context, gtr *tm.GlobalTransaction) error
		rollbackFunc    func(ctx context.Context, gtr *tm.GlobalTransaction) error
		wantHasError    bool
		wantErrorString string
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
			ok: true,
			commitFunc: func(ctx context.Context, gtr *tm.GlobalTransaction) error {
				return nil
			},
		},
		//ok with error
		{
			ctx: context.Background(),
			tx: tm.GlobalTransaction{
				TxRole: tm.Launcher,
			},
			ok: true,
			commitFunc: func(ctx context.Context, gtr *tm.GlobalTransaction) error {
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
			ok: false,
			rollbackFunc: func(ctx context.Context, gtr *tm.GlobalTransaction) error {
				return nil
			},
		},
		// false with error
		{
			ctx: context.Background(),
			tx: tm.GlobalTransaction{
				TxRole: tm.Launcher,
			},
			ok: false,
			rollbackFunc: func(ctx context.Context, gtr *tm.GlobalTransaction) error {
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
		globalTransactionManagerStub.reset()
		globalTransactionManagerStub.commitFunc = v.commitFunc
		globalTransactionManagerStub.rollbackFunc = v.rollbackFunc

		err := tm.CommitOrRollback(v.ctx, v.ok)

		if v.wantHasError {
			assert.Equal(t, v.wantErrorString, err.Error())
		} else {
			assert.Nil(t, err)
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
	globalTransactionManagerStub.reset()
	assert.Nil(t, tm.BeginNewGtx(ctx, g))
	assert.Equal(t, tm.Launcher, *tm.GetTxRole(ctx))
	assert.Equal(t, g.Name, tm.GetTxName(ctx))
	assert.Equal(t, message.GlobalStatusBegin, *tm.GetTxStatus(ctx))

	// case return error
	err := errors.New("Mock Error")
	globalTransactionManagerStub.beginFunc = func(ctx context.Context, timeout time.Duration) error {
		return err
	}
	assert.Error(t, tm.BeginNewGtx(ctx, g))
}

func TestWithGlobalTx(t *testing.T) {
	callbackError := func(ctx context.Context) error {
		return errors.New("mock callback error")
	}
	callbackNil := func(ctx context.Context) error {
		return nil
	}
	callbackWithTimeout := func(ctx context.Context) error {
		time.Sleep(5 * time.Millisecond)
		return nil
	}

	type testCase struct {
		GtxConfig    *tm.GtxConfig
		occurError   bool
		timeoutErr   bool
		errMessage   string
		callbackErr  bool
		beginFunc    func(ctx context.Context, timeout time.Duration) error
		commitFunc   func(ctx context.Context, gtr *tm.GlobalTransaction) error
		rollbackFunc func(ctx context.Context, gtr *tm.GlobalTransaction) error
		secondErr    bool
		callback     tm.CallbackWithCtx
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
			occurError: true,
			errMessage: "transactionTemplate: Begin transaction failed, error mock begin",
			beginFunc: func(ctx context.Context, timeout time.Duration) error {
				return errors.New("mock begin")
			},
		},

		// case callback return error
		{
			GtxConfig: &tm.GtxConfig{
				Name: "Mocktm.GtxConfig",
			},
			callbackErr: true,
			callback:    callbackError,
			beginFunc: func(ctx context.Context, timeout time.Duration) error {
				tm.SetXID(ctx, "123456")
				return nil
			},
		},

		// case callback return nil
		{
			GtxConfig: &tm.GtxConfig{
				Name: "Mocktm.GtxConfig",
			},
			beginFunc: func(ctx context.Context, timeout time.Duration) error {
				tm.SetXID(ctx, "123456")
				return nil
			},
			callback: callbackNil,
		},

		// case second mock error
		{
			GtxConfig: &tm.GtxConfig{
				Name: "Mocktm.GtxConfig",
			},
			beginFunc: func(ctx context.Context, timeout time.Duration) error {
				tm.SetXID(ctx, "123456")
				return nil
			},
			callback: callbackNil,
			commitFunc: func(ctx context.Context, gtr *tm.GlobalTransaction) error {
				return errors.New("second error mock")
			},
			secondErr: true,
		},

		// case tm detected a timeout and executed rollback successfully.
		{
			GtxConfig: &tm.GtxConfig{
				Name:    "MockGtxConfig",
				Timeout: time.Nanosecond,
			},
			timeoutErr: false,
			beginFunc: func(ctx context.Context, timeout time.Duration) error {
				tm.SetXID(ctx, "123456")
				return nil
			},
			callback: callbackWithTimeout,
			rollbackFunc: func(ctx context.Context, gtr *tm.GlobalTransaction) error {
				return nil
			},
		},
		// tm detected a timeout but rollback threw an exception.
		{
			GtxConfig: &tm.GtxConfig{
				Name:    "MockGtxConfig",
				Timeout: time.Nanosecond,
			},
			timeoutErr: true,
			errMessage: "tm detected a timeout but rollback threw an exception",
			beginFunc: func(ctx context.Context, timeout time.Duration) error {
				tm.SetXID(ctx, "123456")
				return nil
			},
			callback: callbackWithTimeout,
			rollbackFunc: func(ctx context.Context, gtr *tm.GlobalTransaction) error {
				return fmt.Errorf("tm detected a timeout but rollback threw an exception")
			},
		},
	}

	for i, v := range gts {
		t.Logf("Case %v: %+v", i, v)
		globalTransactionManagerStub.reset()
		globalTransactionManagerStub.beginFunc = v.beginFunc
		globalTransactionManagerStub.commitFunc = v.commitFunc
		if v.rollbackFunc != nil {
			globalTransactionManagerStub.rollbackFunc = v.rollbackFunc
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
	}
}

func TestNestedRequiredMustCommitOuterTransaction(t *testing.T) {
	globalTransactionManagerStub.reset()
	defer globalTransactionManagerStub.reset()

	var committedXids []string
	xidCounter := 0

	globalTransactionManagerStub.beginFunc = func(ctx context.Context, timeout time.Duration) error {
		xidCounter++
		tm.SetXID(ctx, fmt.Sprintf("xid-%d", xidCounter))
		return nil
	}
	globalTransactionManagerStub.commitFunc = func(ctx context.Context, gtr *tm.GlobalTransaction) error {
		if gtr != nil {
			committedXids = append(committedXids, gtr.Xid)
		}
		return nil
	}

	err := tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name: "outer", Propagation: tm.Required,
	}, func(ctx context.Context) error {
		return tm.WithGlobalTx(ctx, &tm.GtxConfig{
			Name: "inner", Propagation: tm.Required,
		}, func(context.Context) error {
			return nil
		})
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, len(committedXids))
	assert.Equal(t, "xid-1", committedXids[0])
}

func TestGlobalTxPropagationMatrix(t *testing.T) {
	type testMatrixCase struct {
		name              string
		innerPropagation  tm.Propagation
		innerAction       func(ctx context.Context) error
		outerAction       func(ctx context.Context) error
		wantInnerExecuted bool
		wantInnerTx       bool
		wantOuterErr      bool
		wantCommits       []string
		wantRollbacks     []string
	}

	tests := []testMatrixCase{
		// Required -> Required
		{
			name:             "Required_Required_Success",
			innerPropagation: tm.Required,
			innerAction: func(ctx context.Context) error {
				return nil
			},
			wantInnerExecuted: true,
			wantInnerTx:       true,
			wantOuterErr:      false,
			wantCommits:       []string{"xid-1"},
			wantRollbacks:     nil,
		},
		{
			name:             "Required_Required_InnerError",
			innerPropagation: tm.Required,
			innerAction: func(ctx context.Context) error {
				return fmt.Errorf("inner business error")
			},
			wantInnerExecuted: true,
			wantInnerTx:       true,
			wantOuterErr:      true,
			wantCommits:       nil,
			wantRollbacks:     []string{"xid-1"},
		},
		{
			name:             "Required_Required_InnerPanic",
			innerPropagation: tm.Required,
			innerAction: func(ctx context.Context) error {
				panic("inner runtime panic")
			},
			wantInnerExecuted: true,
			wantInnerTx:       true,
			wantOuterErr:      true,
			wantCommits:       nil,
			wantRollbacks:     []string{"xid-1"},
		},

		// Required -> RequiresNew
		{
			name:             "Required_RequiresNew_Success",
			innerPropagation: tm.RequiresNew,
			innerAction: func(ctx context.Context) error {
				return nil
			},
			wantInnerExecuted: true,
			wantInnerTx:       true,
			wantOuterErr:      false,
			wantCommits:       []string{"xid-2", "xid-1"},
			wantRollbacks:     nil,
		},
		{
			name:             "Required_RequiresNew_InnerError",
			innerPropagation: tm.RequiresNew,
			innerAction: func(ctx context.Context) error {
				return fmt.Errorf("requires_new error")
			},
			wantInnerExecuted: true,
			wantInnerTx:       true,
			wantOuterErr:      true,
			wantCommits:       nil,
			wantRollbacks:     []string{"xid-2", "xid-1"},
		},
		{
			name:             "Required_RequiresNew_OuterError_InnerCommitted",
			innerPropagation: tm.RequiresNew,
			innerAction: func(ctx context.Context) error {
				return nil
			},
			outerAction: func(ctx context.Context) error {
				return fmt.Errorf("outer business error")
			},
			wantInnerExecuted: true,
			wantInnerTx:       true,
			wantOuterErr:      true,
			wantCommits:       []string{"xid-2"},
			wantRollbacks:     []string{"xid-1"},
		},

		// Required -> NotSupported
		{
			name:             "Required_NotSupported_Success",
			innerPropagation: tm.NotSupported,
			innerAction: func(ctx context.Context) error {
				return nil
			},
			wantInnerExecuted: true,
			wantInnerTx:       false,
			wantOuterErr:      false,
			wantCommits:       []string{"xid-1"},
			wantRollbacks:     nil,
		},

		// Required -> Supports
		{
			name:             "Required_Supports_Success",
			innerPropagation: tm.Supports,
			innerAction: func(ctx context.Context) error {
				return nil
			},
			wantInnerExecuted: true,
			wantInnerTx:       true,
			wantOuterErr:      false,
			wantCommits:       []string{"xid-1"},
			wantRollbacks:     nil,
		},

		// Required -> Mandatory
		{
			name:             "Required_Mandatory_Success",
			innerPropagation: tm.Mandatory,
			innerAction: func(ctx context.Context) error {
				return nil
			},
			wantInnerExecuted: true,
			wantInnerTx:       true,
			wantOuterErr:      false,
			wantCommits:       []string{"xid-1"},
			wantRollbacks:     nil,
		},

		// Required -> Never
		{
			name:             "Required_Never_Reject",
			innerPropagation: tm.Never,
			innerAction: func(ctx context.Context) error {
				return nil
			},
			wantInnerExecuted: false,
			wantInnerTx:       false,
			wantOuterErr:      true,
			wantCommits:       nil,
			wantRollbacks:     []string{"xid-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalTransactionManagerStub.reset()
			defer globalTransactionManagerStub.reset()

			var committedXids []string
			var rollbackedXids []string
			xidCounter := 0

			globalTransactionManagerStub.beginFunc = func(ctx context.Context, timeout time.Duration) error {
				xidCounter++
				tm.SetXID(ctx, fmt.Sprintf("xid-%d", xidCounter))
				return nil
			}
			globalTransactionManagerStub.commitFunc = func(ctx context.Context, gtr *tm.GlobalTransaction) error {
				if gtr != nil {
					committedXids = append(committedXids, gtr.Xid)
				}
				return nil
			}
			globalTransactionManagerStub.rollbackFunc = func(ctx context.Context, gtr *tm.GlobalTransaction) error {
				if gtr != nil {
					rollbackedXids = append(rollbackedXids, gtr.Xid)
				}
				return nil
			}

			innerExecuted := false
			err := tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
				Name: "outer", Propagation: tm.Required,
			}, func(ctx context.Context) error {
				innerErr := tm.WithGlobalTx(ctx, &tm.GtxConfig{
					Name: "inner", Propagation: tt.innerPropagation,
				}, func(innerCtx context.Context) error {
					innerExecuted = true
					assert.Equal(t, tt.wantInnerTx, tm.IsGlobalTx(innerCtx))
					if tt.innerAction != nil {
						return tt.innerAction(innerCtx)
					}
					return nil
				})
				if innerErr != nil {
					return innerErr
				}
				if tt.outerAction != nil {
					return tt.outerAction(ctx)
				}
				return nil
			})

			assert.Equal(t, tt.wantInnerExecuted, innerExecuted)
			if tt.wantOuterErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCommits, committedXids)
			assert.Equal(t, tt.wantRollbacks, rollbackedXids)
		})
	}
}
