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
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/core"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl/process"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func mockProcessContext(stateMachineName string, stateMachineInstance statelang.StateMachineInstance) core.ProcessContext {
	ctx := core.NewProcessContextBuilder().
		WithProcessType(process.StateLang).
		WithOperationName(constant.OperationNameStart).
		WithInstruction(core.NewStateInstruction(stateMachineName, "000001")).
		WithStateMachineInstance(stateMachineInstance).
		Build()
	return ctx
}

func mockMachineInstance(stateMachineName string) statelang.StateMachineInstance {
	stateMachine := statelang.NewStateMachineImpl()
	stateMachine.SetName(stateMachineName)
	stateMachine.SetComment("This is a test state machine")
	stateMachine.SetCreateTime(time.Now())

	inst := statelang.NewStateMachineInstanceImpl()
	inst.SetStateMachine(stateMachine)
	inst.SetMachineID(stateMachineName)

	inst.SetStartParams(map[string]any{"start": 100})
	inst.SetStatus(statelang.RU)
	inst.SetStartedTime(time.Now())
	inst.SetUpdatedTime(time.Now())
	return inst
}

func mockStateMachineConfig(context core.ProcessContext) core.StateMachineConfig {
	cfg := core.NewDefaultStateMachineConfig()
	context.SetVariable(constant.VarNameStateMachineConfig, cfg)
	return cfg
}

func TestStateLogStore_RecordStateMachineStarted(t *testing.T) {
	prepareDB()

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

func TestStateLogStore_RecordStateMachineFinished(t *testing.T) {
	prepareDB()

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")
	expected := mockMachineInstance(stateMachineName)
	expected.SetBusinessKey("test_finished")
	ctx := mockProcessContext(stateMachineName, expected)
	err := stateLogStore.RecordStateMachineStarted(context.Background(), expected, ctx)
	assert.Nil(t, err)
	expected.SetEndParams(map[string]any{"end": 100})
	expected.SetException(errors.New("this is a test error"))
	expected.SetStatus(statelang.FA)
	expected.SetEndTime(time.Now())
	expected.SetRunning(false)
	err = stateLogStore.RecordStateMachineFinished(context.Background(), expected, ctx)
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
	assert.Greater(t, actual.UpdatedTime().UnixNano(), expected.UpdatedTime().UnixNano())
	assert.False(t, expected.EndTime().IsZero())
}

func TestStateLogStore_RecordStateMachineRestarted(t *testing.T) {
	prepareDB()

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")
	expected := mockMachineInstance(stateMachineName)
	expected.SetBusinessKey("test_restarted")
	ctx := mockProcessContext(stateMachineName, expected)
	err := stateLogStore.RecordStateMachineStarted(context.Background(), expected, ctx)
	assert.Nil(t, err)
	expected.SetRunning(false)
	err = stateLogStore.RecordStateMachineFinished(context.Background(), expected, ctx)

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
	prepareDB()

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")

	machineInstance := mockMachineInstance("stateMachine")
	ctx := mockProcessContext(stateMachineName, machineInstance)
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

func TestStateLogStore_RecordStateFinished(t *testing.T) {
	prepareDB()

	const stateMachineName = "stateMachine"
	stateLogStore := NewStateLogStore(db, "seata_")

	machineInstance := mockMachineInstance("stateMachine")
	ctx := mockProcessContext(stateMachineName, machineInstance)
	machineInstance.SetID("test")

	expected := statelang.NewStateInstanceImpl()
	expected.SetStateMachineInstance(machineInstance)
	expected.SetMachineInstanceID(machineInstance.ID())

	err := stateLogStore.RecordStateStarted(context.Background(), expected, ctx)
	assert.Nil(t, err)

	expected.SetStatus(statelang.UN)
	expected.SetError(errors.New("this is a test error"))
	expected.SetOutputParams(map[string]string{"output": "test"})
	err = stateLogStore.RecordStateFinished(context.Background(), expected, ctx)
	assert.Nil(t, err)
	actual, err := stateLogStore.GetStateInstance(expected.ID(), machineInstance.ID())
	assert.Nil(t, err)
	assert.Equal(t, expected.Status(), actual.Status())
	assert.Equal(t, expected.Error().Error(), actual.Error().Error())
	assert.NotEmpty(t, actual.OutputParams())
	assert.Equal(t, expected.SerializedOutputParams(), actual.SerializedOutputParams())
}

func TestStateLogStore_GetStateMachineInstanceByBusinessKey(t *testing.T) {
	prepareDB()

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
	prepareDB()

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
