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
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
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
	// mark Fail end-state to influence final machine status decision
	if state != nil && state.Type() == constant.StateTypeFail {
		processContext.SetVariable(constant.VarNameFailEndStateFlag, true)
	}
	return nil, nil
}

type TaskStateRouter struct {
}

func (t TaskStateRouter) Route(ctx context.Context, processContext process_ctrl.ProcessContext, state statelang.State) (process_ctrl.Instruction, error) {
	var stateInstruction *StateInstruction
	switch v := processContext.GetInstruction().(type) {
	case *StateInstruction:
		stateInstruction = v
	case StateInstruction:
		tmp := v
		stateInstruction = &tmp
	default:
		return nil, exception.NewEngineExecutionException(seataErrors.ObjectNotExists, "instruction is not a state instruction", nil)
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
		smInst := processContext.GetVariable(constant.VarNameStateMachineInst).(statelang.StateMachineInstance)
		holder := GetCurrentCompensationHolder(ctx, processContext, true)
		stack := holder.StateStackNeedCompensation()
		sm := processContext.GetVariable(constant.VarNameStateMachine).(statelang.StateMachine)
		states := smInst.StateList()
		for i := len(states) - 1; i >= 0; i-- {
			si := states[i]
			// exclude compensation states
			if si.StateIDCompensatedFor() != "" {
				continue
			}
			// only successful forward states are subject to compensation
			if si.Status() != statelang.SU {
				continue
			}
			originName := GetOriginStateName(si)
			def := sm.State(originName)
			// ensure the definition has a compensate state
			var task *sagaState.AbstractTaskState
			switch s := def.(type) {
			case *sagaState.ServiceTaskStateImpl:
				task = s.AbstractTaskState
			case *sagaState.ScriptTaskStateImpl:
				task = s.AbstractTaskState
			case *sagaState.SubStateMachineImpl:
				if s.ServiceTaskStateImpl != nil {
					task = s.ServiceTaskStateImpl.AbstractTaskState
				}
			}
			if task == nil || task.CompensateState() == "" {
				continue
			}
			stack.Push(si)
		}
		// fallback: if nothing pushed, try last successful forward state
		if stack.Empty() {
			for i := len(states) - 1; i >= 0; i-- {
				si := states[i]
				if si.StateIDCompensatedFor() != "" {
					continue
				}
				if si.Status() != statelang.SU {
					continue
				}
				stack.Push(si)
				break
			}
		}

		// mark machine compensation running
		smInst.SetCompensationStatus(statelang.RU)

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
		smInst := processContext.GetVariable(constant.VarNameStateMachineInst).(statelang.StateMachineInstance)
		holder := GetCurrentCompensationHolder(ctx, processContext, true)
		stack := holder.StateStackNeedCompensation()
		sm := processContext.GetVariable(constant.VarNameStateMachine).(statelang.StateMachine)
		// prefer DB list if available
		var states []statelang.StateInstance
		if cfg, ok := processContext.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig); ok && cfg.StateLogRepository() != nil {
			if list, err := cfg.StateLogRepository().GetStateInstanceListByMachineInstanceId(smInst.ID()); err == nil && len(list) > 0 {
				states = list
			}
		}
		if len(states) == 0 {
			states = smInst.StateList()
		}
		for i := len(states) - 1; i >= 0; i-- {
			si := states[i]
			if si.StateIDCompensatedFor() != "" {
				continue
			}
			if si.Status() != statelang.SU {
				continue
			}
			originName := GetOriginStateName(si)
			def := sm.State(originName)
			var task *sagaState.AbstractTaskState
			switch s := def.(type) {
			case *sagaState.ServiceTaskStateImpl:
				task = s.AbstractTaskState
			case *sagaState.ScriptTaskStateImpl:
				task = s.AbstractTaskState
			case *sagaState.SubStateMachineImpl:
				if s.ServiceTaskStateImpl != nil {
					task = s.ServiceTaskStateImpl.AbstractTaskState
				}
			}
			if task == nil || task.CompensateState() == "" {
				continue
			}
			stack.Push(si)
		}
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
			var instPtr *StateInstruction
			switch v := processContext.GetInstruction().(type) {
			case *StateInstruction:
				instPtr = v
			case StateInstruction:
				tmp := v
				instPtr = &tmp
			default:
				return nil, EndStateMachine(ctx, processContext)
			}
			instPtr.SetStateName(compensationTriggerStateNext)
			processContext.SetInstruction(instPtr)
			return instPtr, nil
		}
		stateToBeCompensated := popped.(statelang.StateInstance)

		stateMachine := processContext.GetVariable(constant.VarNameStateMachine).(statelang.StateMachine)
		state := stateMachine.State(GetOriginStateName(stateToBeCompensated))
		// resolve underlying abstract task for various state impls
		var taskState *sagaState.AbstractTaskState
		switch s := state.(type) {
		case *sagaState.ServiceTaskStateImpl:
			taskState = s.AbstractTaskState
		case *sagaState.ScriptTaskStateImpl:
			taskState = s.AbstractTaskState
		case *sagaState.SubStateMachineImpl:
			if s.ServiceTaskStateImpl != nil {
				taskState = s.ServiceTaskStateImpl.AbstractTaskState
			}
		}
		if taskState != nil {
			// pointer-safe fetch of instruction
			var instruction *StateInstruction
			switch v := processContext.GetInstruction().(type) {
			case *StateInstruction:
				instruction = v
			case StateInstruction:
				tmp := v
				instruction = &tmp
			default:
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
			processContext.SetVariable("_compensate_for_state_id_", stateToBeCompensated.ID())

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
