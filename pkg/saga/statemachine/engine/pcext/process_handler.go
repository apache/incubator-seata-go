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
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"sync"
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
	stateInstruction, _ := processContext.GetInstruction().(StateInstruction)

	state, err := stateInstruction.GetState(processContext)
	if err != nil {
		return err
	}

	stateType := state.Type()
	stateHandler := s.GetStateHandler(stateType)
	if stateHandler == nil {
		return errors.New("Not support [" + stateType + "] state handler")
	}

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

	err = stateHandler.Process(ctx, processContext)
	if err != nil {
		return err
	}

	if stateHandlerInterceptorList != nil && len(stateHandlerInterceptorList) > 0 {
		for _, stateHandlerInterceptor := range stateHandlerInterceptorList {
			err = stateHandlerInterceptor.PostProcess(ctx, processContext)
			if err != nil {
				return err
			}
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
