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

package rm

import (
	"context"
	"reflect"
	"sync"

	"github.com/golang/mock/gomock"

	"seata.apache.org/seata-go/pkg/protocol/branch"
)

// MockResource is a mock of Resource interface.
type MockResource struct {
	ctrl     *gomock.Controller
	recorder *MockResourceMockRecorder
}

// MockResourceMockRecorder is the mock recorder for MockResource.
type MockResourceMockRecorder struct {
	mock *MockResource
}

// NewMockResource creates a new mock instance.
func NewMockResource(ctrl *gomock.Controller) *MockResource {
	mock := &MockResource{ctrl: ctrl}
	mock.recorder = &MockResourceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResource) EXPECT() *MockResourceMockRecorder {
	return m.recorder
}

// GetBranchType mocks base method.
func (m *MockResource) GetBranchType() branch.BranchType {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBranchType")
	ret0, _ := ret[0].(branch.BranchType)
	return ret0
}

// GetBranchType indicates an expected call of GetBranchType.
func (mr *MockResourceMockRecorder) GetBranchType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBranchType", reflect.TypeOf((*MockResource)(nil).GetBranchType))
}

// GetResourceGroupId mocks base method.
func (m *MockResource) GetResourceGroupId() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResourceGroupId")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetResourceGroupId indicates an expected call of GetResourceGroupId.
func (mr *MockResourceMockRecorder) GetResourceGroupId() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResourceGroupId", reflect.TypeOf((*MockResource)(nil).GetResourceGroupId))
}

// GetResourceId mocks base method.
func (m *MockResource) GetResourceId() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResourceId")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetResourceId indicates an expected call of GetResourceId.
func (mr *MockResourceMockRecorder) GetResourceId() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResourceId", reflect.TypeOf((*MockResource)(nil).GetResourceId))
}

// MockResourceManagerInbound is a mock of ResourceManagerInbound interface.
type MockResourceManagerInbound struct {
	ctrl     *gomock.Controller
	recorder *MockResourceManagerInboundMockRecorder
}

// MockResourceManagerInboundMockRecorder is the mock recorder for MockResourceManagerInbound.
type MockResourceManagerInboundMockRecorder struct {
	mock *MockResourceManagerInbound
}

// NewMockResourceManagerInbound creates a new mock instance.
func NewMockResourceManagerInbound(ctrl *gomock.Controller) *MockResourceManagerInbound {
	mock := &MockResourceManagerInbound{ctrl: ctrl}
	mock.recorder = &MockResourceManagerInboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResourceManagerInbound) EXPECT() *MockResourceManagerInboundMockRecorder {
	return m.recorder
}

// BranchCommit mocks base method.
func (m *MockResourceManagerInbound) BranchCommit(ctx context.Context, resource BranchResource) (branch.BranchStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BranchCommit", ctx, resource)
	ret0, _ := ret[0].(branch.BranchStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BranchCommit indicates an expected call of BranchCommit.
func (mr *MockResourceManagerInboundMockRecorder) BranchCommit(ctx, resource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BranchCommit", reflect.TypeOf((*MockResourceManagerInbound)(nil).BranchCommit), ctx, resource)
}

// BranchRollback mocks base method.
func (m *MockResourceManagerInbound) BranchRollback(ctx context.Context, resource BranchResource) (branch.BranchStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BranchRollback", ctx, resource)
	ret0, _ := ret[0].(branch.BranchStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BranchRollback indicates an expected call of BranchRollback.
func (mr *MockResourceManagerInboundMockRecorder) BranchRollback(ctx, resource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BranchRollback", reflect.TypeOf((*MockResourceManagerInbound)(nil).BranchRollback), ctx, resource)
}

// MockResourceManagerOutbound is a mock of ResourceManagerOutbound interface.
type MockResourceManagerOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockResourceManagerOutboundMockRecorder
}

// MockResourceManagerOutboundMockRecorder is the mock recorder for MockResourceManagerOutbound.
type MockResourceManagerOutboundMockRecorder struct {
	mock *MockResourceManagerOutbound
}

// NewMockResourceManagerOutbound creates a new mock instance.
func NewMockResourceManagerOutbound(ctrl *gomock.Controller) *MockResourceManagerOutbound {
	mock := &MockResourceManagerOutbound{ctrl: ctrl}
	mock.recorder = &MockResourceManagerOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResourceManagerOutbound) EXPECT() *MockResourceManagerOutboundMockRecorder {
	return m.recorder
}

// BranchRegister mocks base method.
func (m *MockResourceManagerOutbound) BranchRegister(ctx context.Context, param BranchRegisterParam) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BranchRegister", ctx, param)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BranchRegister indicates an expected call of BranchRegister.
func (mr *MockResourceManagerOutboundMockRecorder) BranchRegister(ctx, param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BranchRegister", reflect.TypeOf((*MockResourceManagerOutbound)(nil).BranchRegister), ctx, param)
}

// BranchReport mocks base method.
func (m *MockResourceManagerOutbound) BranchReport(ctx context.Context, param BranchReportParam) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BranchReport", ctx, param)
	ret0, _ := ret[0].(error)
	return ret0
}

// BranchReport indicates an expected call of BranchReport.
func (mr *MockResourceManagerOutboundMockRecorder) BranchReport(ctx, param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BranchReport", reflect.TypeOf((*MockResourceManagerOutbound)(nil).BranchReport), ctx, param)
}

// LockQuery mocks base method.
func (m *MockResourceManagerOutbound) LockQuery(ctx context.Context, param LockQueryParam) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LockQuery", ctx, param)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LockQuery indicates an expected call of LockQuery.
func (mr *MockResourceManagerOutboundMockRecorder) LockQuery(ctx, param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LockQuery", reflect.TypeOf((*MockResourceManagerOutbound)(nil).LockQuery), ctx, param)
}

// MockResourceManager is a mock of ResourceManager interface.
type MockResourceManager struct {
	ctrl     *gomock.Controller
	recorder *MockResourceManagerMockRecorder
}

// MockResourceManagerMockRecorder is the mock recorder for MockResourceManager.
type MockResourceManagerMockRecorder struct {
	mock *MockResourceManager
}

// NewMockResourceManager creates a new mock instance.
func NewMockResourceManager(ctrl *gomock.Controller) *MockResourceManager {
	mock := &MockResourceManager{ctrl: ctrl}
	mock.recorder = &MockResourceManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResourceManager) EXPECT() *MockResourceManagerMockRecorder {
	return m.recorder
}

// BranchCommit mocks base method.
func (m *MockResourceManager) BranchCommit(ctx context.Context, resource BranchResource) (branch.BranchStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BranchCommit", ctx, resource)
	ret0, _ := ret[0].(branch.BranchStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BranchCommit indicates an expected call of BranchCommit.
func (mr *MockResourceManagerMockRecorder) BranchCommit(ctx, resource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BranchCommit", reflect.TypeOf((*MockResourceManager)(nil).BranchCommit), ctx, resource)
}

// BranchRegister mocks base method.
func (m *MockResourceManager) BranchRegister(ctx context.Context, param BranchRegisterParam) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BranchRegister", ctx, param)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BranchRegister indicates an expected call of BranchRegister.
func (mr *MockResourceManagerMockRecorder) BranchRegister(ctx, param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BranchRegister", reflect.TypeOf((*MockResourceManager)(nil).BranchRegister), ctx, param)
}

// BranchReport mocks base method.
func (m *MockResourceManager) BranchReport(ctx context.Context, param BranchReportParam) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BranchReport", ctx, param)
	ret0, _ := ret[0].(error)
	return ret0
}

// BranchReport indicates an expected call of BranchReport.
func (mr *MockResourceManagerMockRecorder) BranchReport(ctx, param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BranchReport", reflect.TypeOf((*MockResourceManager)(nil).BranchReport), ctx, param)
}

// BranchRollback mocks base method.
func (m *MockResourceManager) BranchRollback(ctx context.Context, resource BranchResource) (branch.BranchStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BranchRollback", ctx, resource)
	ret0, _ := ret[0].(branch.BranchStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BranchRollback indicates an expected call of BranchRollback.
func (mr *MockResourceManagerMockRecorder) BranchRollback(ctx, resource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BranchRollback", reflect.TypeOf((*MockResourceManager)(nil).BranchRollback), ctx, resource)
}

// GetBranchType mocks base method.
func (m *MockResourceManager) GetBranchType() branch.BranchType {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBranchType")
	ret0, _ := ret[0].(branch.BranchType)
	return ret0
}

// GetBranchType indicates an expected call of GetBranchType.
func (mr *MockResourceManagerMockRecorder) GetBranchType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBranchType", reflect.TypeOf((*MockResourceManager)(nil).GetBranchType))
}

// GetCachedResources mocks base method.
func (m *MockResourceManager) GetCachedResources() *sync.Map {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCachedResources")
	ret0, _ := ret[0].(*sync.Map)
	return ret0
}

// GetCachedResources indicates an expected call of GetCachedResources.
func (mr *MockResourceManagerMockRecorder) GetCachedResources() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCachedResources", reflect.TypeOf((*MockResourceManager)(nil).GetCachedResources))
}

// LockQuery mocks base method.
func (m *MockResourceManager) LockQuery(ctx context.Context, param LockQueryParam) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LockQuery", ctx, param)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LockQuery indicates an expected call of LockQuery.
func (mr *MockResourceManagerMockRecorder) LockQuery(ctx, param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LockQuery", reflect.TypeOf((*MockResourceManager)(nil).LockQuery), ctx, param)
}

// RegisterResource mocks base method.
func (m *MockResourceManager) RegisterResource(resource Resource) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterResource", resource)
	ret0, _ := ret[0].(error)
	return ret0
}

// RegisterResource indicates an expected call of RegisterResource.
func (mr *MockResourceManagerMockRecorder) RegisterResource(resource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterResource", reflect.TypeOf((*MockResourceManager)(nil).RegisterResource), resource)
}

// UnregisterResource mocks base method.
func (m *MockResourceManager) UnregisterResource(resource Resource) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnregisterResource", resource)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnregisterResource indicates an expected call of UnregisterResource.
func (mr *MockResourceManagerMockRecorder) UnregisterResource(resource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnregisterResource", reflect.TypeOf((*MockResourceManager)(nil).UnregisterResource), resource)
}

// MockResourceManagerGetter is a mock of ResourceManagerGetter interface.
type MockResourceManagerGetter struct {
	ctrl     *gomock.Controller
	recorder *MockResourceManagerGetterMockRecorder
}

// MockResourceManagerGetterMockRecorder is the mock recorder for MockResourceManagerGetter.
type MockResourceManagerGetterMockRecorder struct {
	mock *MockResourceManagerGetter
}

// NewMockResourceManagerGetter creates a new mock instance.
func NewMockResourceManagerGetter(ctrl *gomock.Controller) *MockResourceManagerGetter {
	mock := &MockResourceManagerGetter{ctrl: ctrl}
	mock.recorder = &MockResourceManagerGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResourceManagerGetter) EXPECT() *MockResourceManagerGetterMockRecorder {
	return m.recorder
}

// GetResourceManager mocks base method.
func (m *MockResourceManagerGetter) GetResourceManager(branchType branch.BranchType) ResourceManager {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResourceManager", branchType)
	ret0, _ := ret[0].(ResourceManager)
	return ret0
}

// GetResourceManager indicates an expected call of GetResourceManager.
func (mr *MockResourceManagerGetterMockRecorder) GetResourceManager(branchType interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResourceManager", reflect.TypeOf((*MockResourceManagerGetter)(nil).GetResourceManager), branchType)
}
