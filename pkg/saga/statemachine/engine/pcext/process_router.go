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
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type StateMachineProcessRouter struct {
	stateRouters map[string]process_ctrl.StateRouter
}

func (s *StateMachineProcessRouter) Route(ctx context.Context, processContext process_ctrl.ProcessContext) (process_ctrl.Instruction, error) {
	stateInstruction, ok := processContext.GetInstruction().(StateInstruction)
	if !ok {
		return nil, errors.New("instruction is not a state instruction")
	}

	var state statelang.State
	if stateInstruction.TemporaryState() != nil {
		state = stateInstruction.TemporaryState()
		stateInstruction.SetTemporaryState(nil)
	} else {
		stateMachineConfig, ok := processContext.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig)
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

	var interceptors []process_ctrl.StateRouterInterceptor
	if interceptAbleStateRouter, ok := router.(process_ctrl.InterceptAbleStateRouter); ok {
		interceptors = interceptAbleStateRouter.StateRouterInterceptor()
	}

	var executedInterceptors []process_ctrl.StateRouterInterceptor
	var exception error
	instruction, exception := func() (process_ctrl.Instruction, error) {
		if interceptors == nil || len(executedInterceptors) == 0 {
			executedInterceptors = make([]process_ctrl.StateRouterInterceptor, 0, len(interceptors))
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
		s.stateRouters = make(map[string]process_ctrl.StateRouter)
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

func (s *StateMachineProcessRouter) StateRouters() map[string]process_ctrl.StateRouter {
	return s.stateRouters
}

func (s *StateMachineProcessRouter) SetStateRouters(stateRouters map[string]process_ctrl.StateRouter) {
	s.stateRouters = stateRouters
}
