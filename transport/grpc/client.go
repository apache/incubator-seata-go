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
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"seata-go-ai-transport/common"
)

// Client implements the common.Client interface for gRPC
type Client struct {
	conn    *grpc.ClientConn
	client  interface{} // Generic gRPC client
	closed  int32
	mu      sync.RWMutex
	methods map[string]MethodInfo
}

// MethodInfo contains information about a gRPC method
type MethodInfo struct {
	FullName    string
	IsStreaming bool
	InputType   proto.Message
	OutputType  proto.Message
}

// ClientConfig represents gRPC client configuration
type ClientConfig struct {
	Target   string                `json:"target"`
	Insecure bool                  `json:"insecure"`
	Timeout  time.Duration         `json:"timeout"`
	Options  []grpc.DialOption     `json:"-"`
	Methods  map[string]MethodInfo `json:"-"`
}

var _ common.Client = (*Client)(nil)

// NewClient creates a new gRPC client
func NewClient(config *ClientConfig) (*Client, error) {
	opts := config.Options
	if opts == nil {
		opts = []grpc.DialOption{}
	}

	if config.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if config.Timeout > 0 {
		opts = append(opts, grpc.WithTimeout(config.Timeout))
	}

	conn, err := grpc.NewClient(config.Target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	client := &Client{
		conn:    conn,
		methods: make(map[string]MethodInfo),
	}

	if config.Methods != nil {
		for method, info := range config.Methods {
			client.methods[method] = info
		}
	}

	return client, nil
}

// Call makes a unary RPC call
func (c *Client) Call(ctx context.Context, msg *common.Message) (*common.Response, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, common.ErrClientClosed
	}

	// Get method info
	c.mu.RLock()
	methodInfo, exists := c.methods[msg.Method]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("method %s not registered", msg.Method)
	}

	if methodInfo.IsStreaming {
		return nil, fmt.Errorf("method %s is a streaming method, use Stream() instead", msg.Method)
	}

	// Create input message
	input := proto.Clone(methodInfo.InputType)
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal(msg.Payload, input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request: %w", err)
		}
	}

	// Create output message
	output := proto.Clone(methodInfo.OutputType)

	// Make the call
	err := c.conn.Invoke(ctx, methodInfo.FullName, input, output)
	if err != nil {
		return &common.Response{
			Headers: make(map[string]string),
			Error:   err,
		}, nil
	}

	// Marshal response
	respData, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &common.Response{
		Data:    respData,
		Headers: make(map[string]string),
	}, nil
}

// Stream makes a streaming RPC call
func (c *Client) Stream(ctx context.Context, msg *common.Message) (common.StreamReader, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, common.ErrClientClosed
	}

	// Get method info
	c.mu.RLock()
	methodInfo, exists := c.methods[msg.Method]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("method %s not registered", msg.Method)
	}

	if !methodInfo.IsStreaming {
		return nil, fmt.Errorf("method %s is not a streaming method, use Call() instead", msg.Method)
	}

	// Create input message
	input := proto.Clone(methodInfo.InputType)
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal(msg.Payload, input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request: %w", err)
		}
	}

	// Create streaming description
	streamDesc := &grpc.StreamDesc{
		StreamName:    extractMethodName(methodInfo.FullName),
		ServerStreams: true,
		ClientStreams: false,
	}

	// Start stream
	stream, err := c.conn.NewStream(ctx, streamDesc, methodInfo.FullName)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Send input message
	if err := stream.SendMsg(input); err != nil {
		_ = stream.CloseSend()
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Close sending side
	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send: %w", err)
	}

	return NewGRPCStreamReader(ctx, stream, methodInfo.OutputType), nil
}

// Protocol returns the underlying protocol
func (c *Client) Protocol() common.Protocol {
	return common.ProtocolGRPC
}

// Close closes the client connection
func (c *Client) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return common.ErrClientClosed
	}

	return c.conn.Close()
}

// RegisterMethod registers a method with the client
func (c *Client) RegisterMethod(method string, info MethodInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.methods[method] = info
}

// extractMethodName extracts the method name from full method name
func extractMethodName(fullMethod string) string {
	// Extract method name from full method path like "/package.Service/Method"
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[i+1:]
		}
	}
	return fullMethod
}

// SetClient sets the underlying gRPC client (for custom clients)
func (c *Client) SetClient(client interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.client = client
}

// GetClient gets the underlying gRPC client
func (c *Client) GetClient() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// GetConnection gets the underlying gRPC connection
func (c *Client) GetConnection() *grpc.ClientConn {
	return c.conn
}
