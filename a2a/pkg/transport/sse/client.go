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

package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"seata-go-ai-a2a/pkg/types"
)

// Client provides Server-Sent Events client functionality for A2A protocol
type Client struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.RWMutex
	streams    map[string]*StreamConnection
}

// StreamConnection represents an active SSE stream connection
type StreamConnection struct {
	id      string
	resp    *http.Response
	scanner *bufio.Scanner
	events  chan types.StreamResponse
	errors  chan error
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	closed  bool
}

// ClientConfig configures the SSE client
type ClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// NewClient creates a new SSE client
func NewClient(config *ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 0 // No timeout for streaming connections
	}

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		streams: make(map[string]*StreamConnection),
	}
}

// StreamMessages subscribes to message stream for a task
func (c *Client) StreamMessages(ctx context.Context, taskID string) (*StreamConnection, error) {
	url := fmt.Sprintf("%s/stream/message?taskId=%s", c.baseURL, taskID)
	return c.createStream(ctx, url)
}

// StreamTask subscribes to task updates for a task
func (c *Client) StreamTask(ctx context.Context, taskID string) (*StreamConnection, error) {
	url := fmt.Sprintf("%s/stream/task?taskId=%s", c.baseURL, taskID)
	return c.createStream(ctx, url)
}

// createStream creates a new SSE stream connection
func (c *Client) createStream(ctx context.Context, url string) (*StreamConnection, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "A2A-SSE-Client/1.0")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSE endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("SSE endpoint returned status %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		resp.Body.Close()
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	// Create stream connection
	streamCtx, cancel := context.WithCancel(ctx)
	connectionID := fmt.Sprintf("stream_%d", time.Now().UnixNano())

	conn := &StreamConnection{
		id:      connectionID,
		resp:    resp,
		scanner: bufio.NewScanner(resp.Body),
		events:  make(chan types.StreamResponse, 10),
		errors:  make(chan error, 5),
		ctx:     streamCtx,
		cancel:  cancel,
	}

	// Register connection
	c.mu.Lock()
	c.streams[connectionID] = conn
	c.mu.Unlock()

	// Start reading events
	go conn.readEvents()

	return conn, nil
}

// readEvents reads and parses SSE events from the connection
func (sc *StreamConnection) readEvents() {
	defer func() {
		sc.Close()
	}()

	var event SSEEvent
	var dataLines []string

	for sc.scanner.Scan() {
		line := sc.scanner.Text()

		// Handle different SSE line types
		if line == "" {
			// Empty line indicates end of event
			if event.Event != "" || len(dataLines) > 0 {
				// Process complete event
				event.Data = strings.Join(dataLines, "\n")
				sc.processEvent(&event)

				// Reset for next event
				event = SSEEvent{}
				dataLines = nil
			}
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			event.Event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		} else if strings.HasPrefix(line, "id: ") {
			event.ID = strings.TrimPrefix(line, "id: ")
		} else if strings.HasPrefix(line, "retry: ") {
			// Handle retry directive (not implemented for now)
		} else if strings.HasPrefix(line, ":") {
			// Comment line, ignore
		}

		// Check if context is cancelled
		select {
		case <-sc.ctx.Done():
			return
		default:
		}
	}

	// Check for scanning errors
	if err := sc.scanner.Err(); err != nil {
		select {
		case sc.errors <- fmt.Errorf("error reading SSE stream: %w", err):
		case <-sc.ctx.Done():
		}
	}
}

// processEvent processes a complete SSE event
func (sc *StreamConnection) processEvent(event *SSEEvent) {
	// Skip ping events
	if event.Event == "ping" {
		return
	}

	// Handle error events
	if event.Event == "error" {
		var errorData map[string]interface{}
		if err := json.Unmarshal([]byte(event.Data), &errorData); err == nil {
			if errorInfo, ok := errorData["error"].(map[string]interface{}); ok {
				if message, ok := errorInfo["message"].(string); ok {
					select {
					case sc.errors <- fmt.Errorf("SSE error: %s", message):
					case <-sc.ctx.Done():
					}
					return
				}
			}
		}
		return
	}

	// Handle stream_closed events
	if event.Event == "stream_closed" {
		sc.Close()
		return
	}

	// Parse JSON-RPC response from event data
	var jsonrpcResp map[string]interface{}
	if err := json.Unmarshal([]byte(event.Data), &jsonrpcResp); err != nil {
		select {
		case sc.errors <- fmt.Errorf("failed to parse JSON-RPC response: %w", err):
		case <-sc.ctx.Done():
		}
		return
	}

	// Extract result from JSON-RPC response
	result, ok := jsonrpcResp["result"]
	if !ok {
		select {
		case sc.errors <- fmt.Errorf("JSON-RPC response missing result field"):
		case <-sc.ctx.Done():
		}
		return
	}

	// Convert to appropriate stream response type based on event type
	var streamResp types.StreamResponse

	switch event.Event {
	case "task":
		var task types.Task
		if err := unmarshalInterface(result, &task); err != nil {
			select {
			case sc.errors <- fmt.Errorf("failed to unmarshal task: %w", err):
			case <-sc.ctx.Done():
			}
			return
		}
		streamResp = &types.TaskStreamResponse{Task: &task}

	case "message":
		var message types.Message
		if err := unmarshalInterface(result, &message); err != nil {
			select {
			case sc.errors <- fmt.Errorf("failed to unmarshal message: %w", err):
			case <-sc.ctx.Done():
			}
			return
		}
		streamResp = &types.MessageStreamResponse{Message: &message}

	case "status_update":
		var statusUpdate types.TaskStatusUpdateEvent
		if err := unmarshalInterface(result, &statusUpdate); err != nil {
			select {
			case sc.errors <- fmt.Errorf("failed to unmarshal status update: %w", err):
			case <-sc.ctx.Done():
			}
			return
		}
		streamResp = &types.StatusUpdateStreamResponse{StatusUpdate: &statusUpdate}

	case "artifact_update":
		var artifactUpdate types.TaskArtifactUpdateEvent
		if err := unmarshalInterface(result, &artifactUpdate); err != nil {
			select {
			case sc.errors <- fmt.Errorf("failed to unmarshal artifact update: %w", err):
			case <-sc.ctx.Done():
			}
			return
		}
		streamResp = &types.ArtifactUpdateStreamResponse{ArtifactUpdate: &artifactUpdate}

	case "connected":
		// Handle connection confirmation - could create a custom response type
		return

	default:
		select {
		case sc.errors <- fmt.Errorf("unknown event type: %s", event.Event):
		case <-sc.ctx.Done():
		}
		return
	}

	// Send stream response
	select {
	case sc.events <- streamResp:
	case <-sc.ctx.Done():
	}
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
		sc.cancel()
		sc.resp.Body.Close()
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

// GetID returns the connection ID
func (sc *StreamConnection) GetID() string {
	return sc.id
}

// Close closes the SSE client and all active streams
func (c *Client) Close() error {
	c.mu.Lock()
	streams := make([]*StreamConnection, 0, len(c.streams))
	for _, stream := range c.streams {
		streams = append(streams, stream)
	}
	c.streams = make(map[string]*StreamConnection)
	c.mu.Unlock()

	// Close all streams
	for _, stream := range streams {
		stream.Close()
	}

	// Close HTTP client connections
	c.httpClient.CloseIdleConnections()
	return nil
}

// GetActiveStreams returns the number of active streams
func (c *Client) GetActiveStreams() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.streams)
}

// SSEEvent represents a parsed Server-Sent Event
type SSEEvent struct {
	Event string
	Data  string
	ID    string
}

// unmarshalInterface converts interface{} to a target struct
func unmarshalInterface(source interface{}, target interface{}) error {
	jsonData, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("failed to marshal source: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target: %w", err)
	}

	return nil
}

// SetTimeout sets the client timeout (note: this doesn't affect active streams)
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// GetBaseURL returns the client base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}
