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
	"reflect"
	"strings"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

// TestGetDubboTransactionFilter unit test for GetDubboTransactionFilter
func TestGetDubboTransactionFilter(t *testing.T) {
	tests := []struct {
		name string
		want filter.Filter
	}{
		{
			name: "first_call",
			want: GetDubboTransactionFilter(),
		},
		{
			name: "second_call_returns_same_instance",
			want: GetDubboTransactionFilter(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, GetDubboTransactionFilter(), tt.want)
		})
	}
}

// TestDubboTransactionFilterOnResponse unit test for OnResponse
func TestDubboTransactionFilterOnResponse(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		result     protocol.Result
		invoker    protocol.Invoker
		invocation protocol.Invocation
		want       protocol.Result
	}{
		{
			name:       "with_background_context",
			ctx:        context.Background(),
			result:     nil,
			invoker:    nil,
			invocation: nil,
			want:       nil,
		},
		{
			name:       "with_todo_context",
			ctx:        context.TODO(),
			result:     nil,
			invoker:    nil,
			invocation: nil,
			want:       nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			du := &dubboTransactionFilter{}
			got := du.OnResponse(tt.ctx, tt.result, tt.invoker, tt.invocation)
			assert.Equal(t, tt.want, got)
		})
	}
}

type mockInvocation struct {
	attachments map[string]interface{}
}

func newMockInvocation() *mockInvocation {
	return &mockInvocation{
		attachments: make(map[string]interface{}),
	}
}

func (m *mockInvocation) MethodName() string                  { return "mockMethod" }
func (m *mockInvocation) ParameterTypeNames() []string        { return nil }
func (m *mockInvocation) ParameterTypes() []reflect.Type      { return nil }
func (m *mockInvocation) ParameterValues() []reflect.Value    { return nil }
func (m *mockInvocation) Arguments() []interface{}            { return nil }
func (m *mockInvocation) Reply() interface{}                  { return nil }
func (m *mockInvocation) Attachments() map[string]interface{} { return m.attachments }
func (m *mockInvocation) Attributes() map[string]interface{}  { return nil }
func (m *mockInvocation) AttributeByKey(string, interface{}) interface{} {
	return nil
}
func (m *mockInvocation) SetAttachment(key string, value interface{}) {
	m.attachments[key] = value
}
func (m *mockInvocation) GetAttachment(key string) (string, bool) {
	if val, ok := m.attachments[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}
func (m *mockInvocation) GetAttachmentWithDefaultValue(key, defaultValue string) string {
	if val, ok := m.GetAttachment(key); ok {
		return val
	}
	return defaultValue
}
func (m *mockInvocation) ServiceKey() string          { return "" }
func (m *mockInvocation) Invoker() protocol.Invoker   { return nil }
func (m *mockInvocation) SetInvoker(protocol.Invoker) {}
func (m *mockInvocation) ActualMethodName() string    { return "mockMethod" }
func (m *mockInvocation) IsGenericInvocation() bool   { return false }
func (m *mockInvocation) GetAttachmentInterface(key string) interface{} {
	return m.attachments[key]
}
func (m *mockInvocation) GetAttachmentAsContext() context.Context { return context.Background() }
func (m *mockInvocation) SetAttribute(string, interface{})        {}
func (m *mockInvocation) GetAttribute(string) (interface{}, bool) { return nil, false }
func (m *mockInvocation) GetAttributeWithDefaultValue(string, interface{}) interface{} {
	return nil
}

type mockInvoker struct {
	invoked    bool
	invokeFunc func(context.Context, protocol.Invocation) protocol.Result
}

func (m *mockInvoker) GetURL() *common.URL { return nil }
func (m *mockInvoker) IsAvailable() bool   { return true }
func (m *mockInvoker) Destroy()            {}
func (m *mockInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	m.invoked = true
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, invocation)
	}
	return &mockResult{}
}

type mockResult struct {
	err   error
	value interface{}
}

func (m *mockResult) SetError(err error)                         { m.err = err }
func (m *mockResult) Error() error                               { return m.err }
func (m *mockResult) SetResult(value interface{})                { m.value = value }
func (m *mockResult) Result() interface{}                        { return m.value }
func (m *mockResult) SetAttachments(map[string]interface{})      {}
func (m *mockResult) Attachments() map[string]interface{}        { return nil }
func (m *mockResult) AddAttachment(string, interface{})          {}
func (m *mockResult) Attachment(string, interface{}) interface{} { return nil }

// TestInvoke tests various scenarios of the Invoke method
func TestInvoke(t *testing.T) {
	tests := []struct {
		name               string
		setupCtx           func() context.Context
		setupInvocation    func() *mockInvocation
		setupInvoker       func() *mockInvoker
		wantInvoked        bool
		validateResult     func(*testing.T, protocol.Result)
		validateContext    func(*testing.T, context.Context)
		validateInvocation func(*testing.T, *mockInvocation)
	}{
		{
			name: "with_xid_in_context",
			setupCtx: func() context.Context {
				ctx := tm.InitSeataContext(context.Background())
				tm.SetXID(ctx, "context-xid-123")
				return ctx
			},
			setupInvocation: func() *mockInvocation {
				return newMockInvocation()
			},
			setupInvoker: func() *mockInvoker {
				return &mockInvoker{}
			},
			wantInvoked: true,
			validateResult: func(t *testing.T, result protocol.Result) {
				assert.NotNil(t, result)
			},
			validateInvocation: func(t *testing.T, inv *mockInvocation) {
				seataXid, ok := inv.GetAttachment(constant.SeataXidKey)
				assert.True(t, ok)
				assert.Equal(t, "context-xid-123", seataXid)

				txXid, ok := inv.GetAttachment(constant.XidKey)
				assert.True(t, ok)
				assert.Equal(t, "context-xid-123", txXid)
			},
		},
		{
			name: "with_rpc_xid_only",
			setupCtx: func() context.Context {
				return context.Background()
			},
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(constant.SeataXidKey, "rpc-xid-456")
				return inv
			},
			setupInvoker: func() *mockInvoker {
				return &mockInvoker{
					invokeFunc: func(ctx context.Context, inv protocol.Invocation) protocol.Result {
						// Verify XID was set in context from RPC attachment
						xid := tm.GetXID(ctx)
						if xid != "rpc-xid-456" {
							return &mockResult{err: assert.AnError}
						}
						return &mockResult{}
					},
				}
			},
			wantInvoked: true,
			validateResult: func(t *testing.T, result protocol.Result) {
				assert.NotNil(t, result)
				assert.NoError(t, result.Error())
			},
		},
		{
			name: "no_xid",
			setupCtx: func() context.Context {
				return context.Background()
			},
			setupInvocation: func() *mockInvocation {
				return newMockInvocation()
			},
			setupInvoker: func() *mockInvoker {
				return &mockInvoker{}
			},
			wantInvoked: true,
			validateResult: func(t *testing.T, result protocol.Result) {
				assert.NotNil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &dubboTransactionFilter{}
			ctx := tt.setupCtx()
			invocation := tt.setupInvocation()
			invoker := tt.setupInvoker()

			result := filter.Invoke(ctx, invoker, invocation)

			assert.Equal(t, tt.wantInvoked, invoker.invoked)

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}

			if tt.validateContext != nil {
				tt.validateContext(t, ctx)
			}

			if tt.validateInvocation != nil {
				tt.validateInvocation(t, invocation)
			}
		})
	}
}

// TestGetRpcXid tests getRpcXid with various XID formats
func TestGetRpcXid(t *testing.T) {
	tests := []struct {
		name               string
		setupInvocation    func() *mockInvocation
		expectedXid        string
		expectedPrecedence string
	}{
		{
			name: "dubbo_go_xid",
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(constant.SeataXidKey, "dubbo-go-xid")
				return inv
			},
			expectedXid: "dubbo-go-xid",
		},
		{
			name: "dubbo_go_xid_lowercase",
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(strings.ToLower(constant.SeataXidKey), "dubbo-go-lower-xid")
				return inv
			},
			expectedXid: "dubbo-go-lower-xid",
		},
		{
			name: "dubbo_java_xid",
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(constant.XidKey, "dubbo-java-xid")
				return inv
			},
			expectedXid: "dubbo-java-xid",
		},
		{
			name: "dubbo_java_xid_lowercase",
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(strings.ToLower(constant.XidKey), "dubbo-java-lower-xid")
				return inv
			},
			expectedXid: "dubbo-java-lower-xid",
		},
		{
			name: "dubbo_go_takes_precedence",
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(constant.SeataXidKey, "dubbo-go-xid")
				inv.SetAttachment(constant.XidKey, "dubbo-java-xid")
				return inv
			},
			expectedXid:        "dubbo-go-xid",
			expectedPrecedence: "dubbo-go XID should take precedence",
		},
		{
			name: "empty_xid",
			setupInvocation: func() *mockInvocation {
				return newMockInvocation()
			},
			expectedXid: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &dubboTransactionFilter{}
			invocation := tt.setupInvocation()

			xid := filter.getRpcXid(invocation)

			if tt.expectedPrecedence != "" {
				assert.Equal(t, tt.expectedXid, xid, tt.expectedPrecedence)
			} else {
				assert.Equal(t, tt.expectedXid, xid)
			}
		})
	}
}

// TestGetDubboGoRpcXid tests getDubboGoRpcXid method
func TestGetDubboGoRpcXid(t *testing.T) {
	tests := []struct {
		name            string
		setupInvocation func() *mockInvocation
		expectedXid     string
	}{
		{
			name: "with_seata_xid",
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(constant.SeataXidKey, "go-xid-1")
				return inv
			},
			expectedXid: "go-xid-1",
		},
		{
			name: "with_lowercase_seata_xid",
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(strings.ToLower(constant.SeataXidKey), "go-xid-lower")
				return inv
			},
			expectedXid: "go-xid-lower",
		},
		{
			name: "empty",
			setupInvocation: func() *mockInvocation {
				return newMockInvocation()
			},
			expectedXid: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &dubboTransactionFilter{}
			invocation := tt.setupInvocation()

			xid := filter.getDubboGoRpcXid(invocation)

			assert.Equal(t, tt.expectedXid, xid)
		})
	}
}

// TestGetDubboJavaRpcXid tests getDubboJavaRpcXid method
func TestGetDubboJavaRpcXid(t *testing.T) {
	tests := []struct {
		name            string
		setupInvocation func() *mockInvocation
		expectedXid     string
	}{
		{
			name: "with_xid_key",
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(constant.XidKey, "java-xid-1")
				return inv
			},
			expectedXid: "java-xid-1",
		},
		{
			name: "with_lowercase_xid_key",
			setupInvocation: func() *mockInvocation {
				inv := newMockInvocation()
				inv.SetAttachment(strings.ToLower(constant.XidKey), "java-xid-lower")
				return inv
			},
			expectedXid: "java-xid-lower",
		},
		{
			name: "empty",
			setupInvocation: func() *mockInvocation {
				return newMockInvocation()
			},
			expectedXid: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &dubboTransactionFilter{}
			invocation := tt.setupInvocation()

			xid := filter.getDubboJavaRpcXid(invocation)

			assert.Equal(t, tt.expectedXid, xid)
		})
	}
}

// TestInitSeataDubbo tests InitSeataDubbo function
func TestInitSeataDubbo(t *testing.T) {
	assert.NotPanics(t, func() {
		InitSeataDubbo()
	})
}
