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
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl/process"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/util/log"
)

type RouterHandler interface {
	Route(ctx context.Context, processContext ProcessContext) error
}

type ProcessRouter interface {
	Route(ctx context.Context, processContext ProcessContext) error
}

type InterceptAbleStateRouter interface {
	StateRouter
	StateRouterInterceptor() []StateRouterInterceptor
	RegistryStateRouterInterceptor(stateRouterInterceptor StateRouterInterceptor)
}

type StateRouter interface {
	Route(ctx context.Context, processContext ProcessContext, state statelang.State) (Instruction, error)
}

type StateRouterInterceptor interface {
	PreRoute(ctx context.Context, processContext ProcessContext, state statelang.State) error
	PostRoute(ctx context.Context, processContext ProcessContext, instruction Instruction, err error) error
	Match(stateType string) bool
}

type DefaultRouterHandler struct {
	eventPublisher EventPublisher
	processRouters map[string]ProcessRouter
}

func (d *DefaultRouterHandler) Route(ctx context.Context, processContext ProcessContext) error {
	processType := d.matchProcessType(ctx, processContext)
	if processType == "" {
		log.Warnf("Process type not found, context= %s", processContext)
		return errors.New("Process type not found")
	}

	processRouter := d.processRouters[string(processType)]
	if processRouter == nil {
		log.Errorf("Cannot find process router by type %s, context = %s", processType, processContext)
		return errors.New("Process router not found")
	}

	instruction := processRouter.Route(ctx, processContext)
	if instruction == nil {
		log.Info("route instruction is null, process end")
	} else {
		processContext.SetInstruction(instruction)
		_, err := d.eventPublisher.PushEvent(ctx, processContext)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DefaultRouterHandler) matchProcessType(ctx context.Context, processContext ProcessContext) process.ProcessType {
	processType, ok := processContext.GetVariable(constant.VarNameProcessType).(process.ProcessType)
	if !ok || processType == "" {
		processType = process.StateLang
	}
	return processType
}

func (d *DefaultRouterHandler) EventPublisher() EventPublisher {
	return d.eventPublisher
}

func (d *DefaultRouterHandler) SetEventPublisher(eventPublisher EventPublisher) {
	d.eventPublisher = eventPublisher
}

func (d *DefaultRouterHandler) ProcessRouters() map[string]ProcessRouter {
	return d.processRouters
}

func (d *DefaultRouterHandler) SetProcessRouters(processRouters map[string]ProcessRouter) {
	d.processRouters = processRouters
}

type StateMachineProcessRouter struct {
	stateRouters map[string]StateRouter
}

func (s *StateMachineProcessRouter) Route(ctx context.Context, processContext ProcessContext) (Instruction, error) {
	stateInstruction, ok := processContext.GetInstruction().(StateInstruction)
	if !ok {
		return nil, errors.New("instruction is not a state instruction")
	}

	var state statelang.State
	if stateInstruction.TemporaryState() != nil {
		state = stateInstruction.TemporaryState()
		stateInstruction.SetTemporaryState(nil)
	} else {
		stateMachineConfig, ok := processContext.GetVariable(constant.VarNameStateMachineConfig).(StateMachineConfig)
		if !ok {
			return nil, errors.New("state machine config not found")
		}

		stateMachine, err := stateMachineConfig.StateMachineRepository().GetStateMachineByNameAndTenantId(stateInstruction.StateMachineName(),
			stateInstruction.TenantId())
		if err != nil {
			return nil, err
		}

		state = stateMachine.States()[stateInstruction.StateName()]
	}

	stateType := state.Type()
	router := s.stateRouters[stateType]

	var interceptors []StateRouterInterceptor
	if interceptAbleStateRouter, ok := router.(InterceptAbleStateRouter); ok {
		interceptors = interceptAbleStateRouter.StateRouterInterceptor()
	}

	var executedInterceptors []StateRouterInterceptor
	var exception error
	instruction, exception := func() (Instruction, error) {
		if interceptors == nil || len(executedInterceptors) == 0 {
			executedInterceptors = make([]StateRouterInterceptor, 0, len(interceptors))
			for _, interceptor := range interceptors {
				executedInterceptors = append(executedInterceptors, interceptor)
				err := interceptor.PreRoute(ctx, processContext, state)
				if err != nil {
					return nil, err
				}
			}
		}

		instruction, err := router.Route(ctx, processContext, state)
		if err != nil {
			return nil, err
		}
		return instruction, nil
	}()

	if interceptors == nil || len(executedInterceptors) == 0 {
		for i := len(executedInterceptors) - 1; i >= 0; i-- {
			err := executedInterceptors[i].PostRoute(ctx, processContext, instruction, exception)
			if err != nil {
				return nil, err
			}
		}

		// if 'Succeed' or 'Fail' State did not configured, we must end the state machine
		if instruction == nil && !stateInstruction.End() {
			err := EndStateMachine(ctx, processContext)
			if err != nil {
				return nil, err
			}
		}
	}

	return instruction, nil
}

func (s *StateMachineProcessRouter) InitDefaultStateRouters() {
	if s.stateRouters == nil || len(s.stateRouters) == 0 {
		s.stateRouters = make(map[string]StateRouter)
		taskStateRouter := &TaskStateRouter{}
		s.stateRouters[constant.StateTypeServiceTask] = taskStateRouter
		s.stateRouters[constant.StateTypeScriptTask] = taskStateRouter
		s.stateRouters[constant.StateTypeChoice] = taskStateRouter
		s.stateRouters[constant.StateTypeCompensationTrigger] = taskStateRouter
		s.stateRouters[constant.StateTypeSubStateMachine] = taskStateRouter
		s.stateRouters[constant.StateTypeCompensateSubMachine] = taskStateRouter
		s.stateRouters[constant.StateTypeLoopStart] = taskStateRouter

		endStateRouter := &EndStateRouter{}
		s.stateRouters[constant.StateTypeSucceed] = endStateRouter
		s.stateRouters[constant.StateTypeFail] = endStateRouter
	}
}

func (s *StateMachineProcessRouter) StateRouters() map[string]StateRouter {
	return s.stateRouters
}

func (s *StateMachineProcessRouter) SetStateRouters(stateRouters map[string]StateRouter) {
	s.stateRouters = stateRouters
}
