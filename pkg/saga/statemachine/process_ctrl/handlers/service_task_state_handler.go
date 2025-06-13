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

package handlers

import (
	"context"
	"errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/exception"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/pcext"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	seataErrors "github.com/seata/seata-go/pkg/util/errors"
	"github.com/seata/seata-go/pkg/util/log"
)

type ServiceTaskStateHandler struct {
	interceptors []pcext.StateHandlerInterceptor
}

func NewServiceTaskStateHandler() *ServiceTaskStateHandler {
	return &ServiceTaskStateHandler{}
}

func (s *ServiceTaskStateHandler) State() string {
	return constant.StateTypeServiceTask
}

func (s *ServiceTaskStateHandler) Process(ctx context.Context, processContext process_ctrl.ProcessContext) error {
	stateInstruction, ok := processContext.GetInstruction().(pcext.StateInstruction)
	if !ok {
		return errors.New("invalid state instruction from processContext")
	}
	stateInterface, err := stateInstruction.GetState(processContext)
	if err != nil {
		return err
	}
	serviceTaskStateImpl, ok := stateInterface.(*state.ServiceTaskStateImpl)

	serviceName := serviceTaskStateImpl.ServiceName()
	methodName := serviceTaskStateImpl.ServiceMethod()
	stateInstance, ok := processContext.GetVariable(constant.VarNameStateInst).(statelang.StateInstance)
	if !ok {
		return errors.New("invalid state instance type from processContext")
	}

	// invoke service task and record
	var result any
	var resultErr error
	handleResultErr := func(err error) {
		log.Error("<<<<<<<<<<<<<<<<<<<<<< State[%s], ServiceName[%s], Method[%s] Execute failed.",
			serviceTaskStateImpl.Name(), serviceName, methodName, err)

		hierarchicalProcessContext, ok := processContext.(process_ctrl.HierarchicalProcessContext)
		if !ok {
			return
		}
		hierarchicalProcessContext.SetVariable(constant.VarNameCurrentException, err)
		pcext.HandleException(processContext, serviceTaskStateImpl.AbstractTaskState, err)
	}

	input, ok := processContext.GetVariable(constant.VarNameInputParams).([]any)
	if !ok {
		handleResultErr(errors.New("invalid input params type from processContext"))
		return nil
	}

	stateInstance.SetStatus(statelang.RU)
	log.Debugf(">>>>>>>>>>>>>>>>>>>>>> Start to execute State[%s], ServiceName[%s], Method[%s], Input:%s",
		serviceTaskStateImpl.Name(), serviceName, methodName, input)

	if _, ok := stateInterface.(state.CompensateSubStateMachineState); ok {
		// If it is the compensation of the subState machine,
		// directly call the state machine's compensate method
		stateMachineEngine, ok := processContext.GetVariable(constant.VarNameStateMachineEngine).(engine.StateMachineEngine)
		if !ok {
			handleResultErr(errors.New("invalid stateMachineEngine type from processContext"))
			return nil
		}

		result, resultErr = s.compensateSubStateMachine(ctx, processContext, serviceTaskStateImpl, input,
			stateInstance, stateMachineEngine)
		if resultErr != nil {
			handleResultErr(resultErr)
			return nil
		}
	} else {
		stateMachineConfig, ok := processContext.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig)
		if !ok {
			handleResultErr(errors.New("invalid stateMachineConfig type from processContext"))
			return nil
		}

		serviceInvoker := stateMachineConfig.ServiceInvokerManager().ServiceInvoker(serviceTaskStateImpl.ServiceType())
		if serviceInvoker == nil {
			resultErr = exception.NewEngineExecutionException(seataErrors.ObjectNotExists,
				"No such ServiceInvoker["+serviceTaskStateImpl.ServiceType()+"]", nil)
			handleResultErr(resultErr)
			return nil
		}

		result, resultErr = serviceInvoker.Invoke(ctx, input, serviceTaskStateImpl)
		if resultErr != nil {
			handleResultErr(resultErr)
			return nil
		}
	}

	log.Debugf("<<<<<<<<<<<<<<<<<<<<<< State[%s], ServiceName[%s], Method[%s] Execute finish. result: %s",
		serviceTaskStateImpl.Name(), serviceName, methodName, result)

	if result != nil {
		stateInstance.SetOutputParams(result)
		hierarchicalProcessContext, ok := processContext.(process_ctrl.HierarchicalProcessContext)
		if !ok {
			handleResultErr(errors.New("invalid hierarchical process context type from processContext"))
			return nil
		}

		hierarchicalProcessContext.SetVariable(constant.VarNameOutputParams, result)
	}

	return nil
}

func (s *ServiceTaskStateHandler) StateHandlerInterceptorList() []pcext.StateHandlerInterceptor {
	return s.interceptors
}

func (s *ServiceTaskStateHandler) RegistryStateHandlerInterceptor(stateHandlerInterceptor pcext.StateHandlerInterceptor) {
	s.interceptors = append(s.interceptors, stateHandlerInterceptor)
}

func (s *ServiceTaskStateHandler) compensateSubStateMachine(ctx context.Context, processContext process_ctrl.ProcessContext,
	serviceTaskState state.ServiceTaskState, input any, instance statelang.StateInstance,
	machineEngine engine.StateMachineEngine) (any, error) {
	subStateMachineParentId, ok := processContext.GetVariable(serviceTaskState.Name() + constant.VarNameSubMachineParentId).(string)
	if !ok {
		return nil, errors.New("invalid subStateMachineParentId type from processContext")
	}

	if subStateMachineParentId == "" {
		return nil, exception.NewEngineExecutionException(seataErrors.ObjectNotExists,
			"sub statemachine parentId is required", nil)
	}

	stateMachineConfig := processContext.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig)
	subInst, err := stateMachineConfig.StateLogStore().GetStateMachineInstanceByParentId(subStateMachineParentId)
	if err != nil {
		return nil, err
	}

	if subInst == nil || len(subInst) == 0 {
		return nil, exception.NewEngineExecutionException(seataErrors.ObjectNotExists,
			"cannot find sub statemachine instance by parentId:"+subStateMachineParentId, nil)
	}

	subStateMachineInstId := subInst[0].ID()
	log.Debugf(">>>>>>>>>>>>>>>>>>>>>> Start to compensate sub statemachine [id:%s]", subStateMachineInstId)

	startParams := make(map[string]any)

	if inputList, ok := input.([]any); ok {
		if len(inputList) > 0 {
			startParams = inputList[0].(map[string]any)
		}
	} else if inputMap, ok := input.(map[string]any); ok {
		startParams = inputMap
	}

	compensateInst, err := machineEngine.Compensate(ctx, subStateMachineInstId, startParams)
	instance.SetStatus(compensateInst.CompensationStatus())
	log.Debugf("<<<<<<<<<<<<<<<<<<<<<< Compensate sub statemachine [id:%s] finished with status[%s], "+"compensateState[%s]",
		subStateMachineInstId, compensateInst.Status(), compensateInst.CompensationStatus())
	return compensateInst.EndParams(), nil
}
