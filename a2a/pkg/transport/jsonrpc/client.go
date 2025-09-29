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

package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"seata-go-ai-a2a/pkg/types"
)

// Client represents a JSON-RPC 2.0 client for A2A protocol
type Client struct {
	endpoint   string
	httpClient *http.Client
	requestID  int64
}

// ClientConfig configures the JSON-RPC client
type ClientConfig struct {
	Endpoint string
	Timeout  time.Duration
}

// NewClient creates a new JSON-RPC client
func NewClient(config *ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		endpoint: config.Endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		requestID: 0,
	}
}

// nextRequestID generates the next request ID
func (c *Client) nextRequestID() int64 {
	return atomic.AddInt64(&c.requestID, 1)
}

// Call makes a JSON-RPC call and returns the result
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	req := NewRequest(method, params, c.nextRequestID())

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	if resp.IsError() {
		return &JSONRPCError{
			Code:    resp.Error.Code,
			Message: resp.Error.Message,
			Data:    resp.Error.Data,
		}
	}

	// Unmarshal result if provided
	if result != nil && resp.Result != nil {
		resultJSON, err := json.Marshal(resp.Result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}

		if err := json.Unmarshal(resultJSON, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return nil
}

// Notify makes a JSON-RPC notification (no response expected)
func (c *Client) Notify(ctx context.Context, method string, params interface{}) error {
	req := &Request{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  params,
		// No ID for notifications
	}

	_, err := c.doRequest(ctx, req)
	return err
}

// doRequest performs the actual HTTP request
func (c *Client) doRequest(ctx context.Context, req *Request) (*Response, error) {
	// Marshal request to JSON
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "A2A-JSONRPC-Client/1.0")

	// Make HTTP request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// For notifications, we don't expect a response
	if req.IsNotification() {
		return nil, nil
	}

	// Parse JSON-RPC response
	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Validate response
	if err := ValidateResponse(&resp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &resp, nil
}

// A2A Protocol Methods

// SendMessage sends a message using the A2A protocol
func (c *Client) SendMessage(ctx context.Context, req *types.SendMessageRequest) (*types.SendMessageResponse, error) {
	var resp types.SendMessageResponse
	err := c.Call(ctx, "message/send", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAgentCard retrieves the agent card
func (c *Client) GetAgentCard(ctx context.Context) (*types.AgentCard, error) {
	var agentCard types.AgentCard
	err := c.Call(ctx, "agentCard/get", nil, &agentCard)
	if err != nil {
		return nil, err
	}
	return &agentCard, nil
}

// GetTask retrieves a task by name/ID
func (c *Client) GetTask(ctx context.Context, req *types.GetTaskRequest) (*types.Task, error) {
	var task types.Task
	err := c.Call(ctx, "tasks/get", req, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// CancelTask cancels a task
func (c *Client) CancelTask(ctx context.Context, req *types.CancelTaskRequest) (*types.Task, error) {
	var task types.Task
	err := c.Call(ctx, "tasks/cancel", req, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks lists tasks with optional filtering
func (c *Client) ListTasks(ctx context.Context, req *types.ListTasksRequest) (*types.ListTasksResponse, error) {
	var resp types.ListTasksResponse
	err := c.Call(ctx, "tasks/list", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateTaskPushNotificationConfig creates a push notification config
func (c *Client) CreateTaskPushNotificationConfig(ctx context.Context, req *types.CreateTaskPushNotificationConfigRequest) (*types.TaskPushNotificationConfig, error) {
	var config types.TaskPushNotificationConfig
	err := c.Call(ctx, "tasks/pushNotificationConfig/set", req, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetTaskPushNotificationConfig retrieves a push notification config
func (c *Client) GetTaskPushNotificationConfig(ctx context.Context, req *types.GetTaskPushNotificationConfigRequest) (*types.TaskPushNotificationConfig, error) {
	var config types.TaskPushNotificationConfig
	err := c.Call(ctx, "tasks/pushNotificationConfig/get", req, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ListTaskPushNotificationConfig lists push notification configs
func (c *Client) ListTaskPushNotificationConfig(ctx context.Context, req *types.ListTaskPushNotificationConfigRequest) (*types.ListTaskPushNotificationConfigResponse, error) {
	var resp types.ListTaskPushNotificationConfigResponse
	err := c.Call(ctx, "tasks/pushNotificationConfig/list", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteTaskPushNotificationConfig deletes a push notification config
func (c *Client) DeleteTaskPushNotificationConfig(ctx context.Context, req *types.DeleteTaskPushNotificationConfigRequest) error {
	return c.Call(ctx, "tasks/pushNotificationConfig/delete", req, nil)
}

// JSONRPCError represents a JSON-RPC error returned by the client
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// IsA2AError returns true if this is an A2A specific error code
func (e *JSONRPCError) IsA2AError() bool {
	return e.Code >= -32007 && e.Code <= -32001
}

// IsTaskNotFound returns true if this is a TaskNotFoundError
func (e *JSONRPCError) IsTaskNotFound() bool {
	return e.Code == TaskNotFoundError
}

// IsTaskNotCancelable returns true if this is a TaskNotCancelableError
func (e *JSONRPCError) IsTaskNotCancelable() bool {
	return e.Code == TaskNotCancelableError
}

// Close closes the client and cleans up resources
func (c *Client) Close() error {
	// Close HTTP client connections
	c.httpClient.CloseIdleConnections()
	return nil
}

// SetTimeout sets the client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// GetEndpoint returns the client endpoint
func (c *Client) GetEndpoint() string {
	return c.endpoint
}
