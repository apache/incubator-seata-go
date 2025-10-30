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
	"reflect"
	"seata.apache.org/seata-go/pkg/tm"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/util/log"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	getty2 "seata.apache.org/seata-go/pkg/tm/transaction/getty"
)

func TestGettyGlobalTransactionBegin(t *testing.T) {
	log.Init()
	tm.SetGlobalTransactionManager(&getty2.GettyGlobalTransactionManager{})
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
		gtx                tm.GlobalTransaction
		wantHasError       bool
		wantErrString      string
		wantHasMock        bool
		wantMockTargetName string
		wantMockFunction   interface{}
	}{
		{
			gtx: tm.GlobalTransaction{
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
			gtx: tm.GlobalTransaction{
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
			gtx: tm.GlobalTransaction{
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
			gtx: tm.GlobalTransaction{
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
		ctx := tm.InitSeataContext(context.Background())
		tm.SetTx(ctx, &v.gtx)
		err := tm.GetGlobalTransactionManager().Begin(ctx, time.Second*30)
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

func TestGettyGlobalTransactionCommit(t *testing.T) {
	tm.SetGlobalTransactionManager(&getty2.GettyGlobalTransactionManager{})
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
		gtx                tm.GlobalTransaction
		wantHasError       bool
		wantErrString      string
		wantHasMock        bool
		wantMockTargetName string
		wantMockFunction   interface{}
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
			wantHasError:       true,
			wantErrString:      "mock error retry",
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return nil, errors.New("mock error retry")
			},
		},
		{
			gtx: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
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
		tm.SetTx(ctx, &v.gtx)
		err := tm.GetGlobalTransactionManager().Commit(ctx, &v.gtx)
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

func TestGettyGlobalTransactionRollback(t *testing.T) {
	tm.SetGlobalTransactionManager(&getty2.GettyGlobalTransactionManager{})
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
		globalTransaction  tm.GlobalTransaction
		wantHasError       bool
		wantErrString      string
		wantHasMock        bool
		wantMockTargetName string
		wantMockFunction   interface{}
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
			wantHasError:       true,
			wantErrString:      "mock error retry",
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return nil, errors.New("mock error retry")
			},
		},
		{
			globalTransaction: tm.GlobalTransaction{
				TxName: "DefaultTx",
				TxRole: tm.Launcher,
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

		err := tm.GetGlobalTransactionManager().Rollback(context.Background(), &v.globalTransaction)
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
