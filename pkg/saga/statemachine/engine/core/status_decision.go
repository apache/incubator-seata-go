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
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/util/log"
)

type StatusDecisionStrategy interface {
	// DecideOnEndState Determine state machine execution status when executing to EndState
	DecideOnEndState(ctx context.Context, processContext ProcessContext,
		stateMachineInstance statelang.StateMachineInstance, exp error) error
	// DecideOnTaskStateFail Determine state machine execution status when executing TaskState error
	DecideOnTaskStateFail(ctx context.Context, processContext ProcessContext,
		stateMachineInstance statelang.StateMachineInstance, exp error) error
	// DecideMachineForwardExecutionStatus Determine the forward execution state of the state machine
	DecideMachineForwardExecutionStatus(ctx context.Context,
		stateMachineInstance statelang.StateMachineInstance, exp error, specialPolicy bool) error
}

type DefaultStatusDecisionStrategy struct {
}

func NewDefaultStatusDecisionStrategy() *DefaultStatusDecisionStrategy {
	return &DefaultStatusDecisionStrategy{}
}

func (d DefaultStatusDecisionStrategy) DecideOnEndState(ctx context.Context, processContext ProcessContext,
	stateMachineInstance statelang.StateMachineInstance, exp error) error {
	if statelang.RU == stateMachineInstance.CompensationStatus() {
		compensationHolder := GetCurrentCompensationHolder(ctx, processContext, true)
		if err := decideMachineCompensateStatus(ctx, stateMachineInstance, compensationHolder); err != nil {
			return err
		}
	} else {
		failEndStateFlag, ok := processContext.GetVariable(constant.VarNameFailEndStateFlag).(bool)
		if !ok {
			failEndStateFlag = false
		}
		if _, err := decideMachineForwardExecutionStatus(ctx, stateMachineInstance, exp, failEndStateFlag); err != nil {
			return err
		}
	}

	if stateMachineInstance.CompensationStatus() != "" && constant.OperationNameForward ==
		processContext.GetVariable(constant.VarNameOperationName).(string) && statelang.SU == stateMachineInstance.Status() {
		stateMachineInstance.SetCompensationStatus(statelang.FA)
	}

	log.Debugf("StateMachine Instance[id:%s,name:%s] execute finish with status[%s], compensation status [%s].",
		stateMachineInstance.ID(), stateMachineInstance.StateMachine().Name(),
		stateMachineInstance.Status(), stateMachineInstance.CompensationStatus())

	return nil
}

func decideMachineCompensateStatus(ctx context.Context, stateMachineInstance statelang.StateMachineInstance, compensationHolder *CompensationHolder) error {
	if stateMachineInstance.Status() == "" || statelang.RU == stateMachineInstance.Status() {
		stateMachineInstance.SetStatus(statelang.UN)
	}
	if !compensationHolder.StateStackNeedCompensation().Empty() {
		hasCompensateSUorUN := false
		compensationHolder.StatesForCompensation().Range(
			func(key, value any) bool {
				stateInstance, ok := value.(statelang.StateInstance)
				if !ok {
					return false
				}
				if statelang.UN == stateInstance.Status() || statelang.SU == stateInstance.Status() {
					hasCompensateSUorUN = true
					return true
				}
				return false
			})

		if hasCompensateSUorUN {
			stateMachineInstance.SetCompensationStatus(statelang.UN)
		} else {
			stateMachineInstance.SetCompensationStatus(statelang.FA)
		}
	} else {
		hasCompensateError := false
		compensationHolder.StatesForCompensation().Range(
			func(key, value any) bool {
				stateInstance, ok := value.(statelang.StateInstance)
				if !ok {
					return false
				}
				if statelang.SU != stateInstance.Status() {
					hasCompensateError = true
					return true
				}
				return false
			})

		if hasCompensateError {
			stateMachineInstance.SetCompensationStatus(statelang.UN)
		} else {
			stateMachineInstance.SetCompensationStatus(statelang.SU)
		}
	}
	return nil
}

func decideMachineForwardExecutionStatus(ctx context.Context, stateMachineInstance statelang.StateMachineInstance, exp error, specialPolicy bool) (bool, error) {
	result := false

	if stateMachineInstance.Status() == "" || statelang.RU == stateMachineInstance.Status() {
		result = true
		stateList := stateMachineInstance.StateList()
		setMachineStatusBasedOnStateListAndException(stateMachineInstance, stateList, exp)

		if specialPolicy && statelang.SU == stateMachineInstance.Status() {
			for _, stateInstance := range stateMachineInstance.StateList() {
				if !stateInstance.IsIgnoreStatus() && (stateInstance.IsForUpdate() || stateInstance.IsForCompensation()) {
					stateMachineInstance.SetStatus(statelang.UN)
					break
				}
			}
			if statelang.SU == stateMachineInstance.Status() {
				stateMachineInstance.SetStatus(statelang.FA)
			}
		}
	}
	return result, nil
}

func setMachineStatusBasedOnStateListAndException(stateMachineInstance statelang.StateMachineInstance,
	stateList []statelang.StateInstance, exp error) {
	hasSetStatus := false
	hasSuccessUpdateService := false
	if stateList != nil && len(stateList) > 0 {
		hasUnsuccessService := false

		for i := len(stateList) - 1; i >= 0; i-- {
			stateInstance := stateList[i]

			if stateInstance.IsIgnoreStatus() || stateInstance.IsForCompensation() {
				continue
			}
			if statelang.UN == stateInstance.Status() {
				stateMachineInstance.SetStatus(statelang.UN)
				hasSetStatus = true
			} else if statelang.SU == stateInstance.Status() {
				if constant.StateTypeServiceTask == stateInstance.Type() {
					if stateInstance.IsForUpdate() && !stateInstance.IsForCompensation() {
						hasSuccessUpdateService = true
					}
				}
			} else if statelang.SK == stateInstance.Status() {
				// ignore
			} else {
				hasUnsuccessService = true
			}
		}

		if !hasSetStatus && hasUnsuccessService {
			if hasSuccessUpdateService {
				stateMachineInstance.SetStatus(statelang.UN)
			} else {
				stateMachineInstance.SetStatus(statelang.FA)
			}
			hasSetStatus = true
		}
	}

	if !hasSetStatus {
		setMachineStatusBasedOnException(stateMachineInstance, exp, hasSuccessUpdateService)
	}
}

func setMachineStatusBasedOnException(stateMachineInstance statelang.StateMachineInstance, exp error, hasSuccessUpdateService bool) {
	//TODO implement me
	panic("implement me")
}

func (d DefaultStatusDecisionStrategy) DecideOnTaskStateFail(ctx context.Context, processContext ProcessContext,
	stateMachineInstance statelang.StateMachineInstance, exp error) error {
	//TODO implement me
	panic("implement me")
}

func (d DefaultStatusDecisionStrategy) DecideMachineForwardExecutionStatus(ctx context.Context,
	stateMachineInstance statelang.StateMachineInstance, exp error, specialPolicy bool) error {
	//TODO implement me
	panic("implement me")
}
