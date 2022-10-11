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

	"github.com/agiledragon/gomonkey"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/remoting/getty"
)

func TestBegin(t *testing.T) {
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
				Role: PARTICIPANT,
			},
			wantHasError: false,
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
				Xid:  "123456",
			},
			wantHasError:  true,
			wantErrString: "Global transaction already exists,can't begin a new global transaction, currentXid = 123456 ",
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
			},
			wantHasError:       true,
			wantErrString:      "mock begin return",
			wantHasMock:        true,
			wantMockTargetName: "SendSyncRequest",
			wantMockFunction: func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
				return nil, errors.New("mock begin return")
			},
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
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
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
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
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
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

		err := GetGlobalTransactionManager().Begin(context.Background(), &v.globalTransaction, 1, "GlobalTransaction")
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
				Role: PARTICIPANT,
				Xid:  "123456",
			},
			wantHasError: false,
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
			},
			wantHasError:  true,
			wantErrString: "Commit xid should not be empty",
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
				Xid:  "123456",
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
				Role: LAUNCHER,
				Xid:  "123456",
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

		err := GetGlobalTransactionManager().Commit(context.Background(), &v.globalTransaction)
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
				Role: PARTICIPANT,
				Xid:  "123456",
			},
			wantHasError: false,
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
			},
			wantHasError:  true,
			wantErrString: "Rollback xid should not be empty",
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
				Xid:  "123456",
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
				Role: LAUNCHER,
				Xid:  "123456",
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
