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

package repository

import (
	"context"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/store"
)

var (
	stateLogRepositoryImpl *StateLogRepositoryImpl
)

type StateLogRepositoryImpl struct {
	stateLogStore store.StateLogStore
}

func NewStateLogRepositoryImpl(stateLogStore store.StateLogStore) *StateLogRepositoryImpl {
	if stateLogRepositoryImpl == nil {
		stateLogRepositoryImpl = &StateLogRepositoryImpl{
			stateLogStore: stateLogStore,
		}
	}
	return stateLogRepositoryImpl
}

func (s *StateLogRepositoryImpl) RecordStateMachineStarted(
	ctx context.Context,
	machineInstance statelang.StateMachineInstance,
	processContext process_ctrl.ProcessContext,
) error {
	if s.stateLogStore == nil {
		return errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.RecordStateMachineStarted(ctx, machineInstance, processContext)
}

func (s *StateLogRepositoryImpl) RecordStateMachineFinished(
	ctx context.Context,
	machineInstance statelang.StateMachineInstance,
	processContext process_ctrl.ProcessContext,
) error {
	if s.stateLogStore == nil {
		return errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.RecordStateMachineFinished(ctx, machineInstance, processContext)
}

func (s *StateLogRepositoryImpl) RecordStateMachineRestarted(
	ctx context.Context,
	machineInstance statelang.StateMachineInstance,
	processContext process_ctrl.ProcessContext,
) error {
	if s.stateLogStore == nil {
		return errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.RecordStateMachineRestarted(ctx, machineInstance, processContext)
}

func (s *StateLogRepositoryImpl) RecordStateStarted(
	ctx context.Context,
	stateInstance statelang.StateInstance,
	processContext process_ctrl.ProcessContext,
) error {
	if s.stateLogStore == nil {
		return errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.RecordStateStarted(ctx, stateInstance, processContext)
}

func (s *StateLogRepositoryImpl) RecordStateFinished(
	ctx context.Context,
	stateInstance statelang.StateInstance,
	processContext process_ctrl.ProcessContext,
) error {
	if s.stateLogStore == nil {
		return errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.RecordStateFinished(ctx, stateInstance, processContext)
}

func (s *StateLogRepositoryImpl) GetStateMachineInstance(stateMachineInstanceId string) (statelang.StateMachineInstance, error) {
	if s.stateLogStore == nil {
		return nil, errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.GetStateMachineInstance(stateMachineInstanceId)
}

func (s *StateLogRepositoryImpl) GetStateMachineInstanceByBusinessKey(businessKey, tenantId string) (statelang.StateMachineInstance, error) {
	if s.stateLogStore == nil {
		return nil, errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.GetStateMachineInstanceByBusinessKey(businessKey, tenantId)
}

func (s *StateLogRepositoryImpl) QueryStateMachineInstanceByParentId(parentId string) ([]statelang.StateMachineInstance, error) {
	if s.stateLogStore == nil {
		return nil, errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.GetStateMachineInstanceByParentId(parentId)
}

func (s *StateLogRepositoryImpl) GetStateInstance(stateInstanceId, machineInstId string) (statelang.StateInstance, error) {
	if s.stateLogStore == nil {
		return nil, errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.GetStateInstance(stateInstanceId, machineInstId)
}

func (s *StateLogRepositoryImpl) QueryStateInstanceListByMachineInstanceId(stateMachineInstanceId string) ([]statelang.StateInstance, error) {
	if s.stateLogStore == nil {
		return nil, errors.New("stateLogStore is not initialized")
	}
	return s.stateLogStore.GetStateInstanceListByMachineInstanceId(stateMachineInstanceId)

}

func (s *StateLogRepositoryImpl) SetStateLogStore(stateLogStore store.StateLogStore) {
	s.stateLogStore = stateLogStore
}
