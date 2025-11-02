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

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/tm"
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
func (m *mockInvocation) AttributeByKey(key string, defaultValue interface{}) interface{} {
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
func (m *mockInvocation) ServiceKey() string                  { return "" }
func (m *mockInvocation) Invoker() protocol.Invoker           { return nil }
func (m *mockInvocation) SetInvoker(invoker protocol.Invoker) {}
func (m *mockInvocation) ActualMethodName() string            { return "mockMethod" }
func (m *mockInvocation) IsGenericInvocation() bool           { return false }
func (m *mockInvocation) GetAttachmentInterface(key string) interface{} {
	return m.attachments[key]
}
func (m *mockInvocation) GetAttachmentAsContext() context.Context     { return context.Background() }
func (m *mockInvocation) SetAttribute(key string, value interface{})  {}
func (m *mockInvocation) GetAttribute(key string) (interface{}, bool) { return nil, false }
func (m *mockInvocation) GetAttributeWithDefaultValue(key string, defaultValue interface{}) interface{} {
	return defaultValue
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

// TestInvoke_WithXidInContext tests Invoke when XID exists in context
func TestInvoke_WithXidInContext(t *testing.T) {
	filter := &dubboTransactionFilter{}
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "context-xid-123")

	invocation := newMockInvocation()
	invoker := &mockInvoker{}

	result := filter.Invoke(ctx, invoker, invocation)

	assert.NotNil(t, result)
	assert.True(t, invoker.invoked)

	// Verify XID was set in attachments for dubbo-go
	seataXid, ok := invocation.GetAttachment(constant.SeataXidKey)
	assert.True(t, ok)
	assert.Equal(t, "context-xid-123", seataXid)

	// Verify XID was set in attachments for dubbo-java
	txXid, ok := invocation.GetAttachment(constant.XidKey)
	assert.True(t, ok)
	assert.Equal(t, "context-xid-123", txXid)
}

// TestInvoke_WithRpcXidOnly tests Invoke when XID only exists in RPC attachment
func TestInvoke_WithRpcXidOnly(t *testing.T) {
	filter := &dubboTransactionFilter{}
	ctx := context.Background()

	invocation := newMockInvocation()
	invocation.SetAttachment(constant.SeataXidKey, "rpc-xid-456")

	var capturedCtx context.Context
	invoker := &mockInvoker{
		invokeFunc: func(ctx context.Context, inv protocol.Invocation) protocol.Result {
			capturedCtx = ctx
			return &mockResult{}
		},
	}

	result := filter.Invoke(ctx, invoker, invocation)

	assert.NotNil(t, result)
	assert.True(t, invoker.invoked)

	// Verify XID was set in context
	xid := tm.GetXID(capturedCtx)
	assert.Equal(t, "rpc-xid-456", xid)
}

// TestInvoke_NoXid tests Invoke when no XID exists
func TestInvoke_NoXid(t *testing.T) {
	filter := &dubboTransactionFilter{}
	ctx := context.Background()

	invocation := newMockInvocation()
	invoker := &mockInvoker{}

	result := filter.Invoke(ctx, invoker, invocation)

	assert.NotNil(t, result)
	assert.True(t, invoker.invoked)
}

// TestGetRpcXid_DubboGo tests getRpcXid with dubbo-go format
func TestGetRpcXid_DubboGo(t *testing.T) {
	filter := &dubboTransactionFilter{}
	invocation := newMockInvocation()
	invocation.SetAttachment(constant.SeataXidKey, "dubbo-go-xid")

	xid := filter.getRpcXid(invocation)

	assert.Equal(t, "dubbo-go-xid", xid)
}

// TestGetRpcXid_DubboGoLowercase tests getRpcXid with lowercase dubbo-go format
func TestGetRpcXid_DubboGoLowercase(t *testing.T) {
	filter := &dubboTransactionFilter{}
	invocation := newMockInvocation()
	invocation.SetAttachment(strings.ToLower(constant.SeataXidKey), "dubbo-go-lower-xid")

	xid := filter.getRpcXid(invocation)

	assert.Equal(t, "dubbo-go-lower-xid", xid)
}

// TestGetRpcXid_DubboJava tests getRpcXid with dubbo-java format
func TestGetRpcXid_DubboJava(t *testing.T) {
	filter := &dubboTransactionFilter{}
	invocation := newMockInvocation()
	invocation.SetAttachment(constant.XidKey, "dubbo-java-xid")

	xid := filter.getRpcXid(invocation)

	assert.Equal(t, "dubbo-java-xid", xid)
}

// TestGetRpcXid_DubboJavaLowercase tests getRpcXid with lowercase dubbo-java format
func TestGetRpcXid_DubboJavaLowercase(t *testing.T) {
	filter := &dubboTransactionFilter{}
	invocation := newMockInvocation()
	invocation.SetAttachment(strings.ToLower(constant.XidKey), "dubbo-java-lower-xid")

	xid := filter.getRpcXid(invocation)

	assert.Equal(t, "dubbo-java-lower-xid", xid)
}

// TestGetRpcXid_Precedence tests that dubbo-go XID takes precedence over dubbo-java
func TestGetRpcXid_Precedence(t *testing.T) {
	filter := &dubboTransactionFilter{}
	invocation := newMockInvocation()
	invocation.SetAttachment(constant.SeataXidKey, "dubbo-go-xid")
	invocation.SetAttachment(constant.XidKey, "dubbo-java-xid")

	xid := filter.getRpcXid(invocation)

	assert.Equal(t, "dubbo-go-xid", xid, "dubbo-go XID should take precedence")
}

// TestGetRpcXid_Empty tests getRpcXid with no XID
func TestGetRpcXid_Empty(t *testing.T) {
	filter := &dubboTransactionFilter{}
	invocation := newMockInvocation()

	xid := filter.getRpcXid(invocation)

	assert.Equal(t, "", xid)
}

// TestGetDubboGoRpcXid tests getDubboGoRpcXid method
func TestGetDubboGoRpcXid(t *testing.T) {
	filter := &dubboTransactionFilter{}

	t.Run("with_seata_xid", func(t *testing.T) {
		inv := newMockInvocation()
		inv.SetAttachment(constant.SeataXidKey, "go-xid-1")

		xid := filter.getDubboGoRpcXid(inv)
		assert.Equal(t, "go-xid-1", xid)
	})

	t.Run("with_lowercase_seata_xid", func(t *testing.T) {
		inv := newMockInvocation()
		inv.SetAttachment(strings.ToLower(constant.SeataXidKey), "go-xid-lower")

		xid := filter.getDubboGoRpcXid(inv)
		assert.Equal(t, "go-xid-lower", xid)
	})

	t.Run("empty", func(t *testing.T) {
		inv := newMockInvocation()

		xid := filter.getDubboGoRpcXid(inv)
		assert.Equal(t, "", xid)
	})
}

// TestGetDubboJavaRpcXid tests getDubboJavaRpcXid method
func TestGetDubboJavaRpcXid(t *testing.T) {
	filter := &dubboTransactionFilter{}

	t.Run("with_xid_key", func(t *testing.T) {
		inv := newMockInvocation()
		inv.SetAttachment(constant.XidKey, "java-xid-1")

		xid := filter.getDubboJavaRpcXid(inv)
		assert.Equal(t, "java-xid-1", xid)
	})

	t.Run("with_lowercase_xid_key", func(t *testing.T) {
		inv := newMockInvocation()
		inv.SetAttachment(strings.ToLower(constant.XidKey), "java-xid-lower")

		xid := filter.getDubboJavaRpcXid(inv)
		assert.Equal(t, "java-xid-lower", xid)
	})

	t.Run("empty", func(t *testing.T) {
		inv := newMockInvocation()

		xid := filter.getDubboJavaRpcXid(inv)
		assert.Equal(t, "", xid)
	})
}

// TestInitSeataDubbo tests InitSeataDubbo function
func TestInitSeataDubbo(t *testing.T) {
	assert.NotPanics(t, func() {
		InitSeataDubbo()
	})
}
