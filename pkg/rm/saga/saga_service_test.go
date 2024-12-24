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

package saga

import (
	"context"
	"fmt"
	"github.com/agiledragon/gomonkey"
	gostnet "github.com/dubbogo/gost/net"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
	testdata2 "github.com/seata/seata-go/testdata"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

var (
	testSagaServiceProxy *SagaServiceProxy
	testBranchID         = int64(121324345353)
	names                []interface{}
	values               = make([]reflect.Value, 0, 2)
)

type UserProvider struct {
	Action       func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"action"`
	Compensation func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"compensation"`
}

func InitMock() {
	log.Init()
	var (
		registerResource = func(_ *SagaServiceProxy) error {
			return nil
		}
		branchRegister = func(_ *rm.RMRemoting, param rm.BranchRegisterParam) (int64, error) {
			return testBranchID, nil
		}
	)
	log.Infof("run init mock")
	gomonkey.ApplyMethod(reflect.TypeOf(testSagaServiceProxy), "RegisterResource", registerResource)
	gomonkey.ApplyMethod(reflect.TypeOf(rm.GetRMRemotingInstance()), "BranchRegister", branchRegister)
	testSagaServiceProxy, _ = NewSagaServiceProxy(GetTestTwoPhaseService())
}

func TestMain(m *testing.M) {
	InitMock()
	code := m.Run()
	os.Exit(code)
}

func TestInitActionContext(t *testing.T) {
	param := struct {
		name  string `sagaParam:"name"`
		Age   int64  `sagaParam:""`
		Addr  string `sagaParam:"addr"`
		Job   string `sagaParam:"-"`
		Class string
		Other []int8 `sagaParam:"Other"`
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
	result := testSagaServiceProxy.initActionContext(param)
	localIp, _ := gostnet.GetLocalIP()
	assert.Equal(t, map[string]interface{}{
		"addr":                      "Earth",
		"Other":                     []int8{1, 2, 3},
		constant.ActionStartTime:    now.UnixNano() / 1e6,
		constant.ActionMethod:       "Action",
		constant.CompensationMethod: "Compensation",
		constant.ActionName:         testdata2.ActionName,
		constant.HostName:           localIp,
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
		result := testSagaServiceProxy.getOrCreateBusinessActionContext(tt.param)
		assert.Equal(t, tt.want, *result)
	}
}

func TestRegisteBranch(t *testing.T) {
	ctx := testdata2.GetTestContext()
	err := testSagaServiceProxy.registerBranch(ctx, nil)
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

	twoPhaseAction1, _ := rm.ParseSagaTwoPhaseAction(userProvider)
	twoPhaseAction2, _ := rm.ParseSagaTwoPhaseAction(userProvider)

	tests := []struct {
		name    string
		args    args
		want    *SagaServiceProxy
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"test1", args1, &SagaServiceProxy{
				SagaResource: &SagaResource{
					ResourceGroupId: `default:"DEFAULT"`,
					AppName:         "seata-go-mock-app-name",
					SagaAction:      twoPhaseAction1,
				},
			}, assert.NoError,
		},
		{
			"test2", args2, &SagaServiceProxy{
				SagaResource: &SagaResource{
					ResourceGroupId: `default:"DEFAULT"`,
					AppName:         "seata-go-mock-app-name",
					SagaAction:      twoPhaseAction2,
				},
			}, assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSagaServiceProxy(tt.args.service)
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
		SagaResource         *SagaResource
	}

	userProvider := &UserProvider{}
	twoPhaseAction1, _ := rm.ParseSagaTwoPhaseAction(userProvider)

	tests := struct {
		name   string
		fields fields
		want   tm.GtxConfig
	}{
		"test1",
		fields{
			referenceName:        "test1",
			registerResourceOnce: sync.Once{},
			SagaResource: &SagaResource{
				ResourceGroupId: "default1",
				AppName:         "app1",
				SagaAction:      twoPhaseAction1,
			},
		},
		tm.GtxConfig{Name: "TwoPhaseDemoService", Timeout: time.Second * 10, Propagation: 0, LockRetryInternal: 0, LockRetryTimes: 0},
	}

	t1.Run(tests.name, func(t1 *testing.T) {
		t := &SagaServiceProxy{
			referenceName:        tests.fields.referenceName,
			registerResourceOnce: sync.Once{},
			SagaResource:         tests.fields.SagaResource,
		}
		assert.Equalf(t1, tests.want, t.GetTransactionInfo(), "GetTransactionInfo()")
	})
}

func GetTestTwoPhaseService() rm.SagaActionInterface {
	return &testdata2.TestSagaTwoPhaseService{}
}
