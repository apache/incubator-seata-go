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
	"encoding/json"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestTCCServiceProxy_SetReferenceName(t1 *testing.T) {
	type fields struct {
		referenceName        string
		registerResourceOnce sync.Once
		TCCResource          *TCCResource
	}
	type args struct {
		referenceName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "TestNewTCCServiceProxy",
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
		})
	}
}

func TestTCCServiceProxy_registeBranch(t1 *testing.T) {
	applicationData := `{"actionContext":{"dzw":"lisi"}}`
	businessActionContext := GetTCCResourceManagerInstance().
		getBusinessActionContext("1111111111", 2645236141, "TestActionContext", []byte(applicationData))

	assert.NotEmpty(t1, businessActionContext)
	bytes, err := json.Marshal(businessActionContext.ActionContext)
	assert.Nil(t1, err)
	assert.Equal(t1, `{"dzw":"lisi"}`, string(bytes))
}
