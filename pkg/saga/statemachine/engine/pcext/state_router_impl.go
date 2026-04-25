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

	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine/exception"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang"
	sagaState "seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang/state"
	seataErrors "seata.apache.org/seata-go/v2/pkg/util/errors"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type EndStateRouter struct {
}

func (e EndStateRouter) Route(ctx context.Context, processContext process_ctrl.ProcessContext, state statelang.State) (process_ctrl.Instruction, error) {
	// mark Fail end-state to influence final machine status decision
	if state != nil && state.Type() == constant.StateTypeFail {
		processContext.SetVariable(constant.VarNameFailEndStateFlag, true)
	}
	return nil, nil
}

type TaskStateRouter struct {
}

func (t TaskStateRouter) Route(ctx context.Context, processContext process_ctrl.ProcessContext, state statelang.State) (process_ctrl.Instruction, error) {
	stateInstruction, err := GetStateInstruction(processContext)
	if err != nil {
		return nil, err
	}
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

	// If current state is CompensationTrigger or compensation has been flagged, start compensation routing
	if state != nil && state.Type() == constant.StateTypeCompensationTrigger {
		// flag into context for downstream calls (best-effort)
		if hpc, ok := processContext.(process_ctrl.HierarchicalProcessContext); ok {
			hpc.SetVariableLocally(constant.VarNameCurrentCompensateTriggerState, state)
			hpc.SetVariableLocally(constant.VarNameFirstCompensationStateStarted, false)
		} else {
			processContext.SetVariable(constant.VarNameCurrentCompensateTriggerState, state)
			processContext.SetVariable(constant.VarNameFirstCompensationStateStarted, false)
		}

		// build compensation stack from executed forward states (latest first)
		smInst := processContext.GetVariable(constant.VarNameStateMachineInst).(*statelang.StateMachineInstance)
		holder := GetCurrentCompensationHolder(ctx, processContext, true)
		stack := holder.StateStackNeedCompensation()
		sm := processContext.GetVariable(constant.VarNameStateMachine).(statelang.StateMachine)
		states := smInst.StateList()
		BuildCompensationStack(states, sm, stack)
		// fallback: if nothing pushed, try last successful forward state
		if stack.Empty() {
			for i := len(states) - 1; i >= 0; i-- {
				si := states[i]
				if si.StateIDCompensatedFor != "" {
					continue
				}
				if si.Status != statelang.SU {
					continue
				}
				stack.Push(si)
				break
			}
		}

		// mark machine compensation running
		smInst.CompensationStatus = statelang.RU

		return t.compensateRoute(ctx, processContext, state)
	}
	if compensationTriggerState, ok := processContext.GetVariable(constant.VarNameCurrentCompensateTriggerState).(statelang.State); ok {
		return t.compensateRoute(ctx, processContext, compensationTriggerState)
	}

	// There is an exception route, indicating that an exception is thrown, and the exception route is prioritized.
	var next string
	if v, ok := processContext.GetVariable(constant.VarNameCurrentExceptionRoute).(string); ok {
		next = v
	}

	if next != "" {
		processContext.RemoveVariable(constant.VarNameCurrentExceptionRoute)
	} else {
		next = state.Next()
	}

	// If next is empty, the state selected by the Choice state was taken.
	if next == "" && processContext.HasVariable(constant.VarNameCurrentChoice) {
		if v, ok := processContext.GetVariable(constant.VarNameCurrentChoice).(string); ok {
			next = v
		}
		processContext.RemoveVariable(constant.VarNameCurrentChoice)
	}

	if next == "" {
		return nil, nil
	}

	// If we are routing due to exception to CompensationTrigger, pre-build compensation stack
	if next == "CompensationTrigger" || (processContext.GetVariable(constant.VarNameCurrentException) != nil) {
		smInst := processContext.GetVariable(constant.VarNameStateMachineInst).(*statelang.StateMachineInstance)
		holder := GetCurrentCompensationHolder(ctx, processContext, true)
		stack := holder.StateStackNeedCompensation()
		sm := processContext.GetVariable(constant.VarNameStateMachine).(statelang.StateMachine)
		// prefer DB list if available
		var states []*statelang.StateInstance
		if cfg, ok := processContext.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig); ok && cfg.StateLogRepository() != nil {
			if list, err := cfg.StateLogRepository().GetStateInstanceListByMachineInstanceId(smInst.ID); err == nil && len(list) > 0 {
				states = list
			}
		}
		if len(states) == 0 {
			states = smInst.StateList()
		}
		BuildCompensationStack(states, sm, stack)
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

		stateInstance := processContext.GetVariable(constant.VarNameStateInst).(*statelang.StateInstance)
		if stateInstance != nil && statelang.SU != stateInstance.Status {
			return nil, EndStateMachine(ctx, processContext)
		}
	}

	stateStackToBeCompensated := GetCurrentCompensationHolder(ctx, processContext, true).StateStackNeedCompensation()
	if stateStackToBeCompensated != nil {
		popped := stateStackToBeCompensated.Pop()
		if popped == nil {
			// no states to compensate, finish or go next
			processContext.RemoveVariable(constant.VarNameCurrentCompensateTriggerState)
			// mark this as a no-compensation path; final status decided by Fail end
			processContext.SetVariable(constant.VarNameNoCompensation, true)
			compensationTriggerStateNext := compensationTriggerState.Next()
			if compensationTriggerStateNext == "" {
				return nil, EndStateMachine(ctx, processContext)
			}
			// set next on instruction (pointer-safe)
			instPtr, err := GetStateInstruction(processContext)
			if err != nil {
				return nil, EndStateMachine(ctx, processContext)
			}
			instPtr.SetStateName(compensationTriggerStateNext)
			processContext.SetInstruction(instPtr)
			return instPtr, nil
		}
		stateToBeCompensated := popped.(*statelang.StateInstance)

		stateMachine := processContext.GetVariable(constant.VarNameStateMachine).(statelang.StateMachine)
		state := stateMachine.State(GetOriginStateName(stateToBeCompensated))
		// resolve underlying abstract task for various state impls
		taskState := ExtractAbstractTaskState(state)
		if taskState != nil {
			// pointer-safe fetch of instruction
			instruction, err := GetStateInstruction(processContext)
			if err != nil {
				return nil, EndStateMachine(ctx, processContext)
			}

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

			// expose the forward state id to be compensated so handler can mark the compensation instance
			processContext.SetVariable("_compensate_for_state_id_", stateToBeCompensated.ID)

			if _, ok := compensateState.(sagaState.CompensateSubStateMachineState); ok {
				hierarchicalProcessContext = processContext.(process_ctrl.HierarchicalProcessContext)
				hierarchicalProcessContext.SetVariableLocally(
					compensateState.Name()+constant.VarNameSubMachineParentId,
					GenerateParentId(stateToBeCompensated))
			}

			processContext.SetInstruction(instruction)
			return instruction, nil
		}
	}

	processContext.RemoveVariable(constant.VarNameCurrentCompensateTriggerState)

	compensationTriggerStateNext := compensationTriggerState.Next()
	if compensationTriggerStateNext == "" {
		return nil, EndStateMachine(ctx, processContext)
	}

	instruction := processContext.GetInstruction()
	if si, ok := instruction.(*StateInstruction); ok {
		si.SetStateName(compensationTriggerStateNext)
		return si, nil
	}
	if siv, ok := instruction.(StateInstruction); ok {
		tmp := siv
		tmp.SetStateName(compensationTriggerStateNext)
		return &tmp, nil
	}
	return nil, exception.NewEngineExecutionException(seataErrors.ObjectNotExists, "instruction is not a state instruction", nil)
}
