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

package getty_test

import (
	"context"
	"testing"
	"time"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/tm"
	getty2 "seata.apache.org/seata-go/v2/pkg/tm/transaction/getty"
	"seata.apache.org/seata-go/v2/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type remotingClientFunc func(msg interface{}) (interface{}, error)

func (f remotingClientFunc) SendSyncRequest(msg interface{}) (interface{}, error) {
	return f(msg)
}

func newGettyManager(t *testing.T, send remotingClientFunc) *getty2.GettyGlobalTransactionManager {
	t.Helper()
	if send == nil {
		send = func(msg interface{}) (interface{}, error) {
			t.Fatalf("unexpected SendSyncRequest call with %T", msg)
			return nil, nil
		}
	}
	return getty2.NewGettyGlobalTransactionManager(send)
}

func TestGettyGlobalTransactionBegin(t *testing.T) {
	log.Init()
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	gts := []struct {
		name          string
		gtx           tm.GlobalTransaction
		wantErrString string
		wantXID       string
		send          remotingClientFunc
	}{
		{
			name: "transport error",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantErrString: "mock Begin return",
			send: func(msg interface{}) (interface{}, error) {
				return nil, errors.New("mock Begin return")
			},
		},
		{
			name: "empty response",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantErrString: "GlobalBeginRequest result is empty or result code is failed.",
			send: func(msg interface{}) (interface{}, error) {
				return nil, nil
			},
		},
		{
			name: "failed response",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantErrString: "GlobalBeginRequest result is empty or result code is failed.",
			send: func(msg interface{}) (interface{}, error) {
				return message.GlobalBeginResponse{
					AbstractTransactionResponse: message.AbstractTransactionResponse{
						AbstractResultMessage: message.AbstractResultMessage{
							ResultCode: message.ResultCodeFailed,
						},
					},
				}, nil
			},
		},
		{
			name: "success",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantXID: "test-xid",
			send: func(msg interface{}) (interface{}, error) {
				return message.GlobalBeginResponse{
					AbstractTransactionResponse: message.AbstractTransactionResponse{
						AbstractResultMessage: message.AbstractResultMessage{
							ResultCode: message.ResultCodeSuccess,
						},
					},
					Xid: "test-xid",
				}, nil
			},
		},
	}
	for _, v := range gts {
		t.Run(v.name, func(t *testing.T) {
			calls := 0
			manager := newGettyManager(t, func(msg interface{}) (interface{}, error) {
				calls++
				req, ok := msg.(message.GlobalBeginRequest)
				require.True(t, ok)
				assert.Equal(t, v.gtx.TxName, req.TransactionName)
				assert.Equal(t, 30*time.Second, req.Timeout)
				return v.send(msg)
			})
			ctx := tm.InitSeataContext(context.Background())
			tm.SetTx(ctx, &v.gtx)
			err := manager.Begin(ctx, 30*time.Second)
			assert.Equal(t, 1, calls)
			if v.wantErrString != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), v.wantErrString)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, v.wantXID, tm.GetXID(ctx))
		})
	}
}

func TestGettyGlobalTransactionCommit(t *testing.T) {
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	gts := []struct {
		name          string
		gtx           tm.GlobalTransaction
		wantErrString string
		wantCalls     int
		wantStatus    message.GlobalStatus
		send          remotingClientFunc
	}{
		{
			name: "participant",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Participant,
				Xid:    "123456",
			},
		},
		{
			name: "empty xid",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
			},
			wantErrString: "Commit xid should not be empty",
		},
		{
			name: "transport error",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantErrString: "mock error retry",
			wantCalls:     5,
			send: func(msg interface{}) (interface{}, error) {
				return nil, errors.New("mock error retry")
			},
		},
		{
			name: "success",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantCalls:  1,
			wantStatus: message.GlobalStatusCommitted,
			send: func(msg interface{}) (interface{}, error) {
				return message.GlobalCommitResponse{
					AbstractGlobalEndResponse: message.AbstractGlobalEndResponse{
						AbstractTransactionResponse: message.AbstractTransactionResponse{
							AbstractResultMessage: message.AbstractResultMessage{ResultCode: message.ResultCodeSuccess},
						},
						GlobalStatus: message.GlobalStatusCommitted,
					},
				}, nil
			},
		},
	}
	for _, v := range gts {
		t.Run(v.name, func(t *testing.T) {
			calls := 0
			var send remotingClientFunc
			if v.send != nil {
				send = func(msg interface{}) (interface{}, error) {
					calls++
					req, ok := msg.(message.GlobalCommitRequest)
					require.True(t, ok)
					assert.Equal(t, v.gtx.Xid, req.Xid)
					return v.send(msg)
				}
			}
			manager := newGettyManager(t, send)
			err := manager.Commit(context.Background(), &v.gtx)
			assert.Equal(t, v.wantCalls, calls)
			assert.Equal(t, v.wantStatus, v.gtx.TxStatus)
			if v.wantErrString != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), v.wantErrString)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGettyGlobalTransactionCommitTimeoutUsesInjectedClient(t *testing.T) {
	tm.InitTm(tm.TmConfig{RollbackRetryCount: 1})
	ctx := tm.InitSeataContext(context.Background())
	// NotSupported records the timeout without calling the global manager.
	require.NoError(t, tm.Begin(ctx, &tm.GtxConfig{Propagation: tm.NotSupported, Timeout: time.Nanosecond}))
	require.True(t, tm.IsTimeout(ctx))

	calls := 0
	manager := newGettyManager(t, func(msg interface{}) (interface{}, error) {
		calls++
		req, ok := msg.(message.GlobalRollbackRequest)
		require.True(t, ok)
		assert.Equal(t, "test-xid", req.Xid)
		return message.GlobalRollbackResponse{
			AbstractGlobalEndResponse: message.AbstractGlobalEndResponse{
				AbstractTransactionResponse: message.AbstractTransactionResponse{
					AbstractResultMessage: message.AbstractResultMessage{ResultCode: message.ResultCodeSuccess},
				},
				GlobalStatus: message.GlobalStatusRollbacked,
			},
		}, nil
	})

	gtr := &tm.GlobalTransaction{TxRole: tm.Launcher, Xid: "test-xid"}
	require.NoError(t, manager.Commit(ctx, gtr))
	assert.Equal(t, 1, calls)
	assert.Equal(t, message.GlobalStatusRollbacked, gtr.TxStatus)
}

func TestGettyGlobalTransactionRollback(t *testing.T) {
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	gts := []struct {
		name              string
		globalTransaction tm.GlobalTransaction
		wantErrString     string
		wantCalls         int
		wantStatus        message.GlobalStatus
		send              remotingClientFunc
	}{
		{
			name: "participant",
			globalTransaction: tm.GlobalTransaction{
				TxRole: tm.Participant,
				TxName: "DefaultTx",
				Xid:    "123456",
			},
		},
		{
			name: "empty xid",
			globalTransaction: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
			},
			wantErrString: "Rollback xid should not be empty",
		},
		{
			name: "transport error",
			globalTransaction: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantErrString: "mock error retry",
			wantCalls:     5,
			send: func(msg interface{}) (interface{}, error) {
				return nil, errors.New("mock error retry")
			},
		},
		{
			name: "success",
			globalTransaction: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantCalls:  1,
			wantStatus: message.GlobalStatusRollbacked,
			send: func(msg interface{}) (interface{}, error) {
				return message.GlobalRollbackResponse{
					AbstractGlobalEndResponse: message.AbstractGlobalEndResponse{
						AbstractTransactionResponse: message.AbstractTransactionResponse{
							AbstractResultMessage: message.AbstractResultMessage{ResultCode: message.ResultCodeSuccess},
						},
						GlobalStatus: message.GlobalStatusRollbacked,
					},
				}, nil
			},
		},
	}
	for _, v := range gts {
		t.Run(v.name, func(t *testing.T) {
			calls := 0
			var send remotingClientFunc
			if v.send != nil {
				send = func(msg interface{}) (interface{}, error) {
					calls++
					req, ok := msg.(message.GlobalRollbackRequest)
					require.True(t, ok)
					assert.Equal(t, v.globalTransaction.Xid, req.Xid)
					return v.send(msg)
				}
			}
			manager := newGettyManager(t, send)
			err := manager.Rollback(context.Background(), &v.globalTransaction)
			assert.Equal(t, v.wantCalls, calls)
			assert.Equal(t, v.wantStatus, v.globalTransaction.TxStatus)
			if v.wantErrString != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), v.wantErrString)
				return
			}
			require.NoError(t, err)
		})
	}
}
