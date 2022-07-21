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

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/tm"
)

const (
	TwoPhaseActionTag            = "seataTwoPhaseAction"
	TwoPhaseActionNameTag        = "seataTwoPhaseServiceName"
	TwoPhaseActionPrepareTagVal  = "prepare"
	TwoPhaseActionCommitTagVal   = "commit"
	TwoPhaseActionRollbackTagVal = "rollback"
)

type TwoPhaseInterface interface {
	Prepare(ctx context.Context, params ...interface{}) (interface{}, error)
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

func (t *TwoPhaseAction) TwoPhaseService() interface{} {
	return t.twoPhaseService
}

func (t *TwoPhaseAction) SetTwoPhaseService(twoPhaseService interface{}) {
	t.twoPhaseService = twoPhaseService
}

func (t *TwoPhaseAction) ActionName() string {
	return t.actionName
}

func (t *TwoPhaseAction) SetActionName(actionName string) {
	t.actionName = actionName
}

func (t *TwoPhaseAction) PrepareMethodName() string {
	return t.prepareMethodName
}

func (t *TwoPhaseAction) SetPrepareMethodName(prepareMethodName string) {
	t.prepareMethodName = prepareMethodName
}

func (t *TwoPhaseAction) PrepareMethod() *reflect.Value {
	return t.prepareMethod
}

func (t *TwoPhaseAction) SetPrepareMethod(prepareMethod *reflect.Value) {
	t.prepareMethod = prepareMethod
}

func (t *TwoPhaseAction) CommitMethodName() string {
	return t.commitMethodName
}

func (t *TwoPhaseAction) SetCommitMethodName(commitMethodName string) {
	t.commitMethodName = commitMethodName
}

func (t *TwoPhaseAction) CommitMethod() *reflect.Value {
	return t.commitMethod
}

func (t *TwoPhaseAction) SetCommitMethod(commitMethod *reflect.Value) {
	t.commitMethod = commitMethod
}

func (t *TwoPhaseAction) RollbackMethodName() string {
	return t.rollbackMethodName
}

func (t *TwoPhaseAction) SetRollbackMethodName(rollbackMethodName string) {
	t.rollbackMethodName = rollbackMethodName
}

func (t *TwoPhaseAction) RollbackMethod() *reflect.Value {
	return t.rollbackMethod
}

func (t *TwoPhaseAction) SetRollbackMethod(rollbackMethod *reflect.Value) {
	t.rollbackMethod = rollbackMethod
}

func (t *TwoPhaseAction) GetTwoPhaseService() interface{} {
	return t.twoPhaseService
}

func (t *TwoPhaseAction) Prepare(ctx context.Context, params ...interface{}) (bool, error) {
	values := make([]reflect.Value, 0, len(params))
	values = append(values, reflect.ValueOf(ctx))
	for _, param := range params {
		values = append(values, reflect.ValueOf(param))
	}
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
	return m != nil && err != nil
}

func ParseTwoPhaseAction(v interface{}) (*TwoPhaseAction, error) {
	if m, ok := v.(TwoPhaseInterface); ok {
		return parseTwoPhaseActionByTwoPhaseInterface(m), nil
	}
	for _, parse := range remotingParseTable {
		if res, err := parse.ParseTwoPhaseActionByInterface(&v); err != nil {
			return nil, err
		} else if res != nil {
			return res, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found remoting parser for %v", v))
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

var (
	remotingParseTable = make([]RemotingParse, 0)
)

func RegisterRemotingParse(parse RemotingParse) {
	remotingParseTable = append(remotingParseTable, parse)
}
