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

package core

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/exception"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl/process"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	seataErrors "github.com/seata/seata-go/pkg/util/errors"
	"github.com/seata/seata-go/pkg/util/log"
	"time"
)

type ProcessCtrlStateMachineEngine struct {
	StateMachineConfig StateMachineConfig
}

func NewProcessCtrlStateMachineEngine() *ProcessCtrlStateMachineEngine {
	return &ProcessCtrlStateMachineEngine{
		StateMachineConfig: NewDefaultStateMachineConfig(),
	}
}

func (p ProcessCtrlStateMachineEngine) Start(ctx context.Context, stateMachineName string, tenantId string,
	startParams map[string]interface{}) (statelang.StateMachineInstance, error) {
	return p.startInternal(ctx, stateMachineName, tenantId, "", startParams, false, nil)
}

func (p ProcessCtrlStateMachineEngine) StartAsync(ctx context.Context, stateMachineName string, tenantId string,
	startParams map[string]interface{}, callback CallBack) (statelang.StateMachineInstance, error) {
	return p.startInternal(ctx, stateMachineName, tenantId, "", startParams, true, callback)
}

func (p ProcessCtrlStateMachineEngine) StartWithBusinessKey(ctx context.Context, stateMachineName string,
	tenantId string, businessKey string, startParams map[string]interface{}) (statelang.StateMachineInstance, error) {
	return p.startInternal(ctx, stateMachineName, tenantId, businessKey, startParams, false, nil)
}

func (p ProcessCtrlStateMachineEngine) StartWithBusinessKeyAsync(ctx context.Context, stateMachineName string,
	tenantId string, businessKey string, startParams map[string]interface{}, callback CallBack) (statelang.StateMachineInstance, error) {
	return p.startInternal(ctx, stateMachineName, tenantId, businessKey, startParams, true, callback)
}

func (p ProcessCtrlStateMachineEngine) Forward(ctx context.Context, stateMachineInstId string,
	replaceParams map[string]interface{}) (statelang.StateMachineInstance, error) {
	return p.forwardInternal(ctx, stateMachineInstId, replaceParams, false, false, nil)
}

func (p ProcessCtrlStateMachineEngine) ForwardAsync(ctx context.Context, stateMachineInstId string, replaceParams map[string]interface{}, callback CallBack) (statelang.StateMachineInstance, error) {
	return p.forwardInternal(ctx, stateMachineInstId, replaceParams, false, true, callback)
}

func (p ProcessCtrlStateMachineEngine) Compensate(ctx context.Context, stateMachineInstId string,
	replaceParams map[string]any) (statelang.StateMachineInstance, error) {
	return p.compensateInternal(ctx, stateMachineInstId, replaceParams, false, nil)
}

func (p ProcessCtrlStateMachineEngine) CompensateAsync(ctx context.Context, stateMachineInstId string, replaceParams map[string]interface{}, callback CallBack) (statelang.StateMachineInstance, error) {
	return p.compensateInternal(ctx, stateMachineInstId, replaceParams, true, callback)
}

func (p ProcessCtrlStateMachineEngine) SkipAndForward(ctx context.Context, stateMachineInstId string, replaceParams map[string]interface{}) (statelang.StateMachineInstance, error) {
	return p.forwardInternal(ctx, stateMachineInstId, replaceParams, true, false, nil)
}

func (p ProcessCtrlStateMachineEngine) SkipAndForwardAsync(ctx context.Context, stateMachineInstId string, callback CallBack) (statelang.StateMachineInstance, error) {
	return p.forwardInternal(ctx, stateMachineInstId, nil, true, true, callback)
}

func (p ProcessCtrlStateMachineEngine) GetStateMachineConfig() StateMachineConfig {
	return p.StateMachineConfig
}

func (p ProcessCtrlStateMachineEngine) ReloadStateMachineInstance(ctx context.Context, instId string) (statelang.StateMachineInstance, error) {
	inst, err := p.StateMachineConfig.StateLogStore().GetStateMachineInstance(instId)
	if err != nil {
		return nil, err
	}
	if inst != nil {
		stateMachine := inst.StateMachine()
		if stateMachine == nil {
			stateMachine, err = p.StateMachineConfig.StateMachineRepository().GetStateMachineById(inst.MachineID())
			if err != nil {
				return nil, err
			}
			inst.SetStateMachine(stateMachine)
		}
		if stateMachine == nil {
			return nil, exception.NewEngineExecutionException(seataErrors.ObjectNotExists,
				"StateMachine[id:"+inst.MachineID()+"] not exist.", nil)
		}

		stateList := inst.StateList()
		if len(stateList) == 0 {
			stateList, err = p.StateMachineConfig.StateLogStore().GetStateInstanceListByMachineInstanceId(instId)
			if err != nil {
				return nil, err
			}
			if len(stateList) > 0 {
				for _, tmpStateInstance := range stateList {
					inst.PutState(tmpStateInstance.ID(), tmpStateInstance)
				}
			}
		}

		if len(inst.EndParams()) == 0 {
			endParams, err := p.replayContextVariables(ctx, inst)
			if err != nil {
				return nil, err
			}
			inst.SetEndParams(endParams)
		}
	}
	return inst, nil
}

func (p ProcessCtrlStateMachineEngine) startInternal(ctx context.Context, stateMachineName string, tenantId string,
	businessKey string, startParams map[string]interface{}, async bool, callback CallBack) (statelang.StateMachineInstance, error) {
	if tenantId == "" {
		tenantId = p.StateMachineConfig.DefaultTenantId()
	}

	stateMachineInstance, err := p.createMachineInstance(stateMachineName, tenantId, businessKey, startParams)
	if err != nil {
		return nil, err
	}

	// Build the process_ctrl context.
	processContextBuilder := NewProcessContextBuilder().
		WithProcessType(process.StateLang).
		WithOperationName(constant.OperationNameStart).
		WithAsyncCallback(callback).
		WithInstruction(NewStateInstruction(stateMachineName, tenantId)).
		WithStateMachineInstance(stateMachineInstance).
		WithStateMachineConfig(p.StateMachineConfig).
		WithStateMachineEngine(p).
		WithIsAsyncExecution(async)

	contextMap := p.copyMap(startParams)

	stateMachineInstance.SetContext(contextMap)

	processContext := processContextBuilder.WithStateMachineContextVariables(contextMap).Build()

	if stateMachineInstance.StateMachine().IsPersist() && p.StateMachineConfig.StateLogStore() != nil {
		err := p.StateMachineConfig.StateLogStore().RecordStateMachineStarted(ctx, stateMachineInstance, processContext)
		if err != nil {
			return nil, err
		}
	}

	if stateMachineInstance.ID() == "" {
		stateMachineInstance.SetID(p.StateMachineConfig.SeqGenerator().GenerateId(constant.SeqEntityStateMachineInst, ""))
	}

	var eventPublisher EventPublisher
	if async {
		eventPublisher = p.StateMachineConfig.AsyncEventPublisher()
	} else {
		eventPublisher = p.StateMachineConfig.EventPublisher()
	}

	_, err = eventPublisher.PushEvent(ctx, processContext)
	if err != nil {
		return nil, err
	}

	return stateMachineInstance, nil
}

func (p ProcessCtrlStateMachineEngine) forwardInternal(ctx context.Context, stateMachineInstId string,
	replaceParams map[string]interface{}, skip bool, async bool, callback CallBack) (statelang.StateMachineInstance, error) {
	stateMachineInstance, err := p.reloadStateMachineInstance(ctx, stateMachineInstId)
	if err != nil {
		return nil, err
	}

	if stateMachineInstance == nil {
		return nil, exception.NewEngineExecutionException(seataErrors.StateMachineInstanceNotExists, "StateMachineInstance is not exists", nil)
	}

	if stateMachineInstance.Status() == statelang.SU && stateMachineInstance.CompensationStatus() == "" {
		return stateMachineInstance, nil
	}

	acceptStatus := []statelang.ExecutionStatus{statelang.FA, statelang.UN, statelang.RU}
	if _, err := p.checkStatus(ctx, stateMachineInstance, acceptStatus, nil, stateMachineInstance.Status(), "", "forward"); err != nil {
		return nil, err
	}

	actList := stateMachineInstance.StateList()
	if len(actList) == 0 {
		return nil, exception.NewEngineExecutionException(seataErrors.OperationDenied,
			fmt.Sprintf("StateMachineInstance[id:%s] has no stateInstance, please start a new StateMachine execution instead", stateMachineInstId), nil)
	}

	lastForwardState, err := p.findOutLastForwardStateInstance(actList)
	if err != nil {
		return nil, err
	}
	if lastForwardState == nil {
		return nil, exception.NewEngineExecutionException(seataErrors.OperationDenied,
			fmt.Sprintf("StateMachineInstance[id:%s] Cannot find last forward execution stateInstance", stateMachineInstId), nil)
	}

	contextBuilder := NewProcessContextBuilder().
		WithProcessType(process.StateLang).
		WithOperationName(constant.OperationNameForward).
		WithAsyncCallback(callback).
		WithStateMachineInstance(stateMachineInstance).
		WithStateInstance(lastForwardState).
		WithStateMachineConfig(p.StateMachineConfig).
		WithStateMachineEngine(p).
		WithIsAsyncExecution(async)

	context := contextBuilder.Build()

	contextVariables, err := p.getStateMachineContextVariables(ctx, stateMachineInstance)
	if err != nil {
		return nil, err
	}

	if replaceParams != nil {
		for k, v := range replaceParams {
			contextVariables[k] = v
		}
	}
	p.putBusinesskeyToContextariables(stateMachineInstance, contextVariables)

	concurrentContextVariables := p.copyMap(contextVariables)

	context.SetVariable(constant.VarNameStateMachineContext, concurrentContextVariables)
	stateMachineInstance.SetContext(concurrentContextVariables)

	originStateName := GetOriginStateName(lastForwardState)
	lastState := stateMachineInstance.StateMachine().State(originStateName)
	loop := GetLoopConfig(ctx, context, lastState)
	if loop != nil && lastForwardState.Status() == statelang.SU {
		lastForwardState = p.findOutLastNeedForwardStateInstance(ctx, context)
	}

	context.SetVariable(lastForwardState.Name()+constant.VarNameRetriedStateInstId, lastForwardState.ID())
	if lastForwardState.Type() == constant.StateTypeSubStateMachine && lastForwardState.CompensationStatus() != statelang.SU {
		context.SetVariable(constant.VarNameIsForSubStatMachineForward, true)
	}

	if lastForwardState.Status() != statelang.SU {
		lastForwardState.SetIgnoreStatus(true)
	}

	inst := NewStateInstruction(stateMachineInstance.StateMachine().Name(), stateMachineInstance.TenantID())
	if skip || lastForwardState.Status() == statelang.SU {
		next := ""
		curState := stateMachineInstance.StateMachine().State(GetOriginStateName(lastForwardState))
		if taskState, ok := curState.(*state.AbstractTaskState); ok {
			next = taskState.Next()
		}
		if next == "" {
			log.Warn(fmt.Sprintf("Last Forward execution StateInstance was succeed, and it has not Next State, skip forward operation"))
			return stateMachineInstance, nil
		}
		inst.SetStateName(next)
	} else {
		if lastForwardState.Status() == statelang.RU && !IsTimeout(lastForwardState.StartedTime(), p.StateMachineConfig.ServiceInvokeTimeout()) {
			return nil, exception.NewEngineExecutionException(seataErrors.OperationDenied,
				fmt.Sprintf("State [%s] is running, operation[forward] denied", lastForwardState.Name()), nil)
		}
		inst.SetStateName(GetOriginStateName(lastForwardState))
	}
	context.SetInstruction(inst)

	stateMachineInstance.SetStatus(statelang.RU)
	stateMachineInstance.SetRunning(true)

	log.Info(fmt.Sprintf("Operation [forward] started  stateMachineInstance[id:%s]", stateMachineInstance.ID()))

	if stateMachineInstance.StateMachine().IsPersist() {
		if err := p.StateMachineConfig.StateLogStore().RecordStateMachineRestarted(ctx, stateMachineInstance, context); err != nil {
			return nil, err
		}
	}

	curState, err := inst.GetState(context)
	if err != nil {
		return nil, err
	}
	loop = GetLoopConfig(ctx, context, curState)
	if loop != nil {
		inst.SetTemporaryState(state.NewLoopStartStateImpl())
	}

	if async {
		if _, err := p.StateMachineConfig.AsyncEventPublisher().PushEvent(ctx, context); err != nil {
			return nil, err
		}
	} else {
		if _, err := p.StateMachineConfig.EventPublisher().PushEvent(ctx, context); err != nil {
			return nil, err
		}
	}

	return stateMachineInstance, nil
}

func (p ProcessCtrlStateMachineEngine) findOutLastForwardStateInstance(stateInstanceList []statelang.StateInstance) (statelang.StateInstance, error) {
	var lastForwardStateInstance statelang.StateInstance
	var err error
	for i := len(stateInstanceList) - 1; i >= 0; i-- {
		stateInstance := stateInstanceList[i]
		if !stateInstance.IsForCompensation() {
			if stateInstance.CompensationStatus() == statelang.SU {
				continue
			}

			if stateInstance.Type() == constant.StateTypeSubStateMachine {
				finalState := stateInstance
				for finalState.StateIDRetriedFor() != "" {
					if finalState, err = p.StateMachineConfig.StateLogStore().GetStateInstance(finalState.StateIDRetriedFor(),
						finalState.MachineInstanceID()); err != nil {
						return nil, err
					}
				}

				subInst, _ := p.StateMachineConfig.StateLogStore().GetStateMachineInstanceByParentId(GenerateParentId(finalState))
				if len(subInst) > 0 {
					if subInst[0].CompensationStatus() == statelang.SU {
						continue
					}

					if subInst[0].CompensationStatus() == statelang.UN {
						return nil, exception.NewEngineExecutionException(seataErrors.ForwardInvalid,
							"Last forward execution state instance is SubStateMachine and compensation status is [UN], Operation[forward] denied, stateInstanceId:"+stateInstance.ID(),
							nil)
					}
				}
			} else if stateInstance.CompensationStatus() == statelang.UN {
				return nil, exception.NewEngineExecutionException(seataErrors.ForwardInvalid,
					"Last forward execution state instance compensation status is [UN], Operation[forward] denied, stateInstanceId:"+stateInstance.ID(),
					nil)
			}

			lastForwardStateInstance = stateInstance
			break
		}
	}
	return lastForwardStateInstance, nil
}

// copyMap not deep copy, so best practice: Don’t pass by reference
func (p ProcessCtrlStateMachineEngine) copyMap(startParams map[string]interface{}) map[string]interface{} {
	copyMap := make(map[string]interface{}, len(startParams))
	for k, v := range startParams {
		copyMap[k] = v
	}
	return copyMap
}

func (p ProcessCtrlStateMachineEngine) createMachineInstance(stateMachineName string, tenantId string, businessKey string, startParams map[string]interface{}) (statelang.StateMachineInstance, error) {
	stateMachine, err := p.StateMachineConfig.StateMachineRepository().GetLastVersionStateMachine(stateMachineName, tenantId)
	if err != nil {
		return nil, err
	}

	if stateMachine == nil {
		return nil, errors.New("StateMachine [" + stateMachineName + "] is not exists")
	}

	stateMachineInstance := statelang.NewStateMachineInstanceImpl()
	stateMachineInstance.SetStateMachine(stateMachine)
	stateMachineInstance.SetTenantID(tenantId)
	stateMachineInstance.SetBusinessKey(businessKey)
	stateMachineInstance.SetStartParams(startParams)
	if startParams != nil {
		if businessKey != "" {
			startParams[constant.VarNameBusinesskey] = businessKey
		}

		if startParams[constant.VarNameParentId] != nil {
			parentId, ok := startParams[constant.VarNameParentId].(string)
			if !ok {

			}
			stateMachineInstance.SetParentID(parentId)
			delete(startParams, constant.VarNameParentId)
		}
	}

	stateMachineInstance.SetStatus(statelang.RU)
	stateMachineInstance.SetRunning(true)

	now := time.Now()
	stateMachineInstance.SetStartedTime(now)
	stateMachineInstance.SetUpdatedTime(now)
	return stateMachineInstance, nil
}

func (p ProcessCtrlStateMachineEngine) compensateInternal(ctx context.Context, stateMachineInstId string, replaceParams map[string]any,
	async bool, callback CallBack) (statelang.StateMachineInstance, error) {
	stateMachineInstance, err := p.reloadStateMachineInstance(ctx, stateMachineInstId)
	if err != nil {
		return nil, err
	}

	if stateMachineInstance == nil {
		return nil, exception.NewEngineExecutionException(seataErrors.StateMachineInstanceNotExists,
			"StateMachineInstance is not exits", nil)
	}

	if statelang.SU == stateMachineInstance.CompensationStatus() {
		return stateMachineInstance, nil
	}

	if stateMachineInstance.CompensationStatus() != "" {
		denyStatus := make([]statelang.ExecutionStatus, 0)
		denyStatus = append(denyStatus, statelang.SU)
		p.checkStatus(ctx, stateMachineInstance, nil, denyStatus, "", stateMachineInstance.CompensationStatus(),
			"compensate")
	}

	if replaceParams != nil {
		for key, value := range replaceParams {
			stateMachineInstance.EndParams()[key] = value
		}
	}

	contextBuilder := NewProcessContextBuilder().WithProcessType(process.StateLang).
		WithOperationName(constant.OperationNameCompensate).WithAsyncCallback(callback).
		WithStateMachineInstance(stateMachineInstance).
		WithStateMachineConfig(p.StateMachineConfig).WithStateMachineEngine(p).WithIsAsyncExecution(async)

	context := contextBuilder.Build()

	contextVariables, err := p.getStateMachineContextVariables(ctx, stateMachineInstance)

	if replaceParams != nil {
		for key, value := range replaceParams {
			contextVariables[key] = value
		}
	}

	p.putBusinesskeyToContextariables(stateMachineInstance, contextVariables)

	// TODO: Here is not use sync.map, make sure whether to use it
	concurrentContextVariables := make(map[string]any)
	p.nullSafeCopy(contextVariables, concurrentContextVariables)

	context.SetVariable(constant.VarNameStateMachineContext, concurrentContextVariables)
	stateMachineInstance.SetContext(concurrentContextVariables)

	tempCompensationTriggerState := state.NewCompensationTriggerStateImpl()
	tempCompensationTriggerState.SetStateMachine(stateMachineInstance.StateMachine())

	stateMachineInstance.SetRunning(true)

	log.Info("Operation [compensate] start.  stateMachineInstance[id:" + stateMachineInstance.ID() + "]")

	if stateMachineInstance.StateMachine().IsPersist() {
		err := p.StateMachineConfig.StateLogStore().RecordStateMachineRestarted(ctx, stateMachineInstance, context)
		if err != nil {
			return nil, err
		}
	}

	inst := NewStateInstruction(stateMachineInstance.TenantID(), stateMachineInstance.StateMachine().Name())
	inst.SetTemporaryState(tempCompensationTriggerState)
	context.SetInstruction(inst)

	if async {
		_, err := p.StateMachineConfig.AsyncEventPublisher().PushEvent(ctx, context)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := p.StateMachineConfig.EventPublisher().PushEvent(ctx, context)
		if err != nil {
			return nil, err
		}
	}

	return stateMachineInstance, nil
}

func (p ProcessCtrlStateMachineEngine) reloadStateMachineInstance(ctx context.Context, instId string) (statelang.StateMachineInstance, error) {
	instance, err := p.StateMachineConfig.StateLogStore().GetStateMachineInstance(instId)
	if err != nil {
		return nil, err
	}
	if instance != nil {
		stateMachine := instance.StateMachine()
		if stateMachine == nil {
			stateMachine, err = p.StateMachineConfig.StateMachineRepository().GetStateMachineById(instance.MachineID())
			if err != nil {
				return nil, err
			}
			instance.SetStateMachine(stateMachine)
		}
		if stateMachine == nil {
			return nil, exception.NewEngineExecutionException(seataErrors.ObjectNotExists,
				"StateMachine[id:"+instance.MachineID()+"] not exist.", nil)
		}

		stateList := instance.StateList()
		if stateList == nil || len(stateList) == 0 {
			stateList, err = p.StateMachineConfig.StateLogStore().GetStateInstanceListByMachineInstanceId(instId)
			if err != nil {
				return nil, err
			}
			if stateList != nil && len(stateList) > 0 {
				for _, tmpStateInstance := range stateList {
					instance.PutState(tmpStateInstance.ID(), tmpStateInstance)
				}
			}
		}

		if instance.EndParams() == nil || len(instance.EndParams()) == 0 {
			variables, err := p.replayContextVariables(ctx, instance)
			if err != nil {
				return nil, err
			}
			instance.SetEndParams(variables)
		}
	}
	return instance, nil
}

func (p ProcessCtrlStateMachineEngine) replayContextVariables(ctx context.Context, stateMachineInstance statelang.StateMachineInstance) (map[string]any, error) {
	contextVariables := make(map[string]any)
	if stateMachineInstance.StartParams() != nil {
		for key, value := range stateMachineInstance.StartParams() {
			contextVariables[key] = value
		}
	}

	stateInstanceList := stateMachineInstance.StateList()
	if stateInstanceList == nil || len(stateInstanceList) == 0 {
		return contextVariables, nil
	}

	for _, stateInstance := range stateInstanceList {
		serviceOutputParams := stateInstance.OutputParams()
		if serviceOutputParams != nil {
			serviceTaskStateImpl, ok := stateMachineInstance.StateMachine().State(GetOriginStateName(stateInstance)).(*state.ServiceTaskStateImpl)
			if !ok {
				return nil, exception.NewEngineExecutionException(seataErrors.ObjectNotExists,
					"Cannot find State by state name ["+stateInstance.Name()+"], may be this is a bug", nil)
			}

			if serviceTaskStateImpl.Output() != nil && len(serviceTaskStateImpl.Output()) != 0 {
				outputVariablesToContext, err := CreateOutputParams(p.StateMachineConfig,
					p.StateMachineConfig.ExpressionResolver(), serviceTaskStateImpl.AbstractTaskState, serviceOutputParams)
				if err != nil {
					return nil, exception.NewEngineExecutionException(seataErrors.ObjectNotExists,
						"Context variable replay failed", err)
				}
				if outputVariablesToContext != nil && len(outputVariablesToContext) != 0 {
					for key, value := range outputVariablesToContext {
						contextVariables[key] = value
					}
				}
				if len(stateInstance.BusinessKey()) > 0 {
					contextVariables[serviceTaskStateImpl.Name()+constant.VarNameBusinesskey] = stateInstance.BusinessKey()
				}
			}
		}
	}

	return contextVariables, nil
}

func (p ProcessCtrlStateMachineEngine) checkStatus(ctx context.Context, stateMachineInstance statelang.StateMachineInstance,
	acceptStatus []statelang.ExecutionStatus, denyStatus []statelang.ExecutionStatus, status statelang.ExecutionStatus,
	compenStatus statelang.ExecutionStatus, operation string) (bool, error) {
	if status != "" && compenStatus != "" {
		return false, exception.NewEngineExecutionException(seataErrors.InvalidParameter,
			"status and compensationStatus are not supported at the same time", nil)
	}
	if status == "" && compenStatus == "" {
		return false, exception.NewEngineExecutionException(seataErrors.InvalidParameter,
			"status and compensationStatus must input at least one", nil)
	}
	if statelang.SU == compenStatus {
		message := p.buildExceptionMessage(stateMachineInstance, nil, nil, "", statelang.SU, operation)
		return false, exception.NewEngineExecutionException(seataErrors.OperationDenied,
			message, nil)
	}

	if stateMachineInstance.IsRunning() &&
		!IsTimeout(stateMachineInstance.UpdatedTime(), p.StateMachineConfig.TransOperationTimeout()) {
		return false, exception.NewEngineExecutionException(seataErrors.OperationDenied,
			"StateMachineInstance [id:"+stateMachineInstance.ID()+"] is running, operation["+operation+
				"] denied", nil)
	}

	if (denyStatus == nil || len(denyStatus) == 0) && (acceptStatus == nil || len(acceptStatus) == 0) {
		return false, exception.NewEngineExecutionException(seataErrors.InvalidParameter,
			"StateMachineInstance[id:"+stateMachineInstance.ID()+
				"], acceptable status and deny status must input at least one", nil)
	}

	currentStatus := compenStatus
	if status != "" {
		currentStatus = status
	}

	if denyStatus != nil && len(denyStatus) == 0 {
		for _, tempDenyStatus := range denyStatus {
			if tempDenyStatus == currentStatus {
				message := p.buildExceptionMessage(stateMachineInstance, acceptStatus, denyStatus, status,
					compenStatus, operation)
				return false, exception.NewEngineExecutionException(seataErrors.OperationDenied,
					message, nil)
			}
		}
	}

	if acceptStatus == nil || len(acceptStatus) == 0 {
		return true, nil
	} else {
		for _, tempStatus := range acceptStatus {
			if tempStatus == currentStatus {
				return true, nil
			}
		}
	}

	message := p.buildExceptionMessage(stateMachineInstance, acceptStatus, denyStatus, status, compenStatus,
		operation)
	return false, exception.NewEngineExecutionException(seataErrors.OperationDenied,
		message, nil)
}

func (p ProcessCtrlStateMachineEngine) getStateMachineContextVariables(ctx context.Context,
	stateMachineInstance statelang.StateMachineInstance) (map[string]any, error) {
	contextVariables := stateMachineInstance.EndParams()
	if contextVariables == nil || len(contextVariables) == 0 {
		return p.replayContextVariables(ctx, stateMachineInstance)
	}
	return contextVariables, nil
}

func (p ProcessCtrlStateMachineEngine) buildExceptionMessage(instance statelang.StateMachineInstance,
	acceptStatus []statelang.ExecutionStatus, denyStatus []statelang.ExecutionStatus, status statelang.ExecutionStatus,
	compenStatus statelang.ExecutionStatus, operation string) string {
	message := fmt.Sprintf("StateMachineInstance[id:%s]", instance.ID())
	if len(acceptStatus) > 0 {
		message += ",acceptable status :"
		for _, tempStatus := range acceptStatus {
			message += string(tempStatus) + " "
		}
	}

	if len(denyStatus) > 0 {
		message += ",deny status:"
		for _, tempStatus := range denyStatus {
			message += string(tempStatus) + " "
		}
	}

	if status != "" {
		message += ",current status:" + string(status)
	}

	if compenStatus != "" {
		message += ",current compensation status:" + string(compenStatus)
	}

	message += fmt.Sprintf(",so operation [%s] denied", operation)
	return message
}

func (p ProcessCtrlStateMachineEngine) putBusinesskeyToContextariables(instance statelang.StateMachineInstance, variables map[string]any) {
	if instance.BusinessKey() != "" && variables[constant.VarNameBusinesskey] == "" {
		variables[constant.VarNameBusinesskey] = instance.BusinessKey()
	}
}

func (p ProcessCtrlStateMachineEngine) nullSafeCopy(srcMap map[string]any, destMap map[string]any) {
	for key, value := range srcMap {
		if value == nil {
			destMap[key] = value
		}
	}
}

func (p ProcessCtrlStateMachineEngine) findOutLastNeedForwardStateInstance(ctx context.Context, processContext ProcessContext) statelang.StateInstance {
	stateMachineInstance := processContext.GetVariable(constant.VarNameStateMachineInst).(statelang.StateMachineInstance)
	lastForwardState := processContext.GetVariable(constant.VarNameStateInst).(statelang.StateInstance)

	actList := stateMachineInstance.StateList()
	for i := len(actList) - 1; i >= 0; i-- {
		stateInstance := actList[i]
		if GetOriginStateName(stateInstance) == GetOriginStateName(lastForwardState) && stateInstance.Status() != statelang.SU {
			return stateInstance
		}
	}
	return lastForwardState
}
