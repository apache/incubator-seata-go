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

package dubbo

import (
	"context"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/stretchr/testify/assert"
)

// TestGetDubboTransactionFilter unit test for GetDubboTransactionFilter
func TestGetDubboTransactionFilter(t *testing.T) {
	tests := []struct {
		name string
		want filter.Filter
	}{
		{
			name: "TestGetDubboTransactionFilter",
			want: GetDubboTransactionFilter(),
		},
		{
			name: "TestGetDubboTransactionFilter1",
			want: GetDubboTransactionFilter(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, GetDubboTransactionFilter(), tt.want)
		})
	}
}

// TestGetDubboTransactionFilter unit test for GetDubboTransactionFilter
func TestDubboTransactionFilterOnResponse(t *testing.T) {
	type args struct {
		ctx        context.Context
		result     protocol.Result
		invoker    protocol.Invoker
		invocation protocol.Invocation
	}
	tests := []struct {
		name string
		args args
		want protocol.Result
	}{
		{
			name: "Test_dubboTransactionFilter_OnResponse",
			args: args{
				ctx:        context.Background(),
				result:     nil,
				invoker:    nil,
				invocation: nil,
			},
			want: nil,
		},
		{
			name: "Test_dubboTransactionFilter_OnResponse1",
			args: args{
				ctx:        context.TODO(),
				result:     nil,
				invoker:    nil,
				invocation: nil,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			du := &dubboTransactionFilter{}
			got := du.OnResponse(tt.args.ctx, tt.args.result, tt.args.invoker, tt.args.invocation)
			assert.Equal(t, got, tt.want)
		})
	}
}
