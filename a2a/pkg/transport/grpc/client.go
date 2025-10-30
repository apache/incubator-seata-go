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
	"fmt"
	"io"
	"sync"
	"time"

	"seata-go-ai-a2a/pkg/types"

	pb "seata-go-ai-a2a/pkg/proto/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client provides gRPC client implementation for A2A protocol
type Client struct {
	conn   *grpc.ClientConn
	client pb.A2AServiceClient

	mu     sync.RWMutex
	closed bool
}

// ClientConfig configures the gRPC client
type ClientConfig struct {
	Address     string
	Timeout     time.Duration
	GRPCOptions []grpc.DialOption
}

// NewClient creates a new gRPC client
func NewClient(config *ClientConfig) (*Client, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	// Set default timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Set default gRPC dial options
	grpcOptions := config.GRPCOptions
	if grpcOptions == nil {
		grpcOptions = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
			grpc.WithTimeout(timeout),
		}
	}

	// Create connection
	conn, err := grpc.Dial(config.Address, grpcOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", config.Address, err)
	}

	client := pb.NewA2AServiceClient(conn)

	return &Client{
		conn:   conn,
		client: client,
	}, nil
}

// Close closes the gRPC client connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.conn.Close()
}

// IsClosed returns true if the client is closed
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// A2A Protocol Methods Implementation

// SendMessage sends a message to the agent
func (c *Client) SendMessage(ctx context.Context, req *types.SendMessageRequest) (*types.Task, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq, err := types.SendMessageRequestToProto(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make gRPC call
	pbResp, err := c.client.SendMessage(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Extract task from response payload
	taskResp, ok := pbResp.Payload.(*pb.SendMessageResponse_Task)
	if !ok {
		return nil, fmt.Errorf("unexpected response payload type: %T", pbResp.Payload)
	}

	// Convert response from protobuf
	task, err := types.TaskFromProto(taskResp.Task)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return task, nil
}

// SendStreamingMessage sends a message and streams task updates
func (c *Client) SendStreamingMessage(ctx context.Context, req *types.SendMessageRequest) (*StreamConnection, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq, err := types.SendMessageRequestToProto(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Create streaming call
	stream, err := c.client.SendStreamingMessage(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Create stream connection
	streamConn := &StreamConnection{
		stream: stream,
		events: make(chan types.StreamResponse, 10),
		errors: make(chan error, 5),
		ctx:    ctx,
	}

	// Start receiving stream responses
	go streamConn.receiveResponses()

	return streamConn, nil
}

// GetTask retrieves a task by ID
func (c *Client) GetTask(ctx context.Context, req *types.GetTaskRequest) (*types.Task, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq := types.GetTaskRequestToProto(req)

	// Make gRPC call
	pbTask, err := c.client.GetTask(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Convert from protobuf
	task, err := types.TaskFromProto(pbTask)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return task, nil
}

// CancelTask cancels a task
func (c *Client) CancelTask(ctx context.Context, req *types.CancelTaskRequest) (*types.Task, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq := types.CancelTaskRequestToProto(req)

	// Make gRPC call
	pbTask, err := c.client.CancelTask(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Convert from protobuf
	task, err := types.TaskFromProto(pbTask)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return task, nil
}

// TaskSubscription subscribes to task updates
func (c *Client) TaskSubscription(ctx context.Context, req *types.TaskSubscriptionRequest) (*StreamConnection, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq := types.TaskSubscriptionRequestToProto(req)

	// Create streaming call
	stream, err := c.client.TaskSubscription(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Create stream connection
	streamConn := &StreamConnection{
		stream: stream,
		events: make(chan types.StreamResponse, 10),
		errors: make(chan error, 5),
		ctx:    ctx,
	}

	// Start receiving stream responses
	go streamConn.receiveResponses()

	return streamConn, nil
}

// ListTasks lists tasks with optional filtering
func (c *Client) ListTasks(ctx context.Context, req *types.ListTasksRequest) (*types.ListTasksResponse, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq := types.ListTasksRequestToProto(req)

	// Make gRPC call
	pbResp, err := c.client.ListTasks(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Convert from protobuf
	response, err := types.ListTasksResponseFromProto(pbResp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return response, nil
}

// GetAgentCard retrieves the agent card
func (c *Client) GetAgentCard(ctx context.Context) (*types.AgentCard, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Create empty request
	pbReq := &pb.GetAgentCardRequest{}

	// Make gRPC call
	pbAgentCard, err := c.client.GetAgentCard(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Convert from protobuf
	agentCard, err := types.AgentCardFromProto(pbAgentCard)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return agentCard, nil
}

// CreateTaskPushNotificationConfig creates a push notification config
func (c *Client) CreateTaskPushNotificationConfig(ctx context.Context, req *types.CreateTaskPushNotificationConfigRequest) (*types.TaskPushNotificationConfig, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq, err := types.CreateTaskPushNotificationConfigRequestToProto(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make gRPC call
	pbConfig, err := c.client.CreateTaskPushNotification(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Convert from protobuf
	config, err := types.TaskPushNotificationConfigFromProto(pbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return config, nil
}

// GetTaskPushNotificationConfig retrieves a push notification config
func (c *Client) GetTaskPushNotificationConfig(ctx context.Context, req *types.GetTaskPushNotificationConfigRequest) (*types.TaskPushNotificationConfig, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq := types.GetTaskPushNotificationConfigRequestToProto(req)

	// Make gRPC call
	pbConfig, err := c.client.GetTaskPushNotification(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Convert from protobuf
	config, err := types.TaskPushNotificationConfigFromProto(pbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return config, nil
}

// ListTaskPushNotificationConfigs lists push notification configs
func (c *Client) ListTaskPushNotificationConfigs(ctx context.Context, req *types.ListTaskPushNotificationConfigRequest) (*types.ListTaskPushNotificationConfigResponse, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq := types.ListTaskPushNotificationConfigRequestToProto(req)

	// Make gRPC call
	pbResp, err := c.client.ListTaskPushNotification(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Convert from protobuf
	response, err := types.ListTaskPushNotificationConfigResponseFromProto(pbResp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return response, nil
}

// DeleteTaskPushNotificationConfig deletes a push notification config
func (c *Client) DeleteTaskPushNotificationConfig(ctx context.Context, req *types.DeleteTaskPushNotificationConfigRequest) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Convert to protobuf request
	pbReq := types.DeleteTaskPushNotificationConfigRequestToProto(req)

	// Make gRPC call
	_, err := c.client.DeleteTaskPushNotification(ctx, pbReq)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %w", err)
	}

	return nil
}

// StreamConnection represents a gRPC streaming connection
type StreamConnection struct {
	stream pb.A2AService_SendStreamingMessageClient
	events chan types.StreamResponse
	errors chan error
	ctx    context.Context

	mu     sync.Mutex
	closed bool
}

// Events returns the events channel for this stream
func (sc *StreamConnection) Events() <-chan types.StreamResponse {
	return sc.events
}

// Errors returns the errors channel for this stream
func (sc *StreamConnection) Errors() <-chan error {
	return sc.errors
}

// Close closes the stream connection
func (sc *StreamConnection) Close() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if !sc.closed {
		sc.closed = true
		close(sc.events)
		close(sc.errors)
	}
}

// IsClosed returns true if the connection is closed
func (sc *StreamConnection) IsClosed() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.closed
}

// receiveResponses receives responses from the gRPC stream
func (sc *StreamConnection) receiveResponses() {
	defer sc.Close()

	for {
		// Check if context is cancelled
		select {
		case <-sc.ctx.Done():
			select {
			case sc.errors <- sc.ctx.Err():
			default:
			}
			return
		default:
		}

		// Receive from stream
		pbResp, err := sc.stream.Recv()
		if err != nil {
			if err == io.EOF {
				// Stream ended normally
				return
			}

			// Send error to error channel
			select {
			case sc.errors <- fmt.Errorf("stream receive error: %w", err):
			case <-sc.ctx.Done():
			}
			return
		}

		// Convert protobuf response to types
		streamResp, err := convertStreamResponseFromProto(pbResp)
		if err != nil {
			select {
			case sc.errors <- fmt.Errorf("failed to convert stream response: %w", err):
			case <-sc.ctx.Done():
			}
			continue
		}

		// Send to events channel
		select {
		case sc.events <- streamResp:
		case <-sc.ctx.Done():
			return
		}
	}
}

// GetConnectionInfo returns information about the gRPC connection
func (c *Client) GetConnectionInfo() (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return "", false
	}

	return c.conn.Target(), !c.closed
}

// Ping tests the connection to the server by calling GetAgentCard
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.GetAgentCard(ctx)
	return err
}
