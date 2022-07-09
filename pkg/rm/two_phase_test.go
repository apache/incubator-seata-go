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
	"testing"

	"github.com/seata/seata-go/pkg/tm"
	"github.com/stretchr/testify/assert"
)

func TestParseTwoPhaseActionGetMethodName(t *testing.T) {
	tests := []struct {
		service      interface{}
		wantService  *TwoPhaseAction
		wantHasError bool
		wantErrMsg   string
	}{{
		service: &struct {
			Prepare111 func(ctx context.Context, params string) (bool, error)                                   `seataTwoPhaseAction:"prepare"`
			Commit111  func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
			Rollback11 func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback" seataTwoPhaseServiceName:"seataTwoPhaseName"`
			GetName    func() string
		}{},
		wantService: &TwoPhaseAction{
			actionName:         "seataTwoPhaseName",
			prepareMethodName:  "Prepare111",
			commitMethodName:   "Commit111",
			rollbackMethodName: "Rollback11",
		},
		wantHasError: false,
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) (bool, error)                              `seataTwoPhaseAction:"prepare"`
			Commit   func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
			Rollback func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing two phase name",
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) (bool, error)                              `serviceName:"seataTwoPhaseName"`
			Commit   func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
			Rollback func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing prepare method",
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) (bool, error) `seataTwoPhaseAction:"commit" seataTwoPhaseName:"seataTwoPhaseName"`
			Commit   func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error)
			Rollback func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing prepare method",
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) (bool, error)                              `seataTwoPhaseAction:"prepare" seataTwoPhaseName:"seataTwoPhaseName"`
			Commit   func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
			Rollback func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error)
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing rollback method",
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) (bool, error)                              `seataTwoPhaseAction:"prepare"`
			Commit   func(ctx context.Context, businessActionContext tm.BusinessActionContext) (bool, error)  `seataTwoPhaseAction:"commit"`
			Rollback func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback" seataTwoPhaseServiceName:"seataTwoPhaseName"`
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing commit method",
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) (bool, error)                              `seataTwoPhaseAction:"prepare"`
			Commit   func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
			Rollback func(ctx context.Context, businessActionContext tm.BusinessActionContext) (bool, error)  `seataTwoPhaseAction:"rollback" seataTwoPhaseServiceName:"seataTwoPhaseName"`
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing rollback method",
	}}

	for _, tt := range tests {
		actual, err := ParseTwoPhaseAction(tt.service)
		if tt.wantHasError {
			assert.NotNil(t, err)
			assert.Equal(t, tt.wantErrMsg, err.Error())
		} else {
			assert.Nil(t, err)
			assert.Equal(t, actual.actionName, tt.wantService.actionName)
			assert.Equal(t, actual.prepareMethodName, tt.wantService.prepareMethodName)
			assert.Equal(t, actual.commitMethodName, tt.wantService.commitMethodName)
			assert.Equal(t, actual.rollbackMethodName, tt.wantService.rollbackMethodName)
		}
	}
}

type TwoPhaseDemoService1 struct {
	TwoPhasePrepare     func(ctx context.Context, params ...interface{}) (bool, error)                           `seataTwoPhaseAction:"prepare" seataTwoPhaseServiceName:"TwoPhaseDemoService"`
	TwoPhaseCommit      func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
	TwoPhaseRollback    func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
	TwoPhaseDemoService func() string
}

func NewTwoPhaseDemoService1() *TwoPhaseDemoService1 {
	return &TwoPhaseDemoService1{
		TwoPhasePrepare: func(ctx context.Context, params ...interface{}) (bool, error) {
			return false, fmt.Errorf("execute two phase prepare method, param %v", params)
		},
		TwoPhaseCommit: func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
			return false, fmt.Errorf("execute two phase commit method, xid %v", businessActionContext.Xid)
		},
		TwoPhaseRollback: func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
			return true, nil
		},
		TwoPhaseDemoService: func() string {
			return "TwoPhaseDemoService"
		},
	}
}

func TestParseTwoPhaseActionExecuteMethod1(t *testing.T) {
	twoPhaseService, err := ParseTwoPhaseAction(NewTwoPhaseDemoService1())
	ctx := context.Background()
	assert.Nil(t, err)
	assert.Equal(t, "TwoPhasePrepare", twoPhaseService.prepareMethodName)
	assert.Equal(t, "TwoPhaseCommit", twoPhaseService.commitMethodName)
	assert.Equal(t, "TwoPhaseRollback", twoPhaseService.rollbackMethodName)
	assert.Equal(t, "TwoPhaseDemoService", twoPhaseService.actionName)

	resp, err := twoPhaseService.Prepare(ctx, 11)
	assert.Equal(t, false, resp)
	assert.Equal(t, "execute two phase prepare method, param [11]", err.Error())

	resp, err = twoPhaseService.Commit(ctx, &tm.BusinessActionContext{Xid: "1234"})
	assert.Equal(t, false, resp)
	assert.Equal(t, "execute two phase commit method, xid 1234", err.Error())

	resp, err = twoPhaseService.Rollback(ctx, &tm.BusinessActionContext{Xid: "1234"})
	assert.Equal(t, true, resp)
	assert.Nil(t, err)

	assert.Equal(t, "TwoPhaseDemoService", twoPhaseService.GetActionName())
}

type TwoPhaseDemoService2 struct {
}

func (t *TwoPhaseDemoService2) Prepare(ctx context.Context, params ...interface{}) (bool, error) {
	return false, fmt.Errorf("execute two phase prepare method, param %v", params)
}

func (t *TwoPhaseDemoService2) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return true, fmt.Errorf("execute two phase commit method, xid %v", businessActionContext.Xid)
}

func (t *TwoPhaseDemoService2) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return false, fmt.Errorf("execute two phase rollback method, xid %v", businessActionContext.Xid)
}

func (t *TwoPhaseDemoService2) GetActionName() string {
	return "TwoPhaseDemoService2"
}

func TestParseTwoPhaseActionExecuteMethod2(t *testing.T) {
	twoPhaseService, err := ParseTwoPhaseAction(&TwoPhaseDemoService2{})
	ctx := context.Background()
	assert.Nil(t, err)
	resp, err := twoPhaseService.Prepare(ctx, 11)
	assert.Equal(t, false, resp)
	assert.Equal(t, "execute two phase prepare method, param [11]", err.Error())

	resp, err = twoPhaseService.Commit(ctx, &tm.BusinessActionContext{Xid: "1234"})
	assert.Equal(t, true, resp)
	assert.Equal(t, "execute two phase commit method, xid 1234", err.Error())

	resp, err = twoPhaseService.Rollback(ctx, &tm.BusinessActionContext{Xid: "1234"})
	assert.Equal(t, false, resp)
	assert.Equal(t, "execute two phase rollback method, xid 1234", err.Error())
}
