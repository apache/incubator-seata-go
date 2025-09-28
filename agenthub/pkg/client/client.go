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

// Package client provides a Go SDK for AgentHub API
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client represents the AgentHub client
type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
}

// ClientConfig holds configuration for the client
type ClientConfig struct {
	BaseURL   string
	Timeout   time.Duration
	AuthToken string
}

// NewClient creates a new AgentHub client
func NewClient(config ClientConfig) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8080"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		authToken: config.AuthToken,
	}
}

// SetAuthToken sets the authentication token
func (c *Client) SetAuthToken(token string) {
	c.authToken = token
}

// doRequest performs HTTP request with common error handling
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// parseResponse parses the HTTP response into the given struct
func (c *Client) parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    errResp.Error.Error,
			Code:       errResp.Error.Code,
		}
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// HealthCheck performs a health check
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	resp, err := c.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return nil, err
	}

	var result HealthResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// RegisterAgent registers a new agent
func (c *Client) RegisterAgent(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	resp, err := c.doRequest(ctx, "POST", "/agent/register", req)
	if err != nil {
		return nil, err
	}

	var result APIResponse[RegisterResponse]
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// DiscoverAgents discovers agents based on query
func (c *Client) DiscoverAgents(ctx context.Context, req *DiscoverRequest) (*DiscoverResponse, error) {
	resp, err := c.doRequest(ctx, "POST", "/agent/discover", req)
	if err != nil {
		return nil, err
	}

	var result APIResponse[DiscoverResponse]
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetAgent retrieves a specific agent by ID
func (c *Client) GetAgent(ctx context.Context, agentID string) (*RegisteredAgent, error) {
	endpoint := fmt.Sprintf("/agent/get?id=%s", url.QueryEscape(agentID))
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result APIResponse[RegisteredAgent]
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// ListAgents lists all registered agents
func (c *Client) ListAgents(ctx context.Context) ([]*RegisteredAgent, error) {
	resp, err := c.doRequest(ctx, "GET", "/agents", nil)
	if err != nil {
		return nil, err
	}

	var result APIResponse[[]*RegisteredAgent]
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return *result.Data, nil
}

// UpdateAgentStatus updates an agent's status
func (c *Client) UpdateAgentStatus(ctx context.Context, agentID, status string) error {
	endpoint := fmt.Sprintf("/agent/status?id=%s", url.QueryEscape(agentID))
	body := map[string]string{"status": status}

	resp, err := c.doRequest(ctx, "PUT", endpoint, body)
	if err != nil {
		return err
	}

	var result APIResponse[interface{}]
	return c.parseResponse(resp, &result)
}

// RemoveAgent removes an agent
func (c *Client) RemoveAgent(ctx context.Context, agentID string) error {
	endpoint := fmt.Sprintf("/agent/remove?id=%s", url.QueryEscape(agentID))
	resp, err := c.doRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	var result APIResponse[interface{}]
	return c.parseResponse(resp, &result)
}

// SendHeartbeat sends a heartbeat for an agent
func (c *Client) SendHeartbeat(ctx context.Context, agentID string) error {
	endpoint := fmt.Sprintf("/agent/heartbeat?id=%s", url.QueryEscape(agentID))
	resp, err := c.doRequest(ctx, "POST", endpoint, nil)
	if err != nil {
		return err
	}

	var result APIResponse[interface{}]
	return c.parseResponse(resp, &result)
}

// AnalyzeContext performs dynamic context analysis
func (c *Client) AnalyzeContext(ctx context.Context, req *ContextAnalysisRequest) (*ContextAnalysisResponse, error) {
	resp, err := c.doRequest(ctx, "POST", "/agent/analyze", req)
	if err != nil {
		return nil, err
	}

	var result APIResponse[ContextAnalysisResponse]
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetMetrics retrieves system metrics
func (c *Client) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.doRequest(ctx, "GET", "/metrics", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Auth methods

// GetAuthToken requests an authentication token
func (c *Client) GetAuthToken(ctx context.Context, username string) (*AuthTokenResponse, error) {
	body := map[string]string{"username": username}
	resp, err := c.doRequest(ctx, "POST", "/auth/token", body)
	if err != nil {
		return nil, err
	}

	var result APIResponse[AuthTokenResponse]
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// RefreshAuthToken refreshes an authentication token
func (c *Client) RefreshAuthToken(ctx context.Context, refreshToken string) (*AuthTokenResponse, error) {
	body := map[string]string{"refresh_token": refreshToken}
	resp, err := c.doRequest(ctx, "POST", "/auth/refresh", body)
	if err != nil {
		return nil, err
	}

	var result APIResponse[AuthTokenResponse]
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
