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
	"sync"
	"sync/atomic"
	"time"

	"seata-go-ai-transport/common"
)

// Client implements the common.Client interface for JSON-RPC
type Client struct {
	endpoint   string
	httpClient *http.Client
	headers    map[string]string
	idCounter  int64
	closed     int32
	mu         sync.RWMutex
}

// ClientConfig represents JSON-RPC client configuration
type ClientConfig struct {
	Endpoint string            `json:"endpoint"`
	Timeout  time.Duration     `json:"timeout"`
	Headers  map[string]string `json:"headers"`
}

var _ common.Client = (*Client)(nil)

// NewClient creates a new JSON-RPC client
func NewClient(config *ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	headers := make(map[string]string)
	if config.Headers != nil {
		for k, v := range config.Headers {
			headers[k] = v
		}
	}

	// Set default content type if not specified
	if _, exists := headers["Content-Type"]; !exists {
		headers["Content-Type"] = "application/json"
	}

	return &Client{
		endpoint: config.Endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		headers: headers,
	}
}

// Call makes a unary RPC call
func (c *Client) Call(ctx context.Context, msg *common.Message) (*common.Response, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, common.ErrClientClosed
	}

	// Generate unique ID for the request
	id := atomic.AddInt64(&c.idCounter, 1)

	// Create JSON-RPC request
	var params interface{}
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal(msg.Payload, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
	}

	req, err := NewRequest(msg.Method, params, id)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Marshal request to JSON
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	c.mu.RLock()
	for k, v := range c.headers {
		httpReq.Header.Set(k, v)
	}
	// Add message headers
	for k, v := range msg.Headers {
		httpReq.Header.Set(k, v)
	}
	c.mu.RUnlock()

	// Send request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(httpResp.Body)

	// Read response body
	respData, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", httpResp.StatusCode, string(respData))
	}

	// Parse JSON-RPC response
	var jsonResp JSONRPCResponse
	if err := json.Unmarshal(respData, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Create common response
	resp := &common.Response{
		Headers: make(map[string]string),
	}

	// Copy HTTP headers to response headers
	for k, values := range httpResp.Header {
		if len(values) > 0 {
			resp.Headers[k] = values[0]
		}
	}

	// Handle JSON-RPC error
	if jsonResp.Error != nil {
		resp.Error = jsonResp.Error
		return resp, nil
	}

	// Set response data
	resp.Data = jsonResp.Result

	return resp, nil
}

// Stream makes a streaming RPC call (not supported for basic JSON-RPC)
func (c *Client) Stream(ctx context.Context, msg *common.Message) (common.StreamReader, error) {
	return nil, fmt.Errorf("streaming not supported by basic JSON-RPC client, use SSE client instead")
}

// Protocol returns the underlying protocol
func (c *Client) Protocol() common.Protocol {
	return common.ProtocolJSONRPC
}

// Close closes the client connection
func (c *Client) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return common.ErrClientClosed
	}

	c.httpClient.CloseIdleConnections()
	return nil
}

// SetHeader sets a header for all requests
func (c *Client) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[key] = value
}

// RemoveHeader removes a header
func (c *Client) RemoveHeader(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.headers, key)
}
