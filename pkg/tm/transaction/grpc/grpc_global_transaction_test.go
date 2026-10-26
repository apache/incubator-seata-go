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

package grpc_test

import (
	"context"
	"testing"
	"time"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/tm"
	grpc2 "seata.apache.org/seata-go/v2/pkg/tm/transaction/grpc"
	"seata.apache.org/seata-go/v2/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type remotingClientFunc func(msg interface{}) (interface{}, error)

func (f remotingClientFunc) SendSyncRequest(msg interface{}) (interface{}, error) {
	return f(msg)
}

func newGrpcManager(t *testing.T, send remotingClientFunc) *grpc2.GrpcGlobalTransactionManager {
	t.Helper()
	if send == nil {
		send = func(msg interface{}) (interface{}, error) {
			t.Fatalf("unexpected SendSyncRequest call with %T", msg)
			return nil, nil
		}
	}
	return grpc2.NewGrpcGlobalTransactionManager(send)
}

func TestGrpcGlobalTransactionBegin(t *testing.T) {
	log.Init()
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	tests := []struct {
		name    string
		gtx     tm.GlobalTransaction
		send    remotingClientFunc
		wantErr string
		wantXID string
	}{
		{
			name: "transport error",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantErr: "mock Begin return",
			send: func(msg interface{}) (interface{}, error) {
				return nil, errors.New("mock Begin return")
			},
		},
		{
			name: "empty response",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantErr: "GlobalBeginRequest result is empty or result code is failed.",
			send: func(msg interface{}) (interface{}, error) {
				return nil, nil
			},
		},
		{
			name: "failed response",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantErr: "GlobalBeginRequest result is empty or result code is failed.",
			send: func(msg interface{}) (interface{}, error) {
				return &pb.GlobalBeginResponseProto{
					AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
						AbstractResultMessage: &pb.AbstractResultMessageProto{
							AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_BEGIN_RESULT},
							ResultCode:      pb.ResultCodeProto_Failed,
						},
						TransactionExceptionCode: 0,
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
				return &pb.GlobalBeginResponseProto{
					AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
						AbstractResultMessage: &pb.AbstractResultMessageProto{
							AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_BEGIN_RESULT},
							ResultCode:      pb.ResultCodeProto_Success,
						},
						TransactionExceptionCode: 0,
					},
					Xid: "test-xid",
				}, nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := newGrpcManager(t, func(msg interface{}) (interface{}, error) {
				req, ok := msg.(*pb.GlobalBeginRequestProto)
				require.True(t, ok)
				assert.Equal(t, pb.MessageTypeProto_TYPE_GLOBAL_BEGIN, req.GetAbstractTransactionRequest().GetAbstractMessage().GetMessageType())
				assert.Equal(t, int32(30000), req.GetTimeout())
				assert.Equal(t, test.gtx.TxName, req.GetTransactionName())
				return test.send(msg)
			})
			ctx := tm.InitSeataContext(context.Background())
			tm.SetTx(ctx, &test.gtx)
			err := manager.Begin(ctx, 30*time.Second)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantXID, tm.GetXID(ctx))
		})
	}
}

func TestGrpcGlobalTransactionCommit(t *testing.T) {
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	tests := []struct {
		name       string
		gtx        tm.GlobalTransaction
		send       remotingClientFunc
		wantErr    string
		wantCalls  int
		wantStatus message.GlobalStatus
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
			wantErr: "Commit xid should not be empty",
		},
		{
			name: "transport error",
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantErr:   "mock error retry",
			wantCalls: 5,
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
				return &pb.GlobalCommitResponseProto{
					AbstractGlobalEndResponse: &pb.AbstractGlobalEndResponseProto{
						AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
							AbstractResultMessage: &pb.AbstractResultMessageProto{
								AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_COMMIT_RESULT},
								ResultCode:      pb.ResultCodeProto_Success,
							},
						},
						GlobalStatus: pb.GlobalStatusProto_Committed,
					},
				}, nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			var send remotingClientFunc
			if test.send != nil {
				send = func(msg interface{}) (interface{}, error) {
					calls++
					req, ok := msg.(*pb.GlobalCommitRequestProto)
					require.True(t, ok)
					assert.Equal(t, pb.MessageTypeProto_TYPE_GLOBAL_COMMIT, req.GetAbstractGlobalEndRequest().GetAbstractTransactionRequest().GetAbstractMessage().GetMessageType())
					assert.Equal(t, test.gtx.Xid, req.GetAbstractGlobalEndRequest().GetXid())
					return test.send(msg)
				}
			}
			manager := newGrpcManager(t, send)
			err := manager.Commit(context.Background(), &test.gtx)
			assert.Equal(t, test.wantCalls, calls)
			assert.Equal(t, test.wantStatus, test.gtx.TxStatus)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGrpcGlobalTransactionCommitSendsGlobalCommitMessageType(t *testing.T) {
	tm.InitTm(tm.TmConfig{CommitRetryCount: 1})

	called := false
	manager := newGrpcManager(t, func(msg interface{}) (interface{}, error) {
		called = true
		req, ok := msg.(*pb.GlobalCommitRequestProto)
		require.True(t, ok)
		assert.Equal(t, pb.MessageTypeProto_TYPE_GLOBAL_COMMIT,
			req.GetAbstractGlobalEndRequest().GetAbstractTransactionRequest().GetAbstractMessage().GetMessageType())

		return &pb.GlobalCommitResponseProto{
			AbstractGlobalEndResponse: &pb.AbstractGlobalEndResponseProto{
				AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
					AbstractResultMessage: &pb.AbstractResultMessageProto{
						AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_COMMIT_RESULT},
						ResultCode:      pb.ResultCodeProto_Success,
					},
				},
				GlobalStatus: pb.GlobalStatusProto_Committed,
			},
		}, nil
	})

	gtr := &tm.GlobalTransaction{TxRole: tm.Launcher, Xid: "test-xid"}
	require.NoError(t, manager.Commit(context.Background(), gtr))
	assert.True(t, called, "Commit should call SendSyncRequest")
	assert.Equal(t, message.GlobalStatusCommitted, gtr.TxStatus)
}

func TestGrpcGlobalTransactionCommitTimeoutUsesInjectedClient(t *testing.T) {
	tm.InitTm(tm.TmConfig{RollbackRetryCount: 1})
	ctx := tm.InitSeataContext(context.Background())
	// NotSupported records the timeout without calling the global manager.
	require.NoError(t, tm.Begin(ctx, &tm.GtxConfig{Propagation: tm.NotSupported, Timeout: time.Nanosecond}))
	require.True(t, tm.IsTimeout(ctx))

	calls := 0
	manager := newGrpcManager(t, func(msg interface{}) (interface{}, error) {
		calls++
		req, ok := msg.(*pb.GlobalRollbackRequestProto)
		require.True(t, ok)
		assert.Equal(t, "test-xid", req.GetAbstractGlobalEndRequest().GetXid())
		return &pb.GlobalRollbackResponseProto{
			AbstractGlobalEndResponse: &pb.AbstractGlobalEndResponseProto{
				AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
					AbstractResultMessage: &pb.AbstractResultMessageProto{
						AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_ROLLBACK_RESULT},
						ResultCode:      pb.ResultCodeProto_Success,
					},
				},
				GlobalStatus: pb.GlobalStatusProto_Rollbacked,
			},
		}, nil
	})

	gtr := &tm.GlobalTransaction{TxRole: tm.Launcher, Xid: "test-xid"}
	require.NoError(t, manager.Commit(ctx, gtr))
	assert.Equal(t, 1, calls)
	assert.Equal(t, message.GlobalStatusRollbacked, gtr.TxStatus)
}

func TestGrpcGlobalTransactionRollback(t *testing.T) {
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	tests := []struct {
		name              string
		globalTransaction tm.GlobalTransaction
		send              remotingClientFunc
		wantErr           string
		wantCalls         int
		wantStatus        message.GlobalStatus
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
			wantErr: "Rollback xid should not be empty",
		},
		{
			name: "transport error",
			globalTransaction: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantErr:   "mock error retry",
			wantCalls: 5,
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
				return &pb.GlobalRollbackResponseProto{
					AbstractGlobalEndResponse: &pb.AbstractGlobalEndResponseProto{
						AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
							AbstractResultMessage: &pb.AbstractResultMessageProto{
								AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_ROLLBACK_RESULT},
								ResultCode:      pb.ResultCodeProto_Success,
							},
						},
						GlobalStatus: pb.GlobalStatusProto_Rollbacked,
					},
				}, nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			var send remotingClientFunc
			if test.send != nil {
				send = func(msg interface{}) (interface{}, error) {
					calls++
					req, ok := msg.(*pb.GlobalRollbackRequestProto)
					require.True(t, ok)
					assert.Equal(t, pb.MessageTypeProto_TYPE_GLOBAL_ROLLBACK, req.GetAbstractGlobalEndRequest().GetAbstractTransactionRequest().GetAbstractMessage().GetMessageType())
					assert.Equal(t, test.globalTransaction.Xid, req.GetAbstractGlobalEndRequest().GetXid())
					return test.send(msg)
				}
			}
			manager := newGrpcManager(t, send)
			err := manager.Rollback(context.Background(), &test.globalTransaction)
			assert.Equal(t, test.wantCalls, calls)
			assert.Equal(t, test.wantStatus, test.globalTransaction.TxStatus)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
