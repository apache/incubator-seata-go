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
	"errors"
	"sync"

	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine/expr"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang"
	stateimpl "seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang/state"
)

type StateHandler interface {
	State() string
	process_ctrl.ProcessHandler
}

type InterceptAbleStateHandler interface {
	StateHandler
	StateHandlerInterceptorList() []StateHandlerInterceptor
	RegistryStateHandlerInterceptor(stateHandlerInterceptor StateHandlerInterceptor)
}

type StateHandlerInterceptor interface {
	PreProcess(ctx context.Context, processContext process_ctrl.ProcessContext) error
	PostProcess(ctx context.Context, processContext process_ctrl.ProcessContext) error
	Match(stateType string) bool
}

type StateMachineProcessHandler struct {
	mp map[string]StateHandler
	mu sync.RWMutex
}

func NewStateMachineProcessHandler() *StateMachineProcessHandler {
	return &StateMachineProcessHandler{
		mp: make(map[string]StateHandler),
	}
}

func (s *StateMachineProcessHandler) Process(ctx context.Context, processContext process_ctrl.ProcessContext) error {
	var stateInstruction *StateInstruction
	switch v := processContext.GetInstruction().(type) {
	case *StateInstruction:
		stateInstruction = v
	case StateInstruction:
		tmp := v
		stateInstruction = &tmp
	default:
		return errors.New("invalid state instruction from processContext")
	}

	state, err := stateInstruction.GetState(processContext)
	if err != nil {
		return err
	}

	// mark Fail end-state to influence final machine status decision
	if state.Type() == constant.StateTypeFail {
		processContext.SetVariable(constant.VarNameFailEndStateFlag, true)
	}

	stateType := state.Type()
	stateHandler := s.GetStateHandler(stateType)

	interceptAbleStateHandler, ok := stateHandler.(InterceptAbleStateHandler)

	var stateHandlerInterceptorList []StateHandlerInterceptor
	if ok {
		stateHandlerInterceptorList = interceptAbleStateHandler.StateHandlerInterceptorList()
	}

	if stateHandlerInterceptorList != nil && len(stateHandlerInterceptorList) > 0 {
		for _, stateHandlerInterceptor := range stateHandlerInterceptorList {
			err = stateHandlerInterceptor.PreProcess(ctx, processContext)
			if err != nil {
				return err
			}
		}
	}

	// Prepare current state instance in context before processing
	smInst, ok := processContext.GetVariable(constant.VarNameStateMachineInst).(statelang.StateMachineInstance)
	if !ok || smInst == nil {
		return errors.New("state machine instance not found in context")
	}

	stInst := statelang.NewStateInstanceImpl()
	stInst.SetName(state.Name())
	stInst.SetType(state.Type())
	// If service task, enrich service attributes
	if svc, ok := state.(*stateimpl.ServiceTaskStateImpl); ok {
		stInst.SetServiceName(svc.ServiceName())
		stInst.SetServiceMethod(svc.ServiceMethod())
		stInst.SetServiceType(svc.ServiceType())
		stInst.SetForUpdate(svc.ForUpdate())

		// ensure mutex lock for parameter evaluation exists
		if !processContext.HasVariable(constant.VarNameProcessContextMutexLock) {
			processContext.SetVariable(constant.VarNameProcessContextMutexLock, &sync.Mutex{})
		}
		// if this is a compensation execution, mark the link to original state
		if v := processContext.GetVariable("_compensate_for_state_id_"); v != nil {
			if sid, ok := v.(string); ok && sid != "" {
				stInst.SetStateIDCompensatedFor(sid)
				// also add to holder's 'StatesForCompensation' for final status decision
				holder := GetCurrentCompensationHolder(ctx, processContext, true)
				holder.StatesForCompensation().Store(stInst.Name(), stInst)
				// clear after consuming
				processContext.RemoveVariable("_compensate_for_state_id_")
			}
		}

		// evaluate input params from CEL expressions if any
		if cfg, ok := processContext.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig); ok {
			var exprResolver expr.ExpressionResolver = cfg.ExpressionResolver()
			// use all variables in process context for expression scope
			variables := processContext.GetVariables()
			inputParams := CreateInputParams(processContext, exprResolver, stInst, svc.AbstractTaskState, variables)
			processContext.SetVariable(constant.VarNameInputParams, inputParams)
		}
	}
	// Assign a temporary ID key for list/map tracking
	tmpId := state.Name()
	if tmpId == "" {
		tmpId = "state"
	}
	smInst.PutState(tmpId, stInst)
	processContext.SetVariable(constant.VarNameStateInst, stInst)

	// Execute handler when present; end states don't require a handler
	if stateHandler != nil {
		err = stateHandler.Process(ctx, processContext)
		if err != nil {
			return err
		}
	}

	if stateHandlerInterceptorList != nil && len(stateHandlerInterceptorList) > 0 {
		for _, stateHandlerInterceptor := range stateHandlerInterceptorList {
			err = stateHandlerInterceptor.PostProcess(ctx, processContext)
			if err != nil {
				return err
			}
		}
	}

	// Set execution result on state instance
	if ex, _ := processContext.GetVariable(constant.VarNameCurrentException).(error); ex != nil {
		stInst.SetStatus(statelang.FA)
		stInst.SetError(ex)
	} else {
		// For Fail end state, mark FA; otherwise mark SU
		if stateType == constant.StateTypeFail {
			stInst.SetStatus(statelang.FA)
		} else {
			stInst.SetStatus(statelang.SU)
		}
	}

	return nil
}

func (s *StateMachineProcessHandler) GetStateHandler(stateType string) StateHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mp[stateType]
}

func (s *StateMachineProcessHandler) RegistryStateHandler(stateType string, stateHandler StateHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mp == nil {
		s.mp = make(map[string]StateHandler)
	}
	s.mp[stateType] = stateHandler
}
