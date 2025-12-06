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
	"reflect"

	"seata.apache.org/seata-go/pkg/tm"
)

const (
	TwoPhaseActionTag            = "seataTwoPhaseAction"
	TwoPhaseActionNameTag        = "seataTwoPhaseServiceName"
	TwoPhaseActionPrepareTagVal  = "prepare"
	TwoPhaseActionCommitTagVal   = "commit"
	TwoPhaseActionRollbackTagVal = "rollback"
)

var (
	typError                    = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()).Type()
	typContext                  = reflect.Zero(reflect.TypeOf((*context.Context)(nil)).Elem()).Type()
	typBool                     = reflect.Zero(reflect.TypeOf((*bool)(nil)).Elem()).Type()
	TypBusinessContextInterface = reflect.Zero(reflect.TypeOf((*tm.BusinessActionContext)(nil))).Type()
)

type TwoPhaseInterface interface {
	Prepare(ctx context.Context, params interface{}) (bool, error)
	Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error)
	Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error)
	GetActionName() string
}

type TwoPhaseAction struct {
	twoPhaseService    interface{}
	actionName         string
	prepareMethodName  string
	prepareMethod      *reflect.Value
	commitMethodName   string
	commitMethod       *reflect.Value
	rollbackMethodName string
	rollbackMethod     *reflect.Value
}

func (t *TwoPhaseAction) GetTwoPhaseService() interface{} {
	return t.twoPhaseService
}

func (t *TwoPhaseAction) GetPrepareMethodName() string {
	return t.prepareMethodName
}

func (t *TwoPhaseAction) GetCommitMethodName() string {
	return t.commitMethodName
}

func (t *TwoPhaseAction) GetRollbackMethodName() string {
	return t.rollbackMethodName
}

func (t *TwoPhaseAction) Prepare(ctx context.Context, params interface{}) (bool, error) {
	values := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(params)}
	res := t.prepareMethod.Call(values)
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

func (t *TwoPhaseAction) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	res := t.commitMethod.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(businessActionContext)})
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

func (t *TwoPhaseAction) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	res := t.rollbackMethod.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(businessActionContext)})
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

func (t *TwoPhaseAction) GetActionName() string {
	return t.actionName
}

func IsTwoPhaseAction(v interface{}) bool {
	m, err := ParseTwoPhaseAction(v)
	return m != nil && err == nil
}

func ParseTwoPhaseAction(v interface{}) (*TwoPhaseAction, error) {
	if m, ok := v.(TwoPhaseInterface); ok {
		return parseTwoPhaseActionByTwoPhaseInterface(m), nil
	}
	return ParseTwoPhaseActionByInterface(v)
}

func parseTwoPhaseActionByTwoPhaseInterface(v TwoPhaseInterface) *TwoPhaseAction {
	value := reflect.ValueOf(v)
	mp := value.MethodByName("Prepare")
	mc := value.MethodByName("Commit")
	mr := value.MethodByName("Rollback")
	return &TwoPhaseAction{
		twoPhaseService:    v,
		actionName:         v.GetActionName(),
		prepareMethodName:  "Prepare",
		prepareMethod:      &mp,
		commitMethodName:   "Commit",
		commitMethod:       &mc,
		rollbackMethodName: "Rollback",
		rollbackMethod:     &mr,
	}
}

func ParseTwoPhaseActionByInterface(v interface{}) (*TwoPhaseAction, error) {
	valueOfElem := reflect.ValueOf(v).Elem()
	typeOf := valueOfElem.Type()
	k := typeOf.Kind()
	if k != reflect.Struct {
		return nil, fmt.Errorf("param should be a struct, instead of a pointer")
	}
	numField := typeOf.NumField()

	var (
		hasPrepareMethodName bool
		hasCommitMethodName  bool
		hasRollbackMethod    bool
		twoPhaseName         string
		result               = TwoPhaseAction{
			twoPhaseService: v,
		}
	)
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		f := valueOfElem.Field(i)
		if ms, m, ok := getPrepareAction(t, f); ok {
			hasPrepareMethodName = true
			result.prepareMethod = m
			result.prepareMethodName = ms
		} else if ms, m, ok = getCommitMethod(t, f); ok {
			hasCommitMethodName = true
			result.commitMethod = m
			result.commitMethodName = ms
		} else if ms, m, ok = getRollbackMethod(t, f); ok {
			hasRollbackMethod = true
			result.rollbackMethod = m
			result.rollbackMethodName = ms
		}
	}
	if !hasPrepareMethodName {
		return nil, fmt.Errorf("missing prepare method")
	}
	if !hasCommitMethodName {
		return nil, fmt.Errorf("missing commit method")
	}
	if !hasRollbackMethod {
		return nil, fmt.Errorf("missing rollback method")
	}
	twoPhaseName = getActionName(v)
	if twoPhaseName == "" {
		return nil, fmt.Errorf("missing two phase name")
	}
	result.actionName = twoPhaseName
	return &result, nil
}

func getPrepareAction(t reflect.StructField, f reflect.Value) (string, *reflect.Value, bool) {
	if t.Tag.Get(TwoPhaseActionTag) != TwoPhaseActionPrepareTagVal {
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

func getCommitMethod(t reflect.StructField, f reflect.Value) (string, *reflect.Value, bool) {
	if t.Tag.Get(TwoPhaseActionTag) != TwoPhaseActionCommitTagVal {
		return "", nil, false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", nil, false
	}
	// commit method has 2 return error value
	if outNum := t.Type.NumOut(); outNum != 2 {
		return "", nil, false
	}
	if returnType := t.Type.Out(0); returnType != typBool {
		return "", nil, false
	}
	if returnType := t.Type.Out(1); returnType != typError {
		return "", nil, false
	}
	// commit method has at least 1 params, context.Context, and other params
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

func getRollbackMethod(t reflect.StructField, f reflect.Value) (string, *reflect.Value, bool) {
	if t.Tag.Get(TwoPhaseActionTag) != TwoPhaseActionRollbackTagVal {
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

func getActionName(v interface{}) string {
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
