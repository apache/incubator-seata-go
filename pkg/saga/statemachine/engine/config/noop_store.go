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

package config

import (
	"context"

	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang"
)

// NoopStateLogStore is a no-op implementation of StateLogStore for out-of-the-box scenarios.
// All methods perform no actual operations and return nil or zero values to ensure validation passes.
type NoopStateLogStore struct{}

func (s *NoopStateLogStore) RecordStateMachineStarted(ctx context.Context, machineInstance statelang.StateMachineInstance, pc process_ctrl.ProcessContext) error {
	return nil
}

func (s *NoopStateLogStore) RecordStateMachineFinished(ctx context.Context, machineInstance statelang.StateMachineInstance, pc process_ctrl.ProcessContext) error {
	return nil
}

func (s *NoopStateLogStore) RecordStateMachineRestarted(ctx context.Context, machineInstance statelang.StateMachineInstance, pc process_ctrl.ProcessContext) error {
	return nil
}

func (s *NoopStateLogStore) RecordStateStarted(ctx context.Context, stateInstance statelang.StateInstance, pc process_ctrl.ProcessContext) error {
	return nil
}

func (s *NoopStateLogStore) RecordStateFinished(ctx context.Context, stateInstance statelang.StateInstance, pc process_ctrl.ProcessContext) error {
	return nil
}

func (s *NoopStateLogStore) GetStateMachineInstance(stateMachineInstanceId string) (statelang.StateMachineInstance, error) {
	return nil, nil
}

func (s *NoopStateLogStore) GetStateMachineInstanceByBusinessKey(businessKey string, tenantId string) (statelang.StateMachineInstance, error) {
	return nil, nil
}

func (s *NoopStateLogStore) GetStateMachineInstanceByParentId(parentId string) ([]statelang.StateMachineInstance, error) {
	return nil, nil
}

func (s *NoopStateLogStore) GetStateInstance(stateInstanceId string, stateMachineInstanceId string) (statelang.StateInstance, error) {
	return nil, nil
}

func (s *NoopStateLogStore) GetStateInstanceListByMachineInstanceId(stateMachineInstanceId string) ([]statelang.StateInstance, error) {
	return nil, nil
}

func (s *NoopStateLogStore) ClearUp(pc process_ctrl.ProcessContext) {
	// no-op
}

type NoopStateLangStore struct{}

func (s *NoopStateLangStore) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	return nil, nil
}

func (s *NoopStateLangStore) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return nil, nil
}

func (s *NoopStateLangStore) StoreStateMachine(stateMachine statelang.StateMachine) error {
	return nil
}
