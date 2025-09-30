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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	"seata-go-ai-transport/common"
)

// SSEServer extends the basic JSON-RPC server with SSE streaming support
type SSEServer struct {
	*Server // Embed regular server
}

// SSEServerConfig extends ServerConfig
type SSEServerConfig struct {
	*ServerConfig
	StreamPath string `json:"stream_path"`
}

var _ common.Server = (*SSEServer)(nil)

// NewSSEServer creates a new SSE-enabled JSON-RPC server
func NewSSEServer(config *SSEServerConfig) *SSEServer {
	server := NewServer(config.ServerConfig)

	streamPath := config.StreamPath
	if streamPath == "" {
		streamPath = "/stream"
	}

	sseServer := &SSEServer{
		Server: server,
	}

	// Add SSE streaming endpoint
	mux := server.server.Handler.(*http.ServeMux)
	mux.HandleFunc(streamPath, sseServer.handleSSEStream)

	return sseServer
}

// handleSSEStream handles SSE streaming requests
func (s *SSEServer) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeSSEError(w, ParseError, nil)
		return
	}

	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeSSEError(w, ParseError, nil)
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != Version {
		s.writeSSEError(w, InvalidRequest, req.ID)
		return
	}

	// Handle streaming request
	s.handleStreamRequest(w, r, &req)
}

// handleStreamRequest handles a single streaming JSON-RPC request
func (s *SSEServer) handleStreamRequest(w http.ResponseWriter, r *http.Request, req *JSONRPCRequest) {
	// Get streaming handler
	s.mu.RLock()
	handler, exists := s.streamHandlers[req.Method]
	s.mu.RUnlock()

	if !exists {
		s.writeSSEError(w, MethodNotFound, req.ID)
		return
	}

	// Create common message
	msg := &common.Message{
		Method:  req.Method,
		Payload: req.Params,
		Headers: make(map[string]string),
	}

	// Copy HTTP headers
	for k, values := range r.Header {
		if len(values) > 0 {
			msg.Headers[k] = values[0]
		}
	}

	// Create stream writer
	streamWriter := NewSSEStreamWriter(w, req.ID)

	// Call streaming handler
	if err := handler(r.Context(), msg, streamWriter); err != nil {
		s.writeSSEError(w, NewJSONRPCError(InternalErrorCode, err.Error(), nil), req.ID)
		return
	}

	// Send end event
	streamWriter.sendEndEvent()
}

// writeSSEError writes a JSON-RPC error as SSE event
func (s *SSEServer) writeSSEError(w http.ResponseWriter, jsonErr *JSONRPCError, id interface{}) {
	resp := NewErrorResponse(jsonErr, id)
	data, err := json.Marshal(resp)
	if err != nil {
		// If we can't marshal the error response, send a basic error event
		event := &SSEEvent{
			Event: "error",
			Data:  "Internal Server Error",
		}
		fmt.Fprint(w, event.String())
		return
	}

	event := &SSEEvent{
		Event: "error",
		Data:  string(data),
	}
	fmt.Fprint(w, event.String())

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// SSEStreamWriter implements common.StreamWriter for SSE
type SSEStreamWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	id      interface{}
	eventID int64
	closed  int32
	mu      sync.Mutex
}

var _ common.StreamWriter = (*SSEStreamWriter)(nil)

// NewSSEStreamWriter creates a new SSE stream writer
func NewSSEStreamWriter(w http.ResponseWriter, id interface{}) *SSEStreamWriter {
	flusher, _ := w.(http.Flusher)
	return &SSEStreamWriter{
		writer:  w,
		flusher: flusher,
		id:      id,
	}
}

// Send sends a message to the stream
func (w *SSEStreamWriter) Send(resp *common.StreamResponse) error {
	if atomic.LoadInt32(&w.closed) == 1 {
		return common.ErrStreamClosed
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Handle error response
	if resp.Error != nil {
		var jsonErr *JSONRPCError
		var jsonRpcErr *JSONRPCError
		if errors.As(resp.Error, &jsonRpcErr) {
			jsonErr = jsonRpcErr
		}

		errorResp := NewErrorResponse(jsonErr, w.id)
		data, err := json.Marshal(errorResp)
		if err != nil {
			return fmt.Errorf("failed to marshal error response: %w", err)
		}

		event := &SSEEvent{
			ID:    fmt.Sprintf("%d", atomic.AddInt64(&w.eventID, 1)),
			Event: "error",
			Data:  string(data),
		}

		if _, err := fmt.Fprint(w.writer, event.String()); err != nil {
			return fmt.Errorf("failed to write SSE event: %w", err)
		}

		if w.flusher != nil {
			w.flusher.Flush()
		}
		return nil
	}

	// Create JSON-RPC response
	var result interface{}
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	jsonResp, err := NewResponse(result, w.id)
	if err != nil {
		return fmt.Errorf("failed to create JSON-RPC response: %w", err)
	}

	data, err := json.Marshal(jsonResp)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON-RPC response: %w", err)
	}

	// Determine event type
	eventType := "data"
	if resp.Done {
		eventType = "end"
	}

	event := &SSEEvent{
		ID:    fmt.Sprintf("%d", atomic.AddInt64(&w.eventID, 1)),
		Event: eventType,
		Data:  string(data),
	}

	if _, err := fmt.Fprint(w.writer, event.String()); err != nil {
		return fmt.Errorf("failed to write SSE event: %w", err)
	}

	if w.flusher != nil {
		w.flusher.Flush()
	}

	return nil
}

// Close closes the stream
func (w *SSEStreamWriter) Close() error {
	if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
		return common.ErrStreamClosed
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Send close event
	event := &SSEEvent{
		ID:    fmt.Sprintf("%d", atomic.AddInt64(&w.eventID, 1)),
		Event: "close",
		Data:  "",
	}

	fmt.Fprint(w.writer, event.String())

	if w.flusher != nil {
		w.flusher.Flush()
	}

	return nil
}

// sendEndEvent sends an end event to mark the stream as complete
func (w *SSEStreamWriter) sendEndEvent() {
	if atomic.LoadInt32(&w.closed) == 1 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	event := &SSEEvent{
		ID:    fmt.Sprintf("%d", atomic.AddInt64(&w.eventID, 1)),
		Event: "end",
		Data:  "",
	}

	fmt.Fprint(w.writer, event.String())

	if w.flusher != nil {
		w.flusher.Flush()
	}
}
