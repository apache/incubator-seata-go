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
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"github.com/seata/seata-go/pkg/util/log"
)

func GetLoopConfig(ctx context.Context, processContext process_ctrl.ProcessContext, currentState statelang.State) state.Loop {
	if matchLoop(currentState) {
		taskState := currentState.(state.AbstractTaskState)
		stateMachineInstance := processContext.GetVariable(constant.VarNameStateMachineInst).(statelang.StateMachineInstance)
		stateMachineConfig := processContext.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig)

		if taskState.Loop() != nil {
			loop := taskState.Loop()
			collectionName := loop.Collection()
			if collectionName != "" {
				expression := CreateValueExpression(stateMachineConfig.ExpressionResolver(), collectionName)
				collection := GetValue(expression, stateMachineInstance.Context(), nil)
				collectionList := collection.([]any)
				if len(collectionList) > 0 {
					current := GetCurrentLoopContextHolder(ctx, processContext, true)
					current.SetCollection(collection)
					return loop
				}
			}
			log.Warn("State [{}] loop collection param [{}] invalid", currentState.Name(), collectionName)
		}

	}
	return nil
}

func matchLoop(currentState statelang.State) bool {
	return currentState != nil && (constant.StateTypeServiceTask == currentState.Type() ||
		constant.StateTypeScriptTask == currentState.Type() || constant.StateTypeSubStateMachine == currentState.Type())
}
