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
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"strings"
	"sync"
)

func CreateInputParams(processContext process_ctrl.ProcessContext, expressionResolver expr.ExpressionResolver,
	stateInstance *statelang.StateInstanceImpl, serviceTaskState *state.AbstractTaskState, variablesFrom any) []any {
	inputAssignments := serviceTaskState.Input()
	if inputAssignments == nil || len(inputAssignments) == 0 {
		return inputAssignments
	}

	inputExpressions := serviceTaskState.InputExpressions()
	if inputExpressions == nil || len(inputExpressions) == 0 {
		lock := processContext.GetVariable(constant.VarNameProcessContextMutexLock).(*sync.Mutex)
		lock.Lock()
		defer lock.Unlock()
		inputExpressions = serviceTaskState.InputExpressions()
		if inputExpressions == nil || len(inputExpressions) == 0 {
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
	if outputAssignments == nil || len(outputAssignments) == 0 {
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
	for paramName, _ := range outputExpressions {
		outputValues[paramName] = GetValue(outputExpressions[paramName], variablesFrom, nil)
	}
	return outputValues, nil
}

func CreateValueExpression(expressionResolver expr.ExpressionResolver, paramAssignment any) any {
	var valueExpression any

	switch paramAssignment.(type) {
	case expr.Expression:
		valueExpression = paramAssignment
	case map[string]any:
		paramMapAssignment := paramAssignment.(map[string]any)
		paramMap := make(map[string]any, len(paramMapAssignment))
		for key, value := range paramMapAssignment {
			paramMap[key] = CreateValueExpression(expressionResolver, value)
		}
		valueExpression = paramMap
	case []any:
		paramListAssignment := paramAssignment.([]any)
		paramList := make([]any, 0, len(paramListAssignment))
		for _, value := range paramListAssignment {
			paramList = append(paramList, CreateValueExpression(expressionResolver, value))
		}
		valueExpression = paramList
	case string:
		value := paramAssignment.(string)
		if !strings.HasPrefix(value, "$") {
			valueExpression = paramAssignment
		}
		valueExpression = expressionResolver.Expression(value)
	default:
		valueExpression = paramAssignment
	}
	return valueExpression
}

func GetValue(valueExpression any, variablesFrom any, stateInstance statelang.StateInstance) any {
	switch valueExpression.(type) {
	case expr.Expression:
		expression := valueExpression.(expr.Expression)
		value := expression.Value(variablesFrom)
		if _, ok := valueExpression.(expr.SequenceExpression); value != nil && stateInstance != nil && stateInstance.BusinessKey() == "" && ok {
			stateInstance.SetBusinessKey(fmt.Sprintf("%v", value))
		}
		return value
	case map[string]any:
		mapValueExpression := valueExpression.(map[string]any)
		mapValue := make(map[string]any, len(mapValueExpression))
		for key, value := range mapValueExpression {
			value = GetValue(value, variablesFrom, stateInstance)
			if value != nil {
				mapValue[key] = value
			}
		}
		return mapValue
	case []any:
		valueExpressionList := valueExpression.([]any)
		listValue := make([]any, 0, len(valueExpression.([]any)))
		for i, _ := range valueExpressionList {
			listValue = append(listValue, GetValue(valueExpressionList[i], variablesFrom, stateInstance))
		}
		return listValue
	default:
		return valueExpression
	}
}
