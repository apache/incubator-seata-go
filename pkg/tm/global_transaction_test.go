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

	"seata.apache.org/seata-go/pkg/util/log"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/getty"
)

func TestBegin(t *testing.T) {
	log.Init()
	InitTm(TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		DegradeCheck:                    false,
		DegradeCheckPeriod:              2000,
		DegradeCheckAllowTimes:          10 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	gts := []struct {
		gtx                GlobalTransaction
		wantHasError       bool
		wantErrString      string
		wantHasMock        bool
		wantMockTargetName string
		wantMockFunction   interface{}
	}{
		{
			gtx: GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantHasError:       true,
			wantErrString:      "mock Begin return",
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return nil, errors.New("mock Begin return")
			},
		},
		{
			gtx: GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantHasError:       true,
			wantErrString:      "GlobalBeginRequest result is empty or result code is failed.",
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return nil, nil
			},
		},
		{
			gtx: GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantHasError:       true,
			wantErrString:      "GlobalBeginRequest result is empty or result code is failed.",
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
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
			gtx: GlobalTransaction{
				TxName: "DefaultTx",
			},
			wantHasError:       false,
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return message.GlobalBeginResponse{
					AbstractTransactionResponse: message.AbstractTransactionResponse{
						AbstractResultMessage: message.AbstractResultMessage{
							ResultCode: message.ResultCodeSuccess,
						},
					},
				}, nil
			},
		},
	}
	for _, v := range gts {
		var stub *gomonkey.Patches
		// set up stub
		if v.wantHasMock {
			stub = gomonkey.ApplyMethod(reflect.TypeOf(getty.GetGettyRemotingClient()), v.wantMockTargetName, v.wantMockFunction)
		}
		ctx := InitSeataContext(context.Background())
		SetTx(ctx, &v.gtx)
		err := GetGlobalTransactionManager().Begin(ctx, time.Second*30)
		if v.wantHasError {
			assert.NotNil(t, err)
			assert.Regexp(t, v.wantErrString, err.Error())
		} else {
			assert.Nil(t, err)
		}

		// reset up stub
		if v.wantHasMock {
			stub.Reset()
		}
	}
}

func TestCommit(t *testing.T) {
	InitTm(TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		DegradeCheck:                    false,
		DegradeCheckPeriod:              2000,
		DegradeCheckAllowTimes:          10 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	gts := []struct {
		gtx                GlobalTransaction
		wantHasError       bool
		wantErrString      string
		wantHasMock        bool
		wantMockTargetName string
		wantMockFunction   interface{}
	}{
		{
			gtx: GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: Participant,
				Xid:    "123456",
			},
			wantHasError: false,
		},
		{
			gtx: GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: Launcher,
			},
			wantHasError:  true,
			wantErrString: "Commit xid should not be empty",
		},
		{
			gtx: GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: Launcher,
				Xid:    "123456",
			},
			wantHasError:       true,
			wantErrString:      "mock error retry",
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return nil, errors.New("mock error retry")
			},
		},
		{
			gtx: GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: Launcher,
				Xid:    "123456",
			},
			wantHasError:       false,
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return message.GlobalCommitResponse{
					AbstractGlobalEndResponse: message.AbstractGlobalEndResponse{
						GlobalStatus: message.GlobalStatusCommitted,
					},
				}, nil
			},
		},
	}
	for _, v := range gts {
		var stub *gomonkey.Patches
		// set up stub
		if v.wantHasMock {
			stub = gomonkey.ApplyMethod(reflect.TypeOf(getty.GetGettyRemotingClient()), v.wantMockTargetName, v.wantMockFunction)
		}

		ctx := context.Background()
		SetTx(ctx, &v.gtx)
		err := GetGlobalTransactionManager().Commit(ctx, &v.gtx)
		if v.wantHasError {
			assert.NotNil(t, err)
			assert.Regexp(t, v.wantErrString, err.Error())
		} else {
			assert.Nil(t, err)
		}

		// rest up stub
		if v.wantHasMock {
			stub.Reset()
		}
	}
}

func TestRollback(t *testing.T) {
	InitTm(TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60 * time.Second,
		DegradeCheck:                    false,
		DegradeCheckPeriod:              2000,
		DegradeCheckAllowTimes:          10 * time.Second,
		InterceptorOrder:                -2147482648,
	})
	gts := []struct {
		globalTransaction  GlobalTransaction
		wantHasError       bool
		wantErrString      string
		wantHasMock        bool
		wantMockTargetName string
		wantMockFunction   interface{}
	}{
		{
			globalTransaction: GlobalTransaction{
				TxRole: Participant,
				TxName: "DefaultTx",
				Xid:    "123456",
			},
			wantHasError: false,
		},
		{
			globalTransaction: GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: Launcher,
			},
			wantHasError:  true,
			wantErrString: "Rollback xid should not be empty",
		},
		{
			globalTransaction: GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: Launcher,
				Xid:    "123456",
			},
			wantHasError:       true,
			wantErrString:      "mock error retry",
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return nil, errors.New("mock error retry")
			},
		},
		{
			globalTransaction: GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: Launcher,
				Xid:    "123456",
			},
			wantHasError:       false,
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return message.GlobalRollbackResponse{
					AbstractGlobalEndResponse: message.AbstractGlobalEndResponse{
						GlobalStatus: message.GlobalStatusRollbacked,
					},
				}, nil
			},
		},
	}
	for _, v := range gts {
		var stub *gomonkey.Patches
		// set up stub
		if v.wantHasMock {
			stub = gomonkey.ApplyMethod(reflect.TypeOf(getty.GetGettyRemotingClient()), v.wantMockTargetName, v.wantMockFunction)
		}

		err := GetGlobalTransactionManager().Rollback(context.Background(), &v.globalTransaction)
		if v.wantHasError {
			assert.NotNil(t, err)
			assert.Regexp(t, v.wantErrString, err.Error())
		} else {
			assert.Nil(t, err)
		}

		// rest up stub
		if v.wantHasMock {
			stub.Reset()
		}
	}
}
