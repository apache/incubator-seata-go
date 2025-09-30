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

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"seata-go-ai-a2a/pkg/types"
)

// Client represents a RESTful HTTP client for A2A protocol
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// ClientConfig configures the REST client
type ClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// NewClient creates a new REST client
func NewClient(config *ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// doRequest performs an HTTP request and returns the response
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader

	// Marshal request body if provided
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	httpReq, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "A2A-REST-Client/1.0")

	// Make HTTP request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle non-2xx status codes
	if httpResp.StatusCode >= 400 {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if errorData, ok := errorResp["error"].(map[string]interface{}); ok {
				if message, ok := errorData["message"].(string); ok {
					return &RESTError{
						StatusCode: httpResp.StatusCode,
						Message:    message,
						Details:    fmt.Sprintf("%v", errorData["details"]),
					}
				}
			}
		}
		return &RESTError{
			StatusCode: httpResp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d", httpResp.StatusCode),
			Details:    string(respBody),
		}
	}

	// Unmarshal result if provided
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// A2A Protocol Methods

// SendMessage sends a message using the A2A REST API
func (c *Client) SendMessage(ctx context.Context, req *types.SendMessageRequest) (*types.TaskSendMessageResponse, error) {
	var resp types.TaskSendMessageResponse
	err := c.doRequest(ctx, http.MethodPost, "/api/v1/messages", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAgentCard retrieves the agent card via REST API
func (c *Client) GetAgentCard(ctx context.Context) (*types.AgentCard, error) {
	var agentCard types.AgentCard
	err := c.doRequest(ctx, http.MethodGet, "/api/v1/agent-card", nil, &agentCard)
	if err != nil {
		return nil, err
	}
	return &agentCard, nil
}

// GetTask retrieves a task by ID via REST API
func (c *Client) GetTask(ctx context.Context, req *types.GetTaskRequest) (*types.Task, error) {
	// Extract task ID from Name field (format: tasks/{task_id})
	taskID := strings.TrimPrefix(req.Name, "tasks/")
	path := fmt.Sprintf("/api/v1/tasks/%s", url.PathEscape(taskID))

	var task types.Task
	err := c.doRequest(ctx, http.MethodGet, path, nil, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// CancelTask cancels a task via REST API
func (c *Client) CancelTask(ctx context.Context, req *types.CancelTaskRequest) (*types.Task, error) {
	// Extract task ID from Name field (format: tasks/{task_id})
	taskID := strings.TrimPrefix(req.Name, "tasks/")
	path := fmt.Sprintf("/api/v1/tasks/%s", url.PathEscape(taskID))

	var task types.Task
	err := c.doRequest(ctx, http.MethodDelete, path, nil, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks lists tasks with optional filtering via REST API
func (c *Client) ListTasks(ctx context.Context, req *types.ListTasksRequest) (*types.ListTasksResponse, error) {
	path := "/api/v1/tasks"

	// Add query parameters for filtering
	params := url.Values{}
	if req.ContextID != "" {
		params.Set("contextId", req.ContextID)
	}
	// Convert states to string representation
	for _, state := range req.States {
		params.Add("state", state.String())
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var resp types.ListTasksResponse
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateTaskPushNotificationConfig creates a push notification config via REST API
func (c *Client) CreateTaskPushNotificationConfig(ctx context.Context, req *types.CreateTaskPushNotificationConfigRequest) (*types.TaskPushNotificationConfig, error) {
	var config types.TaskPushNotificationConfig
	err := c.doRequest(ctx, http.MethodPost, "/api/v1/push-notification-configs", req, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetTaskPushNotificationConfig retrieves a push notification config via REST API
func (c *Client) GetTaskPushNotificationConfig(ctx context.Context, req *types.GetTaskPushNotificationConfigRequest) (*types.TaskPushNotificationConfig, error) {
	// Extract config ID from Name field (format: tasks/{task_id}/pushNotificationConfigs/{config_id})
	parts := strings.Split(req.Name, "/")
	var configID string
	if len(parts) >= 4 {
		configID = parts[3]
	}
	path := fmt.Sprintf("/api/v1/push-notification-configs/%s", url.PathEscape(configID))

	var config types.TaskPushNotificationConfig
	err := c.doRequest(ctx, http.MethodGet, path, nil, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ListTaskPushNotificationConfig lists push notification configs via REST API
func (c *Client) ListTaskPushNotificationConfig(ctx context.Context, req *types.ListTaskPushNotificationConfigRequest) (*types.ListTaskPushNotificationConfigResponse, error) {
	var resp types.ListTaskPushNotificationConfigResponse
	err := c.doRequest(ctx, http.MethodGet, "/api/v1/push-notification-configs", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteTaskPushNotificationConfig deletes a push notification config via REST API
func (c *Client) DeleteTaskPushNotificationConfig(ctx context.Context, req *types.DeleteTaskPushNotificationConfigRequest) error {
	// Extract config ID from Name field (format: tasks/{task_id}/pushNotificationConfigs/{config_id})
	parts := strings.Split(req.Name, "/")
	var configID string
	if len(parts) >= 4 {
		configID = parts[3]
	}
	path := fmt.Sprintf("/api/v1/push-notification-configs/%s", url.PathEscape(configID))
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

// RESTError represents an error returned by the REST API
type RESTError struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
}

func (e *RESTError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("REST API error %d: %s (%s)", e.StatusCode, e.Message, e.Details)
	}
	return fmt.Sprintf("REST API error %d: %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if this is a 404 error
func (e *RESTError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsBadRequest returns true if this is a 400 error
func (e *RESTError) IsBadRequest() bool {
	return e.StatusCode == http.StatusBadRequest
}

// IsUnauthorized returns true if this is a 401 error
func (e *RESTError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsForbidden returns true if this is a 403 error
func (e *RESTError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsInternalServerError returns true if this is a 500 error
func (e *RESTError) IsInternalServerError() bool {
	return e.StatusCode == http.StatusInternalServerError
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

// GetBaseURL returns the client base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}
