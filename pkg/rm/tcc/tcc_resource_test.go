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

package tcc

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/agiledragon/gomonkey/v2"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/tm"

	"github.com/stretchr/testify/assert"
)

type mockTCCManagedResource struct{}

func (m mockTCCManagedResource) GetResourceGroupId() string {
	return "DEFAULT"
}

func (m mockTCCManagedResource) GetResourceId() string {
	return "mock-tcc-resource"
}

func (m mockTCCManagedResource) GetBranchType() branch.BranchType {
	return branch.BranchTypeTCC
}

func TestActionContext(t *testing.T) {
	applicationData := `{"actionContext":{"zhangsan":"lisi"}}`
	businessActionContext, err := GetTCCResourceManagerInstance().
		getBusinessActionContext("1111111111", 2645276141, "TestActionContext", []byte(applicationData))

	assert.NoError(t, err)
	assert.NotEmpty(t, businessActionContext)
	bytes, err := json.Marshal(businessActionContext.ActionContext)
	assert.Nil(t, err)
	assert.Equal(t, `{"zhangsan":"lisi"}`, string(bytes))
}

func TestGetBusinessActionContextRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "invalid json", data: "{"},
		{name: "top level array", data: "[]"},
		{name: "top level string", data: `"text"`},
		{name: "top level number", data: "123"},
		{name: "top level null", data: "null"},
		{name: "action context string", data: `{"actionContext":"text"}`},
		{name: "action context number", data: `{"actionContext":123}`},
		{name: "action context array", data: `{"actionContext":[]}`},
		{name: "action context boolean", data: `{"actionContext":true}`},
		{name: "action context null", data: `{"actionContext":null}`},
	}

	manager := GetTCCResourceManagerInstance()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("malformed applicationData panicked: %v", recovered)
				}
			}()

			businessActionContext, err := manager.getBusinessActionContext("xid", 1, "action", []byte(tc.data))
			assert.Error(t, err)
			assert.Nil(t, businessActionContext)
		})
	}
}

func TestGetBusinessActionContextAllowsEmptyAndValidInput(t *testing.T) {
	manager := GetTCCResourceManagerInstance()
	for _, data := range []string{"", `{"actionContext":{"key":"value"}}`} {
		businessActionContext, err := manager.getBusinessActionContext("xid", 1, "action", []byte(data))
		assert.NoError(t, err)
		assert.NotNil(t, businessActionContext)
	}

	oversized := append([]byte(`{"actionContext":{"key":"`), bytes.Repeat([]byte("x"), maxTCCApplicationDataSize)...)
	oversized = append(oversized, []byte(`"}}`)...)
	_, err := manager.getBusinessActionContext("xid", 1, "action", oversized)
	assert.Error(t, err)
}

func FuzzGetBusinessActionContext(f *testing.F) {
	f.Add([]byte(`{"actionContext":{}}`))
	f.Add([]byte(`{"actionContext":"bad"}`))
	manager := GetTCCResourceManagerInstance()
	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("applicationData panicked: %v", recovered)
			}
		}()
		_, _ = manager.getBusinessActionContext("xid", 1, "action", data)
	})
}

type malformedInputTCCAction struct {
	commitCalls   atomic.Int32
	rollbackCalls atomic.Int32
}

type phaseErrorTCCAction struct{}

func (*phaseErrorTCCAction) Prepare(context.Context, interface{}) (bool, error) { return true, nil }
func (*phaseErrorTCCAction) Commit(context.Context, *tm.BusinessActionContext) (bool, error) {
	return false, assert.AnError
}
func (*phaseErrorTCCAction) Rollback(context.Context, *tm.BusinessActionContext) (bool, error) {
	return false, assert.AnError
}
func (*phaseErrorTCCAction) GetActionName() string { return "phase-error-action" }

func (a *malformedInputTCCAction) Prepare(context.Context, interface{}) (bool, error) {
	return true, nil
}

func (a *malformedInputTCCAction) Commit(context.Context, *tm.BusinessActionContext) (bool, error) {
	a.commitCalls.Add(1)
	return true, nil
}

func (a *malformedInputTCCAction) Rollback(context.Context, *tm.BusinessActionContext) (bool, error) {
	a.rollbackCalls.Add(1)
	return true, nil
}

func (a *malformedInputTCCAction) GetActionName() string { return "malformed-input-action" }

func TestBranchPhaseRejectsMalformedApplicationDataBeforeCallback(t *testing.T) {
	action := &malformedInputTCCAction{}
	resource, err := ParseTCCResource(action)
	assert.NoError(t, err)
	manager := GetTCCResourceManagerInstance()
	manager.resourceManagerMap.Store(resource.GetResourceId(), resource)

	commitStatus, err := manager.BranchCommit(context.Background(), rm.BranchResource{
		BranchType:      branch.BranchTypeTCC,
		Xid:             "xid",
		BranchId:        1,
		ResourceId:      resource.GetResourceId(),
		ApplicationData: []byte(`{"actionContext":"invalid"}`),
	})
	assert.Error(t, err)
	assert.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitFailedUnretryable), commitStatus)

	rollbackStatus, err := manager.BranchRollback(context.Background(), rm.BranchResource{
		BranchType:      branch.BranchTypeTCC,
		Xid:             "xid",
		BranchId:        1,
		ResourceId:      resource.GetResourceId(),
		ApplicationData: []byte(`{"actionContext":[]}`),
	})
	assert.Error(t, err)
	assert.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoRollbackFailedUnretryable), rollbackStatus)
	assert.Zero(t, action.commitCalls.Load())
	assert.Zero(t, action.rollbackCalls.Load())
}

func TestBranchPhaseMissingResourceReturnsFailureStatus(t *testing.T) {
	manager := GetTCCResourceManagerInstance()
	commitStatus, err := manager.BranchCommit(context.Background(), rm.BranchResource{ResourceId: "missing-commit-resource"})
	assert.Error(t, err)
	assert.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitFailedUnretryable), commitStatus)

	rollbackStatus, err := manager.BranchRollback(context.Background(), rm.BranchResource{ResourceId: "missing-rollback-resource"})
	assert.Error(t, err)
	assert.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoRollbackFailedUnretryable), rollbackStatus)
}

func TestBranchPhaseCallbackErrorsRemainRetryable(t *testing.T) {
	resource, err := ParseTCCResource(&phaseErrorTCCAction{})
	assert.NoError(t, err)
	manager := GetTCCResourceManagerInstance()
	manager.resourceManagerMap.Store(resource.GetResourceId(), resource)
	t.Cleanup(func() { manager.resourceManagerMap.Delete(resource.GetResourceId()) })

	commitStatus, err := manager.BranchCommit(context.Background(), rm.BranchResource{
		ResourceId: resource.GetResourceId(), ApplicationData: []byte(`{"actionContext":{}}`),
	})
	assert.Error(t, err)
	assert.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitFailedRetryable), commitStatus)

	rollbackStatus, err := manager.BranchRollback(context.Background(), rm.BranchResource{
		ResourceId: resource.GetResourceId(), ApplicationData: []byte(`{"actionContext":{}}`),
	})
	assert.Error(t, err)
	assert.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoRollbackFailedRetryable), rollbackStatus)
}

// TestBranchReport
func TestBranchReport(t *testing.T) {
	patches := gomonkey.ApplyMethod(reflect.TypeOf(getty.GetGettyRemotingClient()), "SendSyncRequest", func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
		return message.BranchReportResponse{
			AbstractTransactionResponse: message.AbstractTransactionResponse{
				AbstractResultMessage: message.AbstractResultMessage{
					ResultCode: message.ResultCodeSuccess,
				},
			},
		}, nil
	})

	defer patches.Reset()

	err := GetTCCResourceManagerInstance().BranchReport(
		context.Background(), rm.BranchReportParam{
			BranchType:      branch.BranchTypeTCC,
			Xid:             "1111111111",
			BranchId:        2645276141,
			Status:          branch.BranchStatusPhaseoneDone,
			ApplicationData: `{"actionContext":{"zhangsan":"lisi"}}`,
		})

	assert.Nil(t, err)
}

func TestLockQueryReturnsFalse(t *testing.T) {
	lockable, err := GetTCCResourceManagerInstance().LockQuery(context.Background(), rm.LockQueryParam{
		BranchType: branch.BranchTypeTCC,
		ResourceId: "mock-tcc-resource",
		Xid:        "xid-1",
		LockKeys:   "ignored",
	})

	assert.NoError(t, err)
	assert.False(t, lockable)
}

func TestUnregisterResourceReturnsExplicitError(t *testing.T) {
	err := GetTCCResourceManagerInstance().UnregisterResource(mockTCCManagedResource{})

	assert.EqualError(t, err, "UnregisterResource is not supported for TCCResourceManager")
}
