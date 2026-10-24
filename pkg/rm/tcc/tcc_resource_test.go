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
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/tm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type phaseTwoResultService struct {
	commitResult   bool
	commitErr      error
	rollbackResult bool
	rollbackErr    error
}

func (*phaseTwoResultService) Prepare(context.Context, interface{}) (bool, error) {
	return true, nil
}

func (s *phaseTwoResultService) Commit(context.Context, *tm.BusinessActionContext) (bool, error) {
	return s.commitResult, s.commitErr
}

func (s *phaseTwoResultService) Rollback(context.Context, *tm.BusinessActionContext) (bool, error) {
	return s.rollbackResult, s.rollbackErr
}

func (*phaseTwoResultService) GetActionName() string {
	return "phase-two-result-service"
}

func newPhaseTwoResultManager(t *testing.T, service *phaseTwoResultService) (*TCCResourceManager, rm.BranchResource) {
	t.Helper()
	resource, err := ParseTCCResource(service)
	require.NoError(t, err)

	manager := &TCCResourceManager{}
	manager.resourceManagerMap.Store(resource.GetResourceId(), resource)
	return manager, rm.BranchResource{ResourceId: resource.GetResourceId(), Xid: "xid-1", BranchId: 1}
}

func TestBranchCommitUsesActionResult(t *testing.T) {
	actionErr := errors.New("commit failed")
	tests := []struct {
		name       string
		result     bool
		err        error
		wantStatus branch.BranchStatus
	}{
		{"success", true, nil, branch.BranchStatusPhasetwoCommitted},
		{"false_without_error", false, nil, branch.BranchStatusPhasetwoCommitFailedRetryable},
		{"error", true, actionErr, branch.BranchStatusPhasetwoCommitFailedRetryable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, resource := newPhaseTwoResultManager(t, &phaseTwoResultService{commitResult: tt.result, commitErr: tt.err})
			status, err := manager.BranchCommit(context.Background(), resource)
			assert.Equal(t, tt.wantStatus, status)
			if tt.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.err)
			}
		})
	}
}

func TestBranchRollbackUsesActionResult(t *testing.T) {
	actionErr := errors.New("rollback failed")
	tests := []struct {
		name       string
		result     bool
		err        error
		wantStatus branch.BranchStatus
	}{
		{"success", true, nil, branch.BranchStatusPhasetwoRollbacked},
		{"false_without_error", false, nil, branch.BranchStatusPhasetwoRollbackFailedRetryable},
		{"error", true, actionErr, branch.BranchStatusPhasetwoRollbackFailedRetryable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, resource := newPhaseTwoResultManager(t, &phaseTwoResultService{rollbackResult: tt.result, rollbackErr: tt.err})
			status, err := manager.BranchRollback(context.Background(), resource)
			assert.Equal(t, tt.wantStatus, status)
			if tt.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.err)
			}
		})
	}
}

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
	businessActionContext := GetTCCResourceManagerInstance().
		getBusinessActionContext("1111111111", 2645276141, "TestActionContext", []byte(applicationData))

	assert.NotEmpty(t, businessActionContext)
	bytes, err := json.Marshal(businessActionContext.ActionContext)
	assert.Nil(t, err)
	assert.Equal(t, `{"zhangsan":"lisi"}`, string(bytes))
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
