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
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey"
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/common/net"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/tm"
	testdata2 "github.com/seata/seata-go/testdata"
	"github.com/stretchr/testify/assert"
)

var (
	testTccServiceProxy, _ = NewTCCServiceProxy(testdata2.GetTestTwoPhaseService())
	testBranchID           = int64(121324345353)
)

func Init() {
	initRegisterResource()
}

func initRegisterResource() {
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
	gomonkey.ApplyMethod(reflect.TypeOf(testTccServiceProxy), "RegisterResource", registerResource)
	gomonkey.ApplyMethod(reflect.TypeOf(testTccServiceProxy), "Prepare", prepare)
	gomonkey.ApplyMethod(reflect.TypeOf(rm.GetRMRemotingInstance()), "BranchRegister", branchRegister)
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

	now := time.Now().UnixNano()
	gomonkey.ApplyMethod(reflect.TypeOf(time.Now()), "UnixNano", func(_ time.Time) int64 {
		return now
	})
	result := testTccServiceProxy.initActionContext(param)
	assert.Equal(t, map[string]interface{}{
		"addr":                 "Earth",
		"Other":                []int8{1, 2, 3},
		common.ActionStartTime: now / 1e6,
		common.PrepareMethod:   "Prepare",
		common.CommitMethod:    "Commit",
		common.RollbackMethod:  "Rollback",
		common.ActionName:      testdata2.ActionName,
		common.HostName:        net.GetLocalIp(),
	}, result)
}

func TestGetActionContextParameters(t *testing.T) {
	Init()
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
	Init()
	var tests = []struct {
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
		}, {
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
		}, {
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
	Init()
	ctx := testdata2.GetTestContext()
	err := testTccServiceProxy.registeBranch(ctx, nil)
	assert.Nil(t, err)
	bizContext := tm.GetBusinessActionContext(ctx)
	assert.Equal(t, testBranchID, bizContext.BranchId)
}
