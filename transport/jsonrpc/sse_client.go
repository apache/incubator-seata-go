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
	"net/http"
	"sync"
	"sync/atomic"

	"seata-go-ai-transport/common"
)

// SSEClient implements streaming JSON-RPC over Server-Sent Events
type SSEClient struct {
	*Client
	streamEndpoint string
}

// SSEClientConfig extends ClientConfig with streaming endpoint
type SSEClientConfig struct {
	*ClientConfig
	StreamEndpoint string `json:"stream_endpoint"`
}

var _ common.Client = (*SSEClient)(nil)

// NewSSEClient creates a new SSE-enabled JSON-RPC client
func NewSSEClient(config *SSEClientConfig) *SSEClient {
	client := NewClient(config.ClientConfig)

	streamEndpoint := config.StreamEndpoint
	if streamEndpoint == "" {
		streamEndpoint = config.Endpoint + "/stream"
	}

	return &SSEClient{
		Client:         client,
		streamEndpoint: streamEndpoint,
	}
}

// Stream makes a streaming RPC call using SSE
func (c *SSEClient) Stream(ctx context.Context, msg *common.Message) (common.StreamReader, error) {
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

	// Create HTTP request for streaming
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.streamEndpoint, bytes.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers for SSE
	c.mu.RLock()
	for k, v := range c.headers {
		httpReq.Header.Set(k, v)
	}
	// Add message headers
	for k, v := range msg.Headers {
		httpReq.Header.Set(k, v)
	}
	c.mu.RUnlock()

	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")

	// Send request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Check HTTP status
	if httpResp.StatusCode != http.StatusOK {
		_ = httpResp.Body.Close()
		return nil, fmt.Errorf("HTTP error: %d", httpResp.StatusCode)
	}

	// Create stream reader
	return NewSSEStreamReader(ctx, httpResp), nil
}

// SSEStreamReader implements common.StreamReader for SSE
type SSEStreamReader struct {
	ctx    context.Context
	resp   *http.Response
	parser *SSEParser
	closed int32
	mu     sync.Mutex
}

var _ common.StreamReader = (*SSEStreamReader)(nil)

// NewSSEStreamReader creates a new SSE stream reader
func NewSSEStreamReader(ctx context.Context, resp *http.Response) *SSEStreamReader {
	return &SSEStreamReader{
		ctx:    ctx,
		resp:   resp,
		parser: NewSSEParser(ctx, resp.Body),
	}
}

// Recv receives the next message from the stream
func (r *SSEStreamReader) Recv() (*common.StreamResponse, error) {
	if atomic.LoadInt32(&r.closed) == 1 {
		return nil, common.ErrStreamClosed
	}

	event, err := r.parser.NextEvent()
	if err != nil {
		return &common.StreamResponse{
			Done:  true,
			Error: err,
		}, nil
	}

	// Parse the SSE data as JSON-RPC response
	var jsonResp JSONRPCResponse
	if err := json.Unmarshal([]byte(event.Data), &jsonResp); err != nil {
		return &common.StreamResponse{
			Error: fmt.Errorf("failed to unmarshal SSE data: %w", err),
		}, nil
	}

	// Create stream response
	streamResp := &common.StreamResponse{
		Headers: make(map[string]string),
	}

	// Copy headers from HTTP response
	for k, values := range r.resp.Header {
		if len(values) > 0 {
			streamResp.Headers[k] = values[0]
		}
	}

	// Add SSE-specific headers
	if event.ID != "" {
		streamResp.Headers["SSE-Event-ID"] = event.ID
	}
	if event.Event != "" {
		streamResp.Headers["SSE-Event-Type"] = event.Event
	}

	// Handle JSON-RPC error
	if jsonResp.Error != nil {
		streamResp.Error = jsonResp.Error
		return streamResp, nil
	}

	// Set response data
	streamResp.Data = jsonResp.Result

	// Check for end of stream marker
	if event.Event == "end" || event.Event == "close" {
		streamResp.Done = true
	}

	return streamResp, nil
}

// Close closes the stream
func (r *SSEStreamReader) Close() error {
	if !atomic.CompareAndSwapInt32(&r.closed, 0, 1) {
		return common.ErrStreamClosed
	}

	return r.resp.Body.Close()
}
