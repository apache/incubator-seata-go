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

package db

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/pkg/protocol/branch"
	rmpkg "seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine"
	engExc "seata.apache.org/seata-go/pkg/saga/statemachine/engine/exception"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/expr"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/invoker"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/pcext"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/repo"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/sequence"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/strategy"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/utils"
	"seata.apache.org/seata-go/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/pkg/saga/statemachine/process_ctrl/process"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
	stateimpl "seata.apache.org/seata-go/pkg/saga/statemachine/statelang/state"
	storepkg "seata.apache.org/seata-go/pkg/saga/statemachine/store"
	"seata.apache.org/seata-go/pkg/tm"
	seataErrors "seata.apache.org/seata-go/pkg/util/errors"
)

func mockProcessContext(stateMachineName string, stateMachineInstance statelang.StateMachineInstance) process_ctrl.ProcessContext {
	ctx := utils.NewProcessContextBuilder().
		WithProcessType(process.StateLang).
		WithOperationName(constant.OperationNameStart).
		WithInstruction(pcext.NewStateInstruction(stateMachineName, "000001")).
		WithStateMachineInstance(stateMachineInstance).
		Build()
	return ctx
}

func mockMachineInstance(stateMachineName string) statelang.StateMachineInstance {
	stateMachine := statelang.NewStateMachineImpl()
	stateMachine.SetName(stateMachineName)
	stateMachine.SetComment("This is a test state machine")
	stateMachine.SetCreateTime(time.Now())
	stateMachine.SetID(stateMachineName)
	stateMachine.SetTenantId("000001")

	inst := statelang.NewStateMachineInstanceImpl()
	inst.SetStateMachine(stateMachine)
	inst.SetMachineID(stateMachineName)

	inst.SetStartParams(map[string]any{"start": 100})
	inst.SetStatus(statelang.RU)
	inst.SetStartedTime(time.Now())
	inst.SetUpdatedTime(time.Now())
	return inst
}

func mockStateMachineConfig(context process_ctrl.ProcessContext) *stubConfig {
	cfg := newStubConfig()
	context.SetVariable(constant.VarNameStateMachineConfig, cfg)
	return cfg
}

var onceRegisterRM sync.Once
var globalFakeRM = &fakeResourceManager{}

type stubConfig struct {
	transTimeout             int
	sagaBranchRegisterEnable bool
	rmReportSuccessEnable    bool
	seqGenerator             sequence.SeqGenerator
	componentLock            *sync.Mutex
	repository               *stubStateMachineRepository
}

func newStubConfig() *stubConfig {
	onceRegisterRM.Do(func() {
		rmpkg.GetRmCacheInstance().RegisterResourceManager(globalFakeRM)
	})

	return &stubConfig{
		transTimeout:             60000,
		sagaBranchRegisterEnable: true,
		rmReportSuccessEnable:    true,
		seqGenerator:             sequence.NewUUIDSeqGenerator(),
		componentLock:            &sync.Mutex{},
		repository:               newStubStateMachineRepository(),
	}
}

type stubStateMachineRepository struct {
	byNameTenant map[string]statelang.StateMachine
	byID         map[string]statelang.StateMachine
}

func newStubStateMachineRepository() *stubStateMachineRepository {
	return &stubStateMachineRepository{
		byNameTenant: make(map[string]statelang.StateMachine),
		byID:         make(map[string]statelang.StateMachine),
	}
}

func (s *stubStateMachineRepository) key(name, tenant string) string {
	return name + "_" + tenant
}

func (s *stubStateMachineRepository) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	return s.byID[stateMachineId], nil
}

func (s *stubStateMachineRepository) GetStateMachineByNameAndTenantId(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return s.byNameTenant[s.key(stateMachineName, tenantId)], nil
}

func (s *stubStateMachineRepository) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return s.byNameTenant[s.key(stateMachineName, tenantId)], nil
}

func (s *stubStateMachineRepository) RegistryStateMachine(machine statelang.StateMachine) error {
	key := s.key(machine.Name(), machine.TenantId())
	s.byNameTenant[key] = machine
	if machine.ID() != "" {
		s.byID[machine.ID()] = machine
	}
	return nil
}

func (s *stubStateMachineRepository) RegistryStateMachineByReader(reader io.Reader) error {
	return nil
}

type fakeResourceManager struct {
	branchRegisterCalls int
	branchReportCalls   int
	branchRegisterErr   error
	branchReportErr     error
	nextBranchID        int64
	mu                  sync.Mutex
}

func (f *fakeResourceManager) BranchRegister(ctx context.Context, param rmpkg.BranchRegisterParam) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.branchRegisterCalls++
	if f.nextBranchID == 0 {
		f.nextBranchID = 101
	}
	return f.nextBranchID, f.branchRegisterErr
}

func (f *fakeResourceManager) BranchReport(ctx context.Context, param rmpkg.BranchReportParam) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.branchReportCalls++
	return f.branchReportErr
}

func (f *fakeResourceManager) LockQuery(ctx context.Context, param rmpkg.LockQueryParam) (bool, error) {
	return false, nil
}

func (f *fakeResourceManager) BranchCommit(ctx context.Context, resource rmpkg.BranchResource) (branch.BranchStatus, error) {
	return branch.BranchStatusPhasetwoCommitted, nil
}

func (f *fakeResourceManager) BranchRollback(ctx context.Context, resource rmpkg.BranchResource) (branch.BranchStatus, error) {
	return branch.BranchStatusPhasetwoRollbacked, nil
}

func (f *fakeResourceManager) RegisterResource(resource rmpkg.Resource) error {
	return nil
}

func (f *fakeResourceManager) UnregisterResource(resource rmpkg.Resource) error {
	return nil
}

func (f *fakeResourceManager) GetCachedResources() *sync.Map {
	return &sync.Map{}
}

func (f *fakeResourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeSAGA
}

func (s *stubConfig) StateLogRepository() repo.StateLogRepository { return nil }

func (s *stubConfig) StateMachineRepository() repo.StateMachineRepository { return s.repository }

func (s *stubConfig) StateLogStore() storepkg.StateLogStore { return nil }

func (s *stubConfig) StateLangStore() storepkg.StateLangStore { return nil }

func (s *stubConfig) ExpressionFactoryManager() *expr.ExpressionFactoryManager { return nil }

func (s *stubConfig) ExpressionResolver() expr.ExpressionResolver { return nil }

func (s *stubConfig) SeqGenerator() sequence.SeqGenerator { return s.seqGenerator }

func (s *stubConfig) StatusDecisionStrategy() engine.StatusDecisionStrategy {
	return strategy.NewDefaultStatusDecisionStrategy()
}

func (s *stubConfig) EventPublisher() process_ctrl.EventPublisher { return nil }

func (s *stubConfig) AsyncEventPublisher() process_ctrl.EventPublisher { return nil }

func (s *stubConfig) EnableAsync() bool { return false }

func (s *stubConfig) ServiceInvokerManager() invoker.ServiceInvokerManager { return nil }

func (s *stubConfig) ScriptInvokerManager() invoker.ScriptInvokerManager { return nil }

func (s *stubConfig) CharSet() string { return "UTF-8" }

func (s *stubConfig) GetDefaultTenantId() string { return "000001" }

func (s *stubConfig) GetTransOperationTimeout() int { return s.transTimeout }

func (s *stubConfig) GetServiceInvokeTimeout() int { return 60000 }

func (s *stubConfig) ComponentLock() *sync.Mutex { return s.componentLock }

func (s *stubConfig) RegisterStateMachineDef(resources []string) error { return nil }

func (s *stubConfig) RegisterExpressionFactory(expressionType string, factory expr.ExpressionFactory) {
}

func (s *stubConfig) RegisterServiceInvoker(serviceType string, invoker invoker.ServiceInvoker) {}

func (s *stubConfig) GetExpressionFactory(expressionType string) expr.ExpressionFactory { return nil }

func (s *stubConfig) GetServiceInvoker(serviceType string) (invoker.ServiceInvoker, error) {
	return nil, errors.New("not implemented")
}

func (s *stubConfig) IsSagaBranchRegisterEnable() bool { return s.sagaBranchRegisterEnable }

func (s *stubConfig) IsRmReportSuccessEnable() bool { return s.rmReportSuccessEnable }

func (s *stubConfig) SetSagaBranchRegisterEnable(enable bool) { s.sagaBranchRegisterEnable = enable }

func (s *stubConfig) SetRmReportSuccessEnable(enable bool) { s.rmReportSuccessEnable = enable }

type fakeSagaTemplate struct {
	branchRegisterCalls int
	branchReportCalls   int
	nextBranchID        int64
}

func (f *fakeSagaTemplate) CommitTransaction(ctx context.Context, gtr *tm.GlobalTransaction) error {
	return nil
}

func (f *fakeSagaTemplate) RollbackTransaction(ctx context.Context, gtr *tm.GlobalTransaction) error {
	return nil
}

func (f *fakeSagaTemplate) BeginTransaction(ctx context.Context, timeout time.Duration, txName string) (*tm.GlobalTransaction, error) {
	return &tm.GlobalTransaction{}, nil
}

func (f *fakeSagaTemplate) ReloadTransaction(ctx context.Context, xid string) (*tm.GlobalTransaction, error) {
	return &tm.GlobalTransaction{Xid: xid}, nil
}

func (f *fakeSagaTemplate) ReportTransaction(ctx context.Context, gtr *tm.GlobalTransaction) error {
	return nil
}

func (f *fakeSagaTemplate) BranchRegister(ctx context.Context, resourceId string, clientId string, xid string, applicationData string, lockKeys string) (int64, error) {
	f.branchRegisterCalls++
	if f.nextBranchID == 0 {
		f.nextBranchID = 101
	}
	return f.nextBranchID, nil
}

func (f *fakeSagaTemplate) BranchReport(ctx context.Context, xid string, branchId int64, status branch.BranchStatus, applicationData string) error {
	f.branchReportCalls++
	return nil
}

func (f *fakeSagaTemplate) CleanUp(ctx context.Context) {}

func newServiceTaskState(machine statelang.StateMachineInstance) statelang.StateInstance {
	state := statelang.NewStateInstanceImpl()
	state.SetStateMachineInstance(machine)
	state.SetMachineInstanceID(machine.ID())
	state.SetName("ServiceTask1")
	state.SetType(constant.StateTypeServiceTask)
	state.SetServiceName("DemoService")
	state.SetServiceMethod("foo")
	state.SetServiceType("LOCAL")
	state.SetForUpdate(false)
	state.SetStartedTime(time.Now())
	state.SetStatus(statelang.RU)
	return state
}

func attachServiceTaskDefinition(machine statelang.StateMachineInstance, cfg *stubConfig, name string) {
	sm := machine.StateMachine()
	task := stateimpl.NewServiceTaskStateImpl()
	task.SetName(name)
	task.SetServiceName("DemoService")
	task.SetServiceMethod("foo")
	task.SetServiceType("LOCAL")
	sm.States()[name] = task
	sm.SetStartState(name)
	_ = cfg.repository.RegistryStateMachine(sm)
}

func TestStateLogStore_RecordStateMachineStarted(t *testing.T) {
	prepareCleanDB(t)

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")
	expected := mockMachineInstance(stateMachineName)
	expected.SetBusinessKey("test_started")
	ctx := mockProcessContext(stateMachineName, expected)
	mockStateMachineConfig(ctx)
	err := stateLogStore.RecordStateMachineStarted(context.Background(), expected, ctx)
	assert.Nil(t, err)
	actual, err := stateLogStore.GetStateMachineInstance(expected.ID())
	assert.Nil(t, err)
	assert.Equal(t, expected.ID(), actual.ID())
	assert.Equal(t, expected.MachineID(), actual.MachineID())
	assert.Equal(t, fmt.Sprint(expected.StartParams()), fmt.Sprint(actual.StartParams()))
	assert.Nil(t, actual.Exception())
	assert.Nil(t, actual.SerializedError())
	assert.Equal(t, expected.Status(), actual.Status())
	assert.Equal(t, expected.StartedTime().UnixNano(), actual.StartedTime().UnixNano())
	assert.Equal(t, expected.UpdatedTime().UnixNano(), actual.UpdatedTime().UnixNano())
}

func prepareCleanDB(t *testing.T) {
	prepareDB()
	if db == nil {
		return
	}
	_, err := db.Exec("DELETE FROM seata_state_inst")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM seata_state_machine_inst")
	require.NoError(t, err)
}

func TestStateLogStore_RecordStateMachineFinished(t *testing.T) {
	prepareCleanDB(t)

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")
	expected := mockMachineInstance(stateMachineName)
	expected.SetBusinessKey("test_finished")
	ctx := mockProcessContext(stateMachineName, expected)
	mockStateMachineConfig(ctx)
	err := stateLogStore.RecordStateMachineStarted(context.Background(), expected, ctx)
	assert.Nil(t, err)
	expected.SetEndParams(map[string]any{"end": 100})
	expected.SetException(errors.New("this is a test error"))
	expected.SetStatus(statelang.FA)
	expected.SetEndTime(time.Now())
	expected.SetRunning(false)
	err = stateLogStore.RecordStateMachineFinished(context.Background(), expected, ctx)
	assert.Nil(t, err)
	assert.Equal(t, "{\"end\":100}", expected.SerializedEndParams())
	assert.NotEmpty(t, expected.SerializedError())
	actual, err := stateLogStore.GetStateMachineInstance(expected.ID())
	assert.Nil(t, err)

	assert.Equal(t, expected.ID(), actual.ID())
	assert.Equal(t, expected.MachineID(), actual.MachineID())
	assert.Equal(t, fmt.Sprint(expected.StartParams()), fmt.Sprint(actual.StartParams()))
	assert.Equal(t, "this is a test error", actual.Exception().Error())
	assert.Equal(t, expected.Status(), actual.Status())
	assert.Equal(t, expected.IsRunning(), actual.IsRunning())
	assert.Equal(t, expected.StartedTime().UnixNano(), actual.StartedTime().UnixNano())
	assert.False(t, actual.UpdatedTime().IsZero())
	assert.True(t, actual.UpdatedTime().UnixNano() >= actual.StartedTime().UnixNano())
	assert.False(t, expected.EndTime().IsZero())
}

func TestStateLogStore_RecordStateMachineRestarted(t *testing.T) {
	prepareCleanDB(t)

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")
	expected := mockMachineInstance(stateMachineName)
	expected.SetBusinessKey("test_restarted")
	ctx := mockProcessContext(stateMachineName, expected)
	mockStateMachineConfig(ctx)
	err := stateLogStore.RecordStateMachineStarted(context.Background(), expected, ctx)
	assert.Nil(t, err)
	expected.SetRunning(false)
	err = stateLogStore.RecordStateMachineFinished(context.Background(), expected, ctx)
	assert.Nil(t, err)

	actual, err := stateLogStore.GetStateMachineInstance(expected.ID())
	assert.Nil(t, err)
	assert.False(t, actual.IsRunning())

	actual.SetRunning(true)
	err = stateLogStore.RecordStateMachineRestarted(context.Background(), actual, ctx)
	assert.Nil(t, err)
	actual, err = stateLogStore.GetStateMachineInstance(actual.ID())
	assert.Nil(t, err)
	assert.True(t, actual.IsRunning())
}

func TestStateLogStore_RecordStateStarted(t *testing.T) {
	prepareCleanDB(t)

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")
	stateLogStore.sagaTransactionalTemplate = &fakeSagaTemplate{}

	machineInstance := mockMachineInstance("stateMachine")
	ctx := mockProcessContext(stateMachineName, machineInstance)
	_ = mockStateMachineConfig(ctx)
	cfg := mockStateMachineConfig(ctx)
	attachServiceTaskDefinition(machineInstance, cfg, "ServiceTask1")
	machineInstance.SetID("test")

	common := statelang.NewStateInstanceImpl()
	common.SetStateMachineInstance(machineInstance)
	common.SetMachineInstanceID(machineInstance.ID())
	common.SetName("ServiceTask1")
	common.SetType("ServiceTask")
	common.SetStartedTime(time.Now())
	common.SetServiceName("DemoService")
	common.SetServiceMethod("foo")
	common.SetServiceType("RPC")
	common.SetForUpdate(false)
	common.SetInputParams(map[string]string{"input": "test"})
	common.SetStatus(statelang.RU)
	common.SetBusinessKey("test_state_started")

	origin := statelang.NewStateInstanceImpl()
	origin.SetID("origin")
	origin.SetStateMachineInstance(machineInstance)
	origin.SetMachineInstanceID(machineInstance.ID())
	machineInstance.PutState("origin", origin)

	retried := statelang.NewStateInstanceImpl()
	retried.SetStateMachineInstance(machineInstance)
	retried.SetMachineInstanceID(machineInstance.ID())
	retried.SetID("origin.1")
	retried.SetStateIDRetriedFor("origin")

	compensated := statelang.NewStateInstanceImpl()
	compensated.SetStateMachineInstance(machineInstance)
	compensated.SetMachineInstanceID(machineInstance.ID())
	compensated.SetID("origin-1")
	compensated.SetStateIDCompensatedFor("origin")

	tests := []struct {
		name     string
		expected statelang.StateInstance
	}{
		{"common", common},
		{"retried", retried},
		{"compensated", compensated},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := stateLogStore.RecordStateStarted(context.Background(), test.expected, ctx)
			assert.Nil(t, err)
			actual, err := stateLogStore.GetStateInstance(test.expected.ID(), machineInstance.ID())
			assert.Nil(t, err)
			assert.Equal(t, test.expected.ID(), actual.ID())
			assert.Equal(t, test.expected.StateMachineInstance().ID(), actual.MachineInstanceID())
			assert.Equal(t, test.expected.Name(), actual.Name())
			assert.Equal(t, test.expected.Type(), actual.Type())
			assert.Equal(t, test.expected.StartedTime().UnixNano(), actual.StartedTime().UnixNano())
			assert.Equal(t, test.expected.ServiceName(), actual.ServiceName())
			assert.Equal(t, test.expected.ServiceMethod(), actual.ServiceMethod())
			assert.Equal(t, test.expected.ServiceType(), actual.ServiceType())
			assert.Equal(t, test.expected.IsForUpdate(), actual.IsForUpdate())
			assert.Equal(t, test.expected.SerializedInputParams(), actual.SerializedInputParams())
			assert.Equal(t, test.expected.Status(), actual.Status())
			assert.Equal(t, test.expected.BusinessKey(), actual.BusinessKey())
			assert.Equal(t, test.expected.StateIDCompensatedFor(), actual.StateIDCompensatedFor())
			assert.Equal(t, test.expected.StateIDRetriedFor(), actual.StateIDRetriedFor())
		})
	}
}

func TestRecordStateStartedSkipBranchRegister(t *testing.T) {
	prepareCleanDB(t)

	store := NewStateLogStore(db, "seata_")
	machine := mockMachineInstance("stateMachine")
	ctx := mockProcessContext("stateMachine", machine)
	cfg := mockStateMachineConfig(ctx)
	cfg.SetSagaBranchRegisterEnable(false)
	fakeTemplate := &fakeSagaTemplate{nextBranchID: 555}
	store.sagaTransactionalTemplate = fakeTemplate
	baseRegister := globalFakeRM.branchRegisterCalls

	attachServiceTaskDefinition(machine, cfg, "ServiceTask1")
	state := newServiceTaskState(machine)
	require.NoError(t, store.RecordStateStarted(context.Background(), state, ctx))
	require.Equal(t, baseRegister, globalFakeRM.branchRegisterCalls)
	require.NotEmpty(t, state.ID())
}

func TestRecordStateStartedTriggersBranchRegisterWhenEnabled(t *testing.T) {
	prepareCleanDB(t)

	store := NewStateLogStore(db, "seata_")
	machine := mockMachineInstance("stateMachine")
	machine.SetID("machine-branch-enable")
	ctx := mockProcessContext("stateMachine", machine)
	cfg := mockStateMachineConfig(ctx)
	cfg.SetSagaBranchRegisterEnable(true)
	fakeTemplate := &fakeSagaTemplate{nextBranchID: 777}
	store.sagaTransactionalTemplate = fakeTemplate
	baseRegister := globalFakeRM.branchRegisterCalls

	attachServiceTaskDefinition(machine, cfg, "ServiceTask1")
	state := newServiceTaskState(machine)
	require.NoError(t, store.RecordStateStarted(context.Background(), state, ctx))
	require.Equal(t, baseRegister+1, globalFakeRM.branchRegisterCalls)
	require.NotEmpty(t, state.ID())
	_, parseErr := strconv.ParseInt(state.ID(), 10, 64)
	require.NoError(t, parseErr)
}

func TestRecordStateStartedBranchRegisterError(t *testing.T) {
	prepareCleanDB(t)

	store := NewStateLogStore(db, "seata_")
	machine := mockMachineInstance("stateMachine")
	machine.SetID("machine-branch-error")
	ctx := mockProcessContext("stateMachine", machine)
	cfg := mockStateMachineConfig(ctx)
	cfg.SetSagaBranchRegisterEnable(true)
	fakeTemplate := &fakeSagaTemplate{nextBranchID: 888}
	store.sagaTransactionalTemplate = fakeTemplate
	globalFakeRM.mu.Lock()
	globalFakeRM.branchRegisterErr = errors.New("mock branch register error")
	globalFakeRM.nextBranchID = 0
	globalFakeRM.mu.Unlock()
	t.Cleanup(func() {
		globalFakeRM.mu.Lock()
		globalFakeRM.branchRegisterErr = nil
		globalFakeRM.nextBranchID = 0
		globalFakeRM.mu.Unlock()
	})

	attachServiceTaskDefinition(machine, cfg, "ServiceTask1")
	state := newServiceTaskState(machine)
	err := store.RecordStateStarted(context.Background(), state, ctx)
	require.Error(t, err)
	engErr, ok := engExc.IsEngineExecutionException(err)
	require.True(t, ok)
	require.Equal(t, seataErrors.TransactionErrorCodeBranchRegisterFailed, engErr.Code)
}

func TestRecordStateStartedDerivesCompensationIdFromHolder(t *testing.T) {
	prepareCleanDB(t)

	store := NewStateLogStore(db, "seata_")
	machine := mockMachineInstance("stateMachine")
	machine.SetID("machine-comp-holder")
	ctx := mockProcessContext("stateMachine", machine)
	cfg := mockStateMachineConfig(ctx)
	cfg.SetSagaBranchRegisterEnable(true)
	fakeTemplate := &fakeSagaTemplate{nextBranchID: 666}
	store.sagaTransactionalTemplate = fakeTemplate
	baseRegister := globalFakeRM.branchRegisterCalls

	original := newServiceTaskState(machine)
	original.SetName("ReduceInventory")
	original.SetID("origin-branch")
	machine.PutState(original.ID(), original)

	holder := pcext.NewCompensationHolder()
	holder.AddToBeCompensatedState("CompensateReduceInventory", original)
	ctx.SetVariable(constant.VarNameCurrentCompensationHolder, holder)

	attachServiceTaskDefinition(machine, cfg, "CompensateReduceInventory")
	compensate := newServiceTaskState(machine)
	compensate.SetName("CompensateReduceInventory")
	compensate.SetServiceName("inventoryAction")
	compensate.SetServiceMethod("CompensateReduce")
	compensate.SetServiceType("LOCAL")
	compensate.SetStartedTime(time.Now())

	require.NoError(t, store.RecordStateStarted(context.Background(), compensate, ctx))
	require.Equal(t, baseRegister, globalFakeRM.branchRegisterCalls)
	require.Equal(t, "origin-branch-1", compensate.ID())
	require.Equal(t, "origin-branch", compensate.StateIDCompensatedFor())

	stored, err := store.GetStateInstance(compensate.ID(), machine.ID())
	require.NoError(t, err)
	require.Equal(t, compensate.ID(), stored.ID())
	require.Equal(t, "origin-branch", stored.StateIDCompensatedFor())
}

func TestRecordStateFinishedSkipBranchReportOnSuccessWhenDisabled(t *testing.T) {
	prepareCleanDB(t)

	store := NewStateLogStore(db, "seata_")
	machine := mockMachineInstance("stateMachine")
	machine.SetID("machine-report-disabled")
	ctx := mockProcessContext("stateMachine", machine)
	cfg := mockStateMachineConfig(ctx)
	cfg.SetSagaBranchRegisterEnable(true)
	cfg.SetRmReportSuccessEnable(false)
	fakeTemplate := &fakeSagaTemplate{nextBranchID: 888}
	store.sagaTransactionalTemplate = fakeTemplate
	baseRegister := globalFakeRM.branchRegisterCalls
	baseReport := globalFakeRM.branchReportCalls

	attachServiceTaskDefinition(machine, cfg, "ServiceTask1")
	state := newServiceTaskState(machine)
	require.NoError(t, store.RecordStateStarted(context.Background(), state, ctx))

	state.SetStatus(statelang.SU)
	require.NoError(t, store.RecordStateFinished(context.Background(), state, ctx))
	require.Equal(t, baseRegister+1, globalFakeRM.branchRegisterCalls)
	require.Equal(t, baseReport, globalFakeRM.branchReportCalls)
}

func TestRecordStateFinishedReportsWhenEnabled(t *testing.T) {
	prepareCleanDB(t)

	store := NewStateLogStore(db, "seata_")
	machine := mockMachineInstance("stateMachine")
	machine.SetID("machine-report-enabled")
	ctx := mockProcessContext("stateMachine", machine)
	cfg := mockStateMachineConfig(ctx)
	cfg.SetSagaBranchRegisterEnable(true)
	cfg.SetRmReportSuccessEnable(true)
	fakeTemplate := &fakeSagaTemplate{nextBranchID: 999}
	store.sagaTransactionalTemplate = fakeTemplate
	baseRegister := globalFakeRM.branchRegisterCalls
	baseReport := globalFakeRM.branchReportCalls

	attachServiceTaskDefinition(machine, cfg, "ServiceTask1")
	state := newServiceTaskState(machine)
	require.NoError(t, store.RecordStateStarted(context.Background(), state, ctx))

	state.SetStatus(statelang.SU)
	require.NoError(t, store.RecordStateFinished(context.Background(), state, ctx))
	require.Equal(t, baseRegister+1, globalFakeRM.branchRegisterCalls)
	require.Equal(t, baseReport+1, globalFakeRM.branchReportCalls)
}

func TestRecordStateFinishedPropagatesBranchReportError(t *testing.T) {
	prepareCleanDB(t)

	store := NewStateLogStore(db, "seata_")
	machine := mockMachineInstance("stateMachine")
	machine.SetID("machine-report-error")
	ctx := mockProcessContext("stateMachine", machine)
	cfg := mockStateMachineConfig(ctx)
	cfg.SetSagaBranchRegisterEnable(true)
	cfg.SetRmReportSuccessEnable(true)
	fakeTemplate := &fakeSagaTemplate{nextBranchID: 1001}
	store.sagaTransactionalTemplate = fakeTemplate
	baseRegister := globalFakeRM.branchRegisterCalls
	baseReport := globalFakeRM.branchReportCalls

	attachServiceTaskDefinition(machine, cfg, "ServiceTask1")
	state := newServiceTaskState(machine)
	require.NoError(t, store.RecordStateStarted(context.Background(), state, ctx))

	state.SetStatus(statelang.SU)
	globalFakeRM.mu.Lock()
	globalFakeRM.branchReportErr = errors.New("mock branch report error")
	globalFakeRM.mu.Unlock()
	t.Cleanup(func() {
		globalFakeRM.mu.Lock()
		globalFakeRM.branchReportErr = nil
		globalFakeRM.mu.Unlock()
	})

	err := store.RecordStateFinished(context.Background(), state, ctx)
	require.Error(t, err)
	engErr, ok := engExc.IsEngineExecutionException(err)
	require.True(t, ok)
	require.Equal(t, seataErrors.TransactionErrorCodeBranchReportFailed, engErr.Code)
	require.Equal(t, baseRegister+1, globalFakeRM.branchRegisterCalls)
	require.Equal(t, baseReport+1, globalFakeRM.branchReportCalls)
}

func TestRecordStateFinishedWithoutStartDoesNotInsertFallback(t *testing.T) {
	prepareCleanDB(t)

	store := NewStateLogStore(db, "seata_")
	machine := mockMachineInstance("stateMachine")
	machine.SetID("machine-finish-only")
	ctx := mockProcessContext("stateMachine", machine)
	cfg := mockStateMachineConfig(ctx)
	cfg.SetRmReportSuccessEnable(false)

	state := newServiceTaskState(machine)
	state.SetName("ReduceInventory")
	state.SetID("missing-start")
	state.SetStatus(statelang.SU)
	state.SetEndTime(time.Now())
	state.SetUpdatedTime(time.Now())

	err := store.RecordStateFinished(context.Background(), state, ctx)
	require.Error(t, err)
	engErr, ok := engExc.IsEngineExecutionException(err)
	require.True(t, ok)
	require.Equal(t, seataErrors.TransactionErrorCodeFailedWriteSession, engErr.Code)
	_, getErr := store.GetStateInstance(state.ID(), machine.ID())
	require.Error(t, getErr)
}

func TestStateLogStore_RecordStateFinished(t *testing.T) {
	prepareCleanDB(t)

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")
	stateLogStore.sagaTransactionalTemplate = &fakeSagaTemplate{}

	machineInstance := mockMachineInstance("stateMachine")
	ctx := mockProcessContext(stateMachineName, machineInstance)
	machineInstance.SetID("test")
	cfg := mockStateMachineConfig(ctx)
	attachServiceTaskDefinition(machineInstance, cfg, "ServiceTask1")

	state := newServiceTaskState(machineInstance)

	err := stateLogStore.RecordStateStarted(context.Background(), state, ctx)
	assert.Nil(t, err)

	state.SetStatus(statelang.UN)
	state.SetError(errors.New("this is a test error"))
	state.SetOutputParams(map[string]string{"output": "test"})
	err = stateLogStore.RecordStateFinished(context.Background(), state, ctx)
	assert.Nil(t, err)
	actual, err := stateLogStore.GetStateInstance(state.ID(), machineInstance.ID())
	assert.Nil(t, err)
	assert.Equal(t, state.Status(), actual.Status())
	assert.Equal(t, state.Error().Error(), actual.Error().Error())
	assert.NotEmpty(t, actual.OutputParams())
	assert.Equal(t, state.SerializedOutputParams(), actual.SerializedOutputParams())
}

func TestStateLogStore_GetStateMachineInstanceByBusinessKey(t *testing.T) {
	prepareCleanDB(t)

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")
	expected := mockMachineInstance(stateMachineName)
	expected.SetBusinessKey("test_business_key")
	expected.SetTenantID("000001")
	ctx := mockProcessContext(stateMachineName, expected)

	err := stateLogStore.RecordStateMachineStarted(context.Background(), expected, ctx)
	assert.Nil(t, err)
	actual, err := stateLogStore.GetStateMachineInstanceByBusinessKey(expected.BusinessKey(), expected.TenantID())
	assert.Nil(t, err)
	assert.Equal(t, expected.ID(), actual.ID())
	assert.Equal(t, expected.MachineID(), actual.MachineID())
	assert.Equal(t, fmt.Sprint(expected.StartParams()), fmt.Sprint(actual.StartParams()))
	assert.Nil(t, actual.Exception())
	assert.Nil(t, actual.SerializedError())
	assert.Equal(t, expected.Status(), actual.Status())
	assert.Equal(t, expected.StartedTime().UnixNano(), actual.StartedTime().UnixNano())
	assert.Equal(t, expected.UpdatedTime().UnixNano(), actual.UpdatedTime().UnixNano())
}

func TestStateLogStore_GetStateMachineInstanceByParentId(t *testing.T) {
	prepareCleanDB(t)

	const (
		stateMachineName = "stateMachine"
		parentId         = "parent"
	)
	stateLogStore := NewStateLogStore(db, "seata_")
	expected := mockMachineInstance(stateMachineName)
	expected.SetBusinessKey("test_parent_id")
	expected.SetParentID(parentId)
	ctx := mockProcessContext(stateMachineName, expected)

	err := stateLogStore.RecordStateMachineStarted(context.Background(), expected, ctx)
	assert.Nil(t, err)
	actualList, err := stateLogStore.GetStateMachineInstanceByParentId(parentId)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(actualList))
	actual := actualList[0]
	assert.Equal(t, expected.ID(), actual.ID())
	assert.Equal(t, expected.MachineID(), actual.MachineID())
	// no startParams, endParams and Exception
	assert.NotEqual(t, fmt.Sprint(expected.StartParams()), fmt.Sprint(actual.StartParams()))
	assert.Nil(t, actual.Exception())
	assert.Nil(t, actual.SerializedError())
	assert.Equal(t, expected.Status(), actual.Status())
	assert.Equal(t, expected.StartedTime().UnixNano(), actual.StartedTime().UnixNano())
	assert.Equal(t, expected.UpdatedTime().UnixNano(), actual.UpdatedTime().UnixNano())
}
