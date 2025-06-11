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

package pcext

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/exception"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	sagaState "github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	seataErrors "github.com/seata/seata-go/pkg/util/errors"
	"github.com/seata/seata-go/pkg/util/log"
)

type EndStateRouter struct {
}

func (e EndStateRouter) Route(ctx context.Context, processContext process_ctrl.ProcessContext, state statelang.State) (process_ctrl.Instruction, error) {
	return nil, nil
}

type TaskStateRouter struct {
}

func (t TaskStateRouter) Route(ctx context.Context, processContext process_ctrl.ProcessContext, state statelang.State) (process_ctrl.Instruction, error) {
	stateInstruction, _ := processContext.GetInstruction().(StateInstruction)
	if stateInstruction.End() {
		log.Infof("StateInstruction is ended, Stop the StateMachine executing. StateMachine[%s] Current State[%s]",
			stateInstruction.StateMachineName(), stateInstruction.StateName())
	}

	// check if in loop async condition
	isLoop, ok := processContext.GetVariable(constant.VarNameIsLoopState).(bool)
	if ok && isLoop {
		log.Infof("StateMachine[%s] Current State[%s] is in loop async condition, skip route processing.",
			stateInstruction.StateMachineName(), stateInstruction.StateName())
		return nil, nil
	}

	// The current CompensationTriggerState can mark the compensation process is started and perform compensation
	// route processing.
	compensationTriggerState, ok := processContext.GetVariable(constant.VarNameCurrentCompensateTriggerState).(statelang.State)
	if ok {
		return t.compensateRoute(ctx, processContext, compensationTriggerState)
	}

	// There is an exception route, indicating that an exception is thrown, and the exception route is prioritized.
	next := processContext.GetVariable(constant.VarNameCurrentExceptionRoute).(string)

	if next != "" {
		processContext.RemoveVariable(constant.VarNameCurrentExceptionRoute)
	} else {
		next = state.Next()
	}

	// If next is empty, the state selected by the Choice state was taken.
	if next == "" && processContext.HasVariable(constant.VarNameCurrentChoice) {
		next = processContext.GetVariable(constant.VarNameCurrentChoice).(string)
		processContext.RemoveVariable(constant.VarNameCurrentChoice)
	}

	if next == "" {
		return nil, nil
	}

	stateMachine := state.StateMachine()
	nextState := stateMachine.State(next)
	if nextState == nil {
		return nil, exception.NewEngineExecutionException(seataErrors.ObjectNotExists,
			"Next state["+next+"] is not exits", nil)
	}

	stateInstruction.SetStateName(next)

	if nil != GetLoopConfig(ctx, processContext, nextState) {
		stateInstruction.SetTemporaryState(sagaState.NewLoopStartStateImpl())
	}

	return stateInstruction, nil
}

func (t *TaskStateRouter) compensateRoute(ctx context.Context, processContext process_ctrl.ProcessContext,
	compensationTriggerState statelang.State) (process_ctrl.Instruction, error) {
	//If there is already a compensation state that has been executed,
	// it is judged whether it is wrong or unsuccessful,
	// and the compensation process is interrupted.
	isFirstCompensationStateStart := processContext.GetVariable(constant.VarNameFirstCompensationStateStarted).(bool)
	if isFirstCompensationStateStart {
		exception := processContext.GetVariable(constant.VarNameCurrentException).(error)
		if exception != nil {
			return nil, EndStateMachine(ctx, processContext)
		}

		stateInstance := processContext.GetVariable(constant.VarNameStateInst).(statelang.StateInstance)
		if stateInstance != nil && statelang.SU != stateInstance.Status() {
			return nil, EndStateMachine(ctx, processContext)
		}
	}

	stateStackToBeCompensated := GetCurrentCompensationHolder(ctx, processContext, true).StateStackNeedCompensation()
	if stateStackToBeCompensated != nil {
		stateToBeCompensated := stateStackToBeCompensated.Pop().(statelang.StateInstance)

		stateMachine := processContext.GetVariable(constant.VarNameStateMachine).(statelang.StateMachine)
		state := stateMachine.State(GetOriginStateName(stateToBeCompensated))
		if taskState, ok := state.(sagaState.AbstractTaskState); ok {
			instruction := processContext.GetInstruction().(StateInstruction)

			var compensateState statelang.State
			compensateStateName := taskState.CompensateState()
			if len(compensateStateName) != 0 {
				compensateState = stateMachine.State(compensateStateName)
			}

			if subStateMachine, ok := state.(sagaState.SubStateMachine); compensateState == nil && ok {
				compensateState = subStateMachine.CompensateStateImpl()
				instruction.SetTemporaryState(compensateState)
			}

			if compensateState == nil {
				return nil, EndStateMachine(ctx, processContext)
			}

			instruction.SetStateName(compensateState.Name())

			GetCurrentCompensationHolder(ctx, processContext, true).AddToBeCompensatedState(compensateState.Name(),
				stateToBeCompensated)

			hierarchicalProcessContext := processContext.(process_ctrl.HierarchicalProcessContext)
			hierarchicalProcessContext.SetVariableLocally(constant.VarNameFirstCompensationStateStarted, true)

			if _, ok := compensateState.(sagaState.CompensateSubStateMachineState); ok {
				hierarchicalProcessContext = processContext.(process_ctrl.HierarchicalProcessContext)
				hierarchicalProcessContext.SetVariableLocally(
					compensateState.Name()+constant.VarNameSubMachineParentId,
					GenerateParentId(stateToBeCompensated))
			}

			return instruction, nil
		}
	}

	processContext.RemoveVariable(constant.VarNameCurrentCompensateTriggerState)

	compensationTriggerStateNext := compensationTriggerState.Next()
	if compensationTriggerStateNext == "" {
		return nil, EndStateMachine(ctx, processContext)
	}

	instruction := processContext.GetInstruction().(StateInstruction)
	instruction.SetStateName(compensationTriggerStateNext)
	return instruction, nil
}
