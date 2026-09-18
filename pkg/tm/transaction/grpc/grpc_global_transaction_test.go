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
	"os"
	"reflect"
	"testing"
	"time"

	"seata.apache.org/seata-go/v2/pkg/remoting/grpc"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/tm"
	grpc2 "seata.apache.org/seata-go/v2/pkg/tm/transaction/grpc"
	"seata.apache.org/seata-go/v2/pkg/util/log"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// SendSyncRequest is patched once for the whole test binary. Applying and
// resetting a gomonkey patch per case does not reliably restore the original
// method, which let one case's canned response leak into the next; that showed
// up as this package failing without -race.
var sendSyncRequestStub func(msg interface{}) (interface{}, error)

func TestMain(m *testing.M) {
	stub := gomonkey.ApplyMethod(reflect.TypeOf(grpc.GetGrpcRemotingClient()), "SendSyncRequest",
		func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
			if sendSyncRequestStub == nil {
				return nil, errors.New("SendSyncRequest called without a canned result")
			}
			return sendSyncRequestStub(msg)
		})
	code := m.Run()
	stub.Reset()
	os.Exit(code)
}

// armSendSyncRequest points the patched method at one case's canned result. The
// cases keep the receiver in their signature, so it is adapted here.
func armSendSyncRequest(enabled bool, fn interface{}) {
	if !enabled || fn == nil {
		sendSyncRequestStub = nil
		return
	}
	typed := fn.(func(*grpc.GrpcRemotingClient, interface{}) (interface{}, error))
	sendSyncRequestStub = func(msg interface{}) (interface{}, error) { return typed(nil, msg) }
}

func TestGrpcGlobalTransactionBegin(t *testing.T) {
	log.Init()
	tm.SetGlobalTransactionManager(&grpc2.GrpcGlobalTransactionManager{})
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		DegradeCheck:                    false,
		DegradeCheckPeriod:              2000,
		DegradeCheckAllowTimes:          10 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	gts := []struct {
		gtx              tm.GlobalTransaction
		protocol         string
		wantHasError     bool
		wantErrString    string
		wantHasMock      bool
		wantMockFunction interface{}
	}{
		{
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantHasError:  true,
			wantErrString: "mock Begin return",
			wantHasMock:   true,
			wantMockFunction: func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
				return nil, errors.New("mock Begin return")
			},
		},
		{
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantHasError:  true,
			wantErrString: "GlobalBeginRequest result is empty or result code is failed.",
			wantHasMock:   true,
			wantMockFunction: func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
				return nil, nil
			},
		},
		{
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantHasError:  true,
			wantErrString: "GlobalBeginRequest result is empty or result code is failed.",
			wantHasMock:   true,
			wantMockFunction: func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
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
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantHasError: false,
			wantHasMock:  true,
			wantMockFunction: func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
				return &pb.GlobalBeginResponseProto{
					AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
						AbstractResultMessage: &pb.AbstractResultMessageProto{
							AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_BEGIN_RESULT},
							ResultCode:      pb.ResultCodeProto_Success,
						},
						TransactionExceptionCode: 0,
					},
				}, nil
			},
		},
	}
	for _, v := range gts {
		armSendSyncRequest(v.wantHasMock, v.wantMockFunction)
		ctx := tm.InitSeataContext(context.Background())
		tm.SetTx(ctx, &v.gtx)
		err := tm.GetGlobalTransactionManager().Begin(ctx, time.Second*30)
		if v.wantHasError {
			assert.NotNil(t, err)
			assert.Regexp(t, v.wantErrString, err.Error())
		} else {
			assert.Nil(t, err)
		}

		armSendSyncRequest(false, nil)
	}
}

func TestGrpcGlobalTransactionCommit(t *testing.T) {
	tm.SetGlobalTransactionManager(&grpc2.GrpcGlobalTransactionManager{})
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		DegradeCheck:                    false,
		DegradeCheckPeriod:              2000,
		DegradeCheckAllowTimes:          10 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	gts := []struct {
		gtx              tm.GlobalTransaction
		wantHasError     bool
		wantErrString    string
		wantHasMock      bool
		wantMockFunction interface{}
	}{
		{
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Participant,
				Xid:    "123456",
			},
			wantHasError: false,
		},
		{
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
			},
			wantHasError:  true,
			wantErrString: "Commit xid should not be empty",
		},
		{
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantHasError:  true,
			wantErrString: "mock error retry",
			wantHasMock:   true,
			wantMockFunction: func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
				return nil, errors.New("mock error retry")
			},
		},
		{
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantHasError: false,
			wantHasMock:  true,
			wantMockFunction: func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
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
	for _, v := range gts {
		armSendSyncRequest(v.wantHasMock, v.wantMockFunction)

		ctx := context.Background()
		tm.SetTx(ctx, &v.gtx)
		err := tm.GetGlobalTransactionManager().Commit(ctx, &v.gtx)
		if v.wantHasError {
			assert.NotNil(t, err)
			assert.Regexp(t, v.wantErrString, err.Error())
		} else {
			assert.Nil(t, err)
		}

		armSendSyncRequest(false, nil)
	}
}

func TestGrpcGlobalTransactionCommitSendsGlobalCommitMessageType(t *testing.T) {
	tm.SetGlobalTransactionManager(&grpc2.GrpcGlobalTransactionManager{})
	tm.InitTm(tm.TmConfig{CommitRetryCount: 1})

	called := false
	armSendSyncRequest(true,
		func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
			called = true
			req, ok := msg.(*pb.GlobalCommitRequestProto)
			assert.True(t, ok)
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
	defer armSendSyncRequest(false, nil)

	gtr := &tm.GlobalTransaction{TxRole: tm.Launcher, Xid: "test-xid"}
	assert.NoError(t, tm.GetGlobalTransactionManager().Commit(context.Background(), gtr))
	assert.True(t, called, "Commit should call SendSyncRequest")
}

func TestGrpcGlobalTransactionRollback(t *testing.T) {
	tm.SetGlobalTransactionManager(&grpc2.GrpcGlobalTransactionManager{})
	tm.InitTm(tm.TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		DegradeCheck:                    false,
		DegradeCheckPeriod:              2000,
		DegradeCheckAllowTimes:          10 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	gts := []struct {
		globalTransaction tm.GlobalTransaction
		wantHasError      bool
		wantErrString     string
		wantHasMock       bool
		wantMockFunction  interface{}
	}{
		{
			globalTransaction: tm.GlobalTransaction{
				TxRole: tm.Participant,
				TxName: "DefaultTx",
				Xid:    "123456",
			},
			wantHasError: false,
		},
		{
			globalTransaction: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
			},
			wantHasError:  true,
			wantErrString: "Rollback xid should not be empty",
		},
		{
			globalTransaction: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantHasError:  true,
			wantErrString: "mock error retry",
			wantHasMock:   true,
			wantMockFunction: func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
				return nil, errors.New("mock error retry")
			},
		},
		{
			globalTransaction: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
				Xid:    "123456",
			},
			wantHasError: false,
			wantHasMock:  true,
			wantMockFunction: func(_ *grpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
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
	for _, v := range gts {
		armSendSyncRequest(v.wantHasMock, v.wantMockFunction)

		err := tm.GetGlobalTransactionManager().Rollback(context.Background(), &v.globalTransaction)
		if v.wantHasError {
			assert.NotNil(t, err)
			assert.Regexp(t, v.wantErrString, err.Error())
		} else {
			assert.Nil(t, err)
		}

		armSendSyncRequest(false, nil)
	}
}
