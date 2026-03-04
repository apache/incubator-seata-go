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
	"fmt"
	"strings"
	"sync"

	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine/expr"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang/state"
)

func CreateInputParams(processContext process_ctrl.ProcessContext, expressionResolver expr.ExpressionResolver,
	stateInstance *statelang.StateInstanceImpl, serviceTaskState *state.AbstractTaskState, variablesFrom any) []any {
	inputAssignments := serviceTaskState.Input()
	if len(inputAssignments) == 0 {
		return inputAssignments
	}

	inputExpressions := serviceTaskState.InputExpressions()
	if len(inputExpressions) == 0 {
		lock := processContext.GetVariable(constant.VarNameProcessContextMutexLock).(*sync.Mutex)
		lock.Lock()
		defer lock.Unlock()
		inputExpressions = serviceTaskState.InputExpressions()
		if len(inputExpressions) == 0 {
			inputExpressions = make([]any, 0, len(inputAssignments))

			for _, assignment := range inputAssignments {
				inputExpressions = append(inputExpressions, CreateValueExpression(expressionResolver, assignment))
			}
		}
		serviceTaskState.SetInputExpressions(inputExpressions)
	}
	inputValues := make([]any, 0, len(inputExpressions))
	for _, valueExpression := range inputExpressions {
		value := GetValue(valueExpression, variablesFrom, stateInstance)
		inputValues = append(inputValues, value)
	}

	return inputValues
}

func CreateOutputParams(config engine.StateMachineConfig, expressionResolver expr.ExpressionResolver,
	serviceTaskState *state.AbstractTaskState, variablesFrom any) (map[string]any, error) {
	outputAssignments := serviceTaskState.Output()
	if len(outputAssignments) == 0 {
		return make(map[string]any, 0), nil
	}

	outputExpressions := serviceTaskState.OutputExpressions()
	if outputExpressions == nil {
		config.ComponentLock().Lock()
		defer config.ComponentLock().Unlock()
		outputExpressions = serviceTaskState.OutputExpressions()
		if outputExpressions == nil {
			outputExpressions = make(map[string]any, len(outputAssignments))
			for key, value := range outputAssignments {
				outputExpressions[key] = CreateValueExpression(expressionResolver, value)
			}
		}
		serviceTaskState.SetOutputExpressions(outputExpressions)
	}
	outputValues := make(map[string]any, len(outputExpressions))
	for paramName := range outputExpressions {
		outputValues[paramName] = GetValue(outputExpressions[paramName], variablesFrom, nil)
	}
	return outputValues, nil
}

func CreateValueExpression(expressionResolver expr.ExpressionResolver, paramAssignment any) any {
	switch v := paramAssignment.(type) {
	case expr.Expression:
		return v
	case map[string]any:
		paramMap := make(map[string]any, len(v))
		for key, value := range v {
			paramMap[key] = CreateValueExpression(expressionResolver, value)
		}
		return paramMap
	case []any:
		paramList := make([]any, 0, len(v))
		for _, value := range v {
			paramList = append(paramList, CreateValueExpression(expressionResolver, value))
		}
		return paramList
	case string:
		if !strings.HasPrefix(v, "$") {
			return v
		}
		return expressionResolver.Expression(v)
	default:
		return paramAssignment
	}
}

func GetValue(valueExpression any, variablesFrom any, stateInstance statelang.StateInstance) any {
	switch v := valueExpression.(type) {
	case expr.Expression:
		value := v.Value(variablesFrom)
		if _, ok := v.(expr.SequenceExpression); value != nil && stateInstance != nil && stateInstance.BusinessKey() == "" && ok {
			stateInstance.SetBusinessKey(fmt.Sprintf("%v", value))
		}
		return value
	case map[string]any:
		mapValue := make(map[string]any, len(v))
		for key, value := range v {
			resolved := GetValue(value, variablesFrom, stateInstance)
			if resolved != nil {
				mapValue[key] = resolved
			}
		}
		return mapValue
	case []any:
		listValue := make([]any, 0, len(v))
		for i := range v {
			listValue = append(listValue, GetValue(v[i], variablesFrom, stateInstance))
		}
		return listValue
	default:
		return valueExpression
	}
}
