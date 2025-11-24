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

package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/tm"
)

func TestClientTransactionInterceptor_WithSeataContext(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "test-client-xid-123")

	invoked := false
	var capturedCtx context.Context

	mockInvoker := func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		invoked = true
		capturedCtx = ctx
		return nil
	}

	err := ClientTransactionInterceptor(ctx, "/test.Service/Method", nil, nil, nil, mockInvoker)

	assert.NoError(t, err)
	assert.True(t, invoked)
	assert.NotNil(t, capturedCtx)

	// Verify XID was added to metadata
	md, ok := metadata.FromOutgoingContext(capturedCtx)
	assert.True(t, ok)
	xidSlice := md.Get(constant.XidKey)
	assert.NotEmpty(t, xidSlice)
	assert.Equal(t, "test-client-xid-123", xidSlice[0])
}

func TestClientTransactionInterceptor_WithoutSeataContext(t *testing.T) {
	ctx := context.Background()

	invoked := false
	var capturedCtx context.Context

	mockInvoker := func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		invoked = true
		capturedCtx = ctx
		return nil
	}

	err := ClientTransactionInterceptor(ctx, "/test.Service/Method", nil, nil, nil, mockInvoker)

	assert.NoError(t, err)
	assert.True(t, invoked)
	assert.NotNil(t, capturedCtx)

	// Verify no XID metadata was added
	md, ok := metadata.FromOutgoingContext(capturedCtx)
	if ok {
		xidSlice := md.Get(constant.XidKey)
		assert.Empty(t, xidSlice)
	}
}

func TestClientTransactionInterceptor_InvokerError(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "test-xid")

	expectedErr := errors.New("mock invoker error")
	mockInvoker := func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return expectedErr
	}

	err := ClientTransactionInterceptor(ctx, "/test.Service/Method", nil, nil, nil, mockInvoker)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestClientTransactionInterceptor_EmptyXID(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "")

	var capturedCtx context.Context
	mockInvoker := func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		capturedCtx = ctx
		return nil
	}

	err := ClientTransactionInterceptor(ctx, "/test.Service/Method", nil, nil, nil, mockInvoker)

	assert.NoError(t, err)

	// Even with empty XID, metadata should be set if Seata context exists
	_, ok := metadata.FromOutgoingContext(capturedCtx)
	assert.True(t, ok)
}

func TestServerTransactionInterceptor_WithXidKey(t *testing.T) {
	md := metadata.New(map[string]string{
		constant.XidKey: "server-xid-456",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handlerCalled := false
	var handlerCtx context.Context

	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		handlerCtx = ctx
		return "response", nil
	}

	resp, err := ServerTransactionInterceptor(ctx, nil, nil, mockHandler)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, "response", resp)
	assert.NotNil(t, handlerCtx)

	// Verify XID was set in context
	xid := tm.GetXID(handlerCtx)
	assert.Equal(t, "server-xid-456", xid)
}

func TestServerTransactionInterceptor_WithXidKeyLowercase(t *testing.T) {
	md := metadata.New(map[string]string{
		constant.XidKeyLowercase: "server-xid-lowercase-789",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var handlerCtx context.Context
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCtx = ctx
		return "response", nil
	}

	resp, err := ServerTransactionInterceptor(ctx, nil, nil, mockHandler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)

	xid := tm.GetXID(handlerCtx)
	assert.Equal(t, "server-xid-lowercase-789", xid)
}

func TestServerTransactionInterceptor_XidKeyPrecedence(t *testing.T) {
	md := metadata.New(map[string]string{
		constant.XidKey:          "primary-xid",
		constant.XidKeyLowercase: "secondary-xid",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var handlerCtx context.Context
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCtx = ctx
		return "response", nil
	}

	_, err := ServerTransactionInterceptor(ctx, nil, nil, mockHandler)

	assert.NoError(t, err)

	xid := tm.GetXID(handlerCtx)
	assert.Equal(t, "primary-xid", xid)
}

func TestServerTransactionInterceptor_NoMetadata(t *testing.T) {
	ctx := context.Background()

	handlerCalled := false
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := ServerTransactionInterceptor(ctx, nil, nil, mockHandler)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, "response", resp)
}

func TestServerTransactionInterceptor_NilMetadataSlice(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	handlerCalled := false
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := ServerTransactionInterceptor(ctx, nil, nil, mockHandler)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, "response", resp)
}

func TestServerTransactionInterceptor_EmptyXid(t *testing.T) {
	md := metadata.New(map[string]string{
		constant.XidKey: "",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handlerCalled := false
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := ServerTransactionInterceptor(ctx, nil, nil, mockHandler)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, "response", resp)
}

func TestServerTransactionInterceptor_OnlyLowercaseXid(t *testing.T) {
	md := metadata.New(map[string]string{
		constant.XidKeyLowercase: "fallback-xid",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var handlerCtx context.Context
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCtx = ctx
		return "response", nil
	}

	_, err := ServerTransactionInterceptor(ctx, nil, nil, mockHandler)

	assert.NoError(t, err)

	xid := tm.GetXID(handlerCtx)
	assert.Equal(t, "fallback-xid", xid)
}

func TestServerTransactionInterceptor_HandlerError(t *testing.T) {
	md := metadata.New(map[string]string{
		constant.XidKey: "test-xid",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	expectedErr := errors.New("handler error")
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	resp, err := ServerTransactionInterceptor(ctx, nil, nil, mockHandler)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, resp)
}

func TestServerTransactionInterceptor_MultipleXidValues(t *testing.T) {
	md := metadata.MD{
		constant.XidKey: []string{"xid-1", "xid-2", "xid-3"},
	}
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var handlerCtx context.Context
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCtx = ctx
		return "response", nil
	}

	_, err := ServerTransactionInterceptor(ctx, nil, nil, mockHandler)

	assert.NoError(t, err)

	// Should use the first value
	xid := tm.GetXID(handlerCtx)
	assert.Equal(t, "xid-1", xid)
}

func TestClientTransactionInterceptor_Timing(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "timing-test-xid")

	mockInvoker := func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		// Simulate some work
		return nil
	}

	err := ClientTransactionInterceptor(ctx, "/test.Service/Method", nil, nil, nil, mockInvoker)

	assert.NoError(t, err)
}

func TestClientAndServerInterceptor_Integration(t *testing.T) {
	// Simulate client sending request
	clientCtx := tm.InitSeataContext(context.Background())
	tm.SetXID(clientCtx, "integration-xid-999")

	var serverCtx context.Context

	// Mock client invoker that simulates server receiving the request
	mockInvoker := func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, opts ...grpc.CallOption) error {

		// Extract metadata from client context
		md, ok := metadata.FromOutgoingContext(ctx)
		assert.True(t, ok)

		// Simulate server receiving this metadata
		incomingCtx := metadata.NewIncomingContext(context.Background(), md)

		// Server interceptor processes it
		mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			serverCtx = ctx
			return "ok", nil
		}

		_, err := ServerTransactionInterceptor(incomingCtx, req, nil, mockHandler)
		return err
	}

	err := ClientTransactionInterceptor(clientCtx, "/test.Service/Method", nil, nil, nil, mockInvoker)

	assert.NoError(t, err)
	assert.NotNil(t, serverCtx)

	// Verify XID propagated from client to server
	serverXid := tm.GetXID(serverCtx)
	assert.Equal(t, "integration-xid-999", serverXid)
}
