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

package tcc

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"

	gostnet "github.com/dubbogo/gost/net"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/constant"

	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"

	//"github.com/apache/seata-go/sample/tcc/dubbo/client/service"
	testdata2 "seata.apache.org/seata-go/testdata"
)

var (
	testTccServiceProxy *TCCServiceProxy
	testBranchID        = int64(121324345353)
	names               []interface{}
	values              = make([]reflect.Value, 0, 2)
)

type UserProvider struct {
	Prepare       func(ctx context.Context, params ...interface{}) (bool, error)                           `seataTwoPhaseAction:"prepare" seataTwoPhaseServiceName:"TwoPhaseDemoService"`
	Commit        func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
	Rollback      func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
	GetActionName func() string
}

func InitMock() {
	log.Init()
	var (
		registerResource = func(_ *TCCServiceProxy) error {
			return nil
		}
		prepare = func(_ *TCCServiceProxy, ctx context.Context, params interface{}) (interface{}, error) {
			return nil, nil
		}
		branchRegister = func(_ *rm.RMRemoting, param rm.BranchRegisterParam) (int64, error) {
			return testBranchID, nil
		}
	)
	log.Infof("run init mock")
	gomonkey.ApplyMethod(reflect.TypeOf(testTccServiceProxy), "RegisterResource", registerResource)
	gomonkey.ApplyMethod(reflect.TypeOf(testTccServiceProxy), "Prepare", prepare)
	gomonkey.ApplyMethod(reflect.TypeOf(rm.GetRMRemotingInstance()), "BranchRegister", branchRegister)
	testTccServiceProxy, _ = NewTCCServiceProxy(GetTestTwoPhaseService())
}

func TestMain(m *testing.M) {
	InitMock()
	code := m.Run()
	os.Exit(code)
}

func TestInitActionContext(t *testing.T) {
	param := struct {
		name  string `tccParam:"name"`
		Age   int64  `tccParam:""`
		Addr  string `tccParam:"addr"`
		Job   string `tccParam:"-"`
		Class string
		Other []int8 `tccParam:"Other"`
	}{
		name:  "Jack",
		Age:   20,
		Addr:  "Earth",
		Job:   "Dor",
		Class: "1-2",
		Other: []int8{1, 2, 3},
	}

	now := time.Now()
	p := gomonkey.ApplyFunc(time.Now, func() time.Time {
		return now
	})
	defer p.Reset()
	result := testTccServiceProxy.initActionContext(param)
	localIp, _ := gostnet.GetLocalIP()
	assert.Equal(t, map[string]interface{}{
		"addr":                   "Earth",
		"Other":                  []int8{1, 2, 3},
		constant.ActionStartTime: now.UnixNano() / 1e6,
		constant.PrepareMethod:   "Prepare",
		constant.CommitMethod:    "Commit",
		constant.RollbackMethod:  "Rollback",
		constant.ActionName:      testdata2.ActionName,
		constant.HostName:        localIp,
	}, result)
}

func TestGetActionContextParameters(t *testing.T) {
	param := struct {
		name  string `tccParam:"name"`
		Age   int64  `tccParam:""`
		Addr  string `tccParam:"addr"`
		Job   string `tccParam:"-"`
		Class string
		Other []int8 `tccParam:"Other"`
	}{
		name:  "Jack",
		Age:   20,
		Addr:  "Earth",
		Job:   "Dor",
		Class: "1-2",
		Other: []int8{1, 2, 3},
	}

	result := testTccServiceProxy.getActionContextParameters(param)
	assert.Equal(t, map[string]interface{}{
		"addr":  "Earth",
		"Other": []int8{1, 2, 3},
	}, result)
}

func TestGetOrCreateBusinessActionContext(t *testing.T) {
	tests := []struct {
		param interface{}
		want  tm.BusinessActionContext
	}{
		{
			param: nil,
			want:  tm.BusinessActionContext{},
		},
		{
			param: tm.BusinessActionContext{
				ActionContext: map[string]interface{}{
					"name": "Jack",
					"age":  12,
				},
			},
			want: tm.BusinessActionContext{
				ActionContext: map[string]interface{}{
					"name": "Jack",
					"age":  12,
				},
			},
		},
		{
			param: &tm.BusinessActionContext{
				ActionContext: map[string]interface{}{
					"name": "Jack",
					"age":  13,
				},
			},
			want: tm.BusinessActionContext{
				ActionContext: map[string]interface{}{
					"name": "Jack",
					"age":  13,
				},
			},
		},
		{
			param: struct {
				Context *tm.BusinessActionContext
			}{
				Context: &tm.BusinessActionContext{
					ActionContext: map[string]interface{}{
						"name": "Jack",
						"age":  14,
					},
				},
			},
			want: tm.BusinessActionContext{
				ActionContext: map[string]interface{}{
					"name": "Jack",
					"age":  14,
				},
			},
		},
		{
			param: struct {
				Context tm.BusinessActionContext
			}{
				Context: tm.BusinessActionContext{
					ActionContext: map[string]interface{}{
						"name": "Jack",
						"age":  12,
					},
				},
			},
			want: tm.BusinessActionContext{
				ActionContext: map[string]interface{}{
					"name": "Jack",
					"age":  12,
				},
			},
		},
		{
			param: struct {
				context tm.BusinessActionContext
			}{
				context: tm.BusinessActionContext{
					ActionContext: map[string]interface{}{
						"name": "Jack",
						"age":  12,
					},
				},
			},
			want: tm.BusinessActionContext{},
		},
	}

	for _, tt := range tests {
		result := testTccServiceProxy.getOrCreateBusinessActionContext(tt.param)
		assert.Equal(t, tt.want, *result)
	}
}

func TestRegisteBranch(t *testing.T) {
	ctx := testdata2.GetTestContext()
	err := testTccServiceProxy.registeBranch(ctx, nil)
	assert.Nil(t, err)
	bizContext := tm.GetBusinessActionContext(ctx)
	assert.Equal(t, testBranchID, bizContext.BranchId)
}

func TestNewTCCServiceProxy(t *testing.T) {
	type args struct {
		service interface{}
	}

	userProvider := &UserProvider{}
	args1 := args{service: userProvider}
	args2 := args{service: userProvider}

	twoPhaseAction1, _ := rm.ParseTwoPhaseAction(userProvider)
	twoPhaseAction2, _ := rm.ParseTwoPhaseAction(userProvider)

	tests := []struct {
		name    string
		args    args
		want    *TCCServiceProxy
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"test1", args1, &TCCServiceProxy{
				TCCResource: &TCCResource{
					ResourceGroupId: `default:"DEFAULT"`,
					AppName:         "seata-go-mock-app-name",
					TwoPhaseAction:  twoPhaseAction1,
				},
			}, assert.NoError,
		},
		{
			"test2", args2, &TCCServiceProxy{
				TCCResource: &TCCResource{
					ResourceGroupId: `default:"DEFAULT"`,
					AppName:         "seata-go-mock-app-name",
					TwoPhaseAction:  twoPhaseAction2,
				},
			}, assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTCCServiceProxy(tt.args.service)
			if !tt.wantErr(t, err, fmt.Sprintf("NewTCCServiceProxy(%v)", tt.args.service)) {
				return
			}
			assert.Equalf(t, tt.want, got, "NewTCCServiceProxy(%v)", tt.args.service)
		})
	}
}

func TestTCCGetTransactionInfo(t1 *testing.T) {
	type fields struct {
		referenceName        string
		registerResourceOnce sync.Once
		TCCResource          *TCCResource
	}

	userProvider := &UserProvider{}
	twoPhaseAction1, _ := rm.ParseTwoPhaseAction(userProvider)

	tests := struct {
		name   string
		fields fields
		want   tm.GtxConfig
	}{
		"test1",
		fields{
			referenceName:        "test1",
			registerResourceOnce: sync.Once{},
			TCCResource: &TCCResource{
				ResourceGroupId: "default1",
				AppName:         "app1",
				TwoPhaseAction:  twoPhaseAction1,
			},
		},
		tm.GtxConfig{Name: "TwoPhaseDemoService", Timeout: time.Second * 10, Propagation: 0, LockRetryInternal: 0, LockRetryTimes: 0},
	}

	t1.Run(tests.name, func(t1 *testing.T) {
		t := &TCCServiceProxy{
			referenceName:        tests.fields.referenceName,
			registerResourceOnce: sync.Once{},
			TCCResource:          tests.fields.TCCResource,
		}
		assert.Equalf(t1, tests.want, t.GetTransactionInfo(), "GetTransactionInfo()")
	})
}

func GetTestTwoPhaseService() rm.TwoPhaseInterface {
	return &testdata2.TestTwoPhaseService{}
}
