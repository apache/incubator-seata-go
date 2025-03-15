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

package rm

import (
	"context"
	"fmt"
	"github.com/seata/seata-go/pkg/tm"
	"reflect"
)

// Define saga mode action, compensationAction branch transaction
// submission, compensation two actions

const (
	TwoPhaseActionActionTagVal       = "action"
	TwoPhaseActionCompensationTagVal = "compensation"
)

type SagaActionInterface interface {
	Action(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error)

	Compensation(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error)
}

type SagaAction struct {
	sagaTwoPhaseService interface{}
	action              string
	normalActionName    string
	normalAction        *reflect.Value
	compensationName    string
	compensation        *reflect.Value
}

func ParseSagaTwoPhaseAction(v interface{}) (*SagaAction, error) {
	if m, ok := v.(TwoPhaseInterface); ok {
		return parseSagaTwoPhaseActionByTwoPhaseInterface(m), nil
	}
	return ParseSagaTwoPhaseActionByInterface(v)
}

func parseSagaTwoPhaseActionByTwoPhaseInterface(v TwoPhaseInterface) *SagaAction {
	value := reflect.ValueOf(v)
	ma := value.MethodByName("action")
	mc := value.MethodByName("compensation")
	return &SagaAction{
		sagaTwoPhaseService: v,
		action:              v.GetActionName(),
		normalActionName:    "Prepare",
		normalAction:        &ma,
		compensationName:    "Commit",
		compensation:        &mc,
	}
}

func ParseSagaTwoPhaseActionByInterface(v interface{}) (*SagaAction, error) {
	valueOfElem := reflect.ValueOf(v).Elem()
	typeOf := valueOfElem.Type()
	k := typeOf.Kind()
	if k != reflect.Struct {
		return nil, fmt.Errorf("param should be a struct, instead of a pointer")
	}
	numField := typeOf.NumField()

	var (
		hasNormalMethodName   bool
		hasCompensationMethod bool
		twoPhaseName          string
		result                = SagaAction{
			sagaTwoPhaseService: v,
		}
	)
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		f := valueOfElem.Field(i)
		if ms, m, ok := getNormalAction(t, f); ok {
			hasNormalMethodName = true
			result.normalAction = m
			result.normalActionName = ms
		} else if ms, m, ok = getCompensationMethod(t, f); ok {
			hasCompensationMethod = true
			result.compensation = m
			result.compensationName = ms
		}
	}
	if !hasNormalMethodName {
		return nil, fmt.Errorf("missing normal method")
	}
	if !hasCompensationMethod {
		return nil, fmt.Errorf("missing compensation method")
	}
	twoPhaseName = getActionName(v)
	if twoPhaseName == "" {
		return nil, fmt.Errorf("missing two phase name")
	}
	result.action = twoPhaseName
	return &result, nil
}

func getNormalAction(t reflect.StructField, f reflect.Value) (string, *reflect.Value, bool) {
	if t.Tag.Get(TwoPhaseActionTag) != TwoPhaseActionActionTagVal {
		return "", nil, false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", nil, false
	}
	// prepare has 2 return error value
	if outNum := t.Type.NumOut(); outNum != 2 {
		return "", nil, false
	}
	if returnType := t.Type.Out(0); returnType != typBool {
		return "", nil, false
	}
	if returnType := t.Type.Out(1); returnType != typError {
		return "", nil, false
	}
	// prepared method has at least 1 params, context.Context, and other params
	if inNum := t.Type.NumIn(); inNum == 0 {
		return "", nil, false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", nil, false
	}
	return t.Name, &f, true
}

func getCompensationMethod(t reflect.StructField, f reflect.Value) (string, *reflect.Value, bool) {
	if t.Tag.Get(TwoPhaseActionTag) != TwoPhaseActionCompensationTagVal {
		return "", nil, false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", nil, false
	}
	// rollback method has 2 return value
	if outNum := t.Type.NumOut(); outNum != 2 {
		return "", nil, false
	}
	if returnType := t.Type.Out(0); returnType != typBool {
		return "", nil, false
	}
	if returnType := t.Type.Out(1); returnType != typError {
		return "", nil, false
	}
	// rollback method has at least 1 params, context.Context, and other params
	if inNum := t.Type.NumIn(); inNum != 2 {
		return "", nil, false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", nil, false
	}
	if inType := t.Type.In(1); inType != TypBusinessContextInterface {
		return "", nil, false
	}
	return t.Name, &f, true
}

func (sagaAction *SagaAction) GetNormalActionName() string {
	return sagaAction.normalActionName
}

func (sagaAction *SagaAction) GetCompensationName() string {
	return sagaAction.compensationName
}

func (sagaAction *SagaAction) Action(ctx context.Context, param interface{}) (bool, error) {
	values := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(param)}
	res := sagaAction.normalAction.Call(values)
	var (
		r0   = res[0].Interface()
		r1   = res[1].Interface()
		res0 bool
		res1 error
	)
	if r0 != nil {
		res0 = r0.(bool)
	}
	if r1 != nil {
		res1 = r1.(error)
	}
	return res0, res1
}

//透传上下文参数
func (sagaAction *SagaAction) Compensation(ctx context.Context, actionContext *tm.BusinessActionContext) (bool, error) {
	values := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(actionContext)}
	res := sagaAction.normalAction.Call(values)
	var (
		r0   = res[0].Interface()
		r1   = res[1].Interface()
		res0 bool
		res1 error
	)
	if r0 != nil {
		res0 = r0.(bool)
	}
	if r1 != nil {
		res1 = r1.(error)
	}
	return res0, res1
}

func (SagaAction *SagaAction) GetSagaActionName() string {
	return SagaAction.action
}

func (sagaAction *SagaAction) GetActionName(v interface{}) string {
	var (
		actionName  string
		valueOf     = reflect.ValueOf(v)
		valueOfElem = valueOf.Elem()
		typeOf      = valueOfElem.Type()
	)
	if typeOf.Kind() != reflect.Struct {
		return ""
	}
	numField := valueOfElem.NumField()
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		if actionName = t.Tag.Get(TwoPhaseActionNameTag); actionName != "" {
			break
		}
	}
	return actionName
}
