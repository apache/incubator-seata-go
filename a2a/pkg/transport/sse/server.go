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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"seata-go-ai-a2a/pkg/task"
	"seata-go-ai-a2a/pkg/types"
)

// Server provides Server-Sent Events functionality for A2A protocol streaming
type Server struct {
	mu           sync.RWMutex
	connections  map[string]*Connection
	taskManager  *task.TaskManager
	pingInterval time.Duration
}

// Connection represents an active SSE connection
type Connection struct {
	id      string
	writer  http.ResponseWriter
	flusher http.Flusher
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	closed  bool
}

// NewServer creates a new SSE server
func NewServer(taskManager *task.TaskManager) *Server {
	return &Server{
		connections:  make(map[string]*Connection),
		taskManager:  taskManager,
		pingInterval: 30 * time.Second,
	}
}

// ServeHTTP handles SSE requests
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests for SSE
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if client accepts text/event-stream
	if r.Header.Get("Accept") != "text/event-stream" {
		http.Error(w, "Accept header must be text/event-stream", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Create flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create connection
	ctx, cancel := context.WithCancel(r.Context())
	conn := &Connection{
		id:      generateConnectionID(),
		writer:  w,
		flusher: flusher,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Register connection
	s.mu.Lock()
	s.connections[conn.id] = conn
	s.mu.Unlock()

	// Handle connection cleanup
	defer func() {
		s.mu.Lock()
		delete(s.connections, conn.id)
		s.mu.Unlock()
		conn.Close()
	}()

	// Start ping routine for keep-alive
	go s.pingConnection(conn)

	// Handle different streaming endpoints
	switch r.URL.Path {
	case "/stream/message":
		s.handleMessageStream(conn, r)
	case "/stream/task":
		s.handleTaskStream(conn, r)
	default:
		s.sendError(conn, "Unknown stream endpoint", http.StatusNotFound)
	}
}

// handleMessageStream handles message/stream requests via SSE
func (s *Server) handleMessageStream(conn *Connection, r *http.Request) {
	// Parse the JSON-RPC request from query parameters or request body
	// For SSE, we typically get the initial request via query params or headers

	taskID := r.URL.Query().Get("taskId")
	if taskID == "" {
		s.sendError(conn, "taskId parameter required", http.StatusBadRequest)
		return
	}

	// Subscribe to task updates
	subscription, err := s.taskManager.SubscribeToTask(r.Context(), taskID)
	if err != nil {
		s.sendError(conn, fmt.Sprintf("Failed to subscribe to task: %v", err), http.StatusBadRequest)
		return
	}
	defer subscription.Close()

	// Send initial connection confirmation
	s.sendEvent(conn, "connected", map[string]interface{}{
		"taskId":       taskID,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"connectionId": conn.id,
	})

	// Stream task updates
	for {
		select {
		case response, ok := <-subscription.Events():
			if !ok {
				// Stream closed
				s.sendEvent(conn, "stream_closed", map[string]interface{}{
					"reason":    "task_completed",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})
				return
			}

			// Send the stream response as SSE
			s.sendStreamResponse(conn, response)

		case <-conn.ctx.Done():
			// Connection closed
			return
		}
	}
}

// handleTaskStream handles task subscription streams via SSE
func (s *Server) handleTaskStream(conn *Connection, r *http.Request) {
	taskID := r.URL.Query().Get("taskId")
	if taskID == "" {
		s.sendError(conn, "taskId parameter required", http.StatusBadRequest)
		return
	}

	// Subscribe to task updates
	subscription, err := s.taskManager.SubscribeToTask(r.Context(), taskID)
	if err != nil {
		s.sendError(conn, fmt.Sprintf("Failed to subscribe to task: %v", err), http.StatusBadRequest)
		return
	}
	defer subscription.Close()

	// Send initial connection confirmation
	s.sendEvent(conn, "connected", map[string]interface{}{
		"taskId":       taskID,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"connectionId": conn.id,
	})

	// Stream task updates
	for {
		select {
		case response, ok := <-subscription.Events():
			if !ok {
				s.sendEvent(conn, "stream_closed", map[string]interface{}{
					"reason":    "subscription_ended",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})
				return
			}

			// Send the stream response as SSE
			s.sendStreamResponse(conn, response)

		case <-conn.ctx.Done():
			return
		}
	}
}

// sendStreamResponse sends a stream response as an SSE event
func (s *Server) sendStreamResponse(conn *Connection, response types.StreamResponse) {
	var eventType string
	var data interface{}

	switch r := response.(type) {
	case *types.TaskStreamResponse:
		eventType = "task"
		data = r.Task
	case *types.MessageStreamResponse:
		eventType = "message"
		data = r.Message
	case *types.StatusUpdateStreamResponse:
		eventType = "status_update"
		data = r.StatusUpdate
	case *types.ArtifactUpdateStreamResponse:
		eventType = "artifact_update"
		data = r.ArtifactUpdate
	default:
		eventType = "unknown"
		data = response
	}

	// Wrap in JSON-RPC 2.0 format for A2A compatibility
	jsonrpcResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  data,
		"id":      nil, // SSE doesn't have request IDs
	}

	s.sendEvent(conn, eventType, jsonrpcResponse)
}

// sendEvent sends an SSE event
func (s *Server) sendEvent(conn *Connection, eventType string, data interface{}) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.closed {
		return fmt.Errorf("connection closed")
	}

	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write SSE format
	if eventType != "" {
		fmt.Fprintf(conn.writer, "event: %s\n", eventType)
	}
	fmt.Fprintf(conn.writer, "data: %s\n\n", string(jsonData))

	conn.flusher.Flush()
	return nil
}

// sendError sends an error event
func (s *Server) sendError(conn *Connection, message string, statusCode int) {
	errorData := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    -32603, // Internal error
			"message": message,
		},
		"jsonrpc": "2.0",
		"id":      nil,
	}

	s.sendEvent(conn, "error", errorData)
}

// pingConnection sends periodic ping events to keep connection alive
func (s *Server) pingConnection(conn *Connection) {
	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := s.sendEvent(conn, "ping", map[string]interface{}{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			if err != nil {
				// Connection is likely closed
				return
			}
		case <-conn.ctx.Done():
			return
		}
	}
}

// Close closes the connection
func (c *Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		c.cancel()
	}
}

// IsClosed returns true if the connection is closed
func (c *Connection) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// GetActiveConnections returns the number of active connections
func (s *Server) GetActiveConnections() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

// CloseAllConnections closes all active connections
func (s *Server) CloseAllConnections() {
	s.mu.Lock()
	connections := make([]*Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		connections = append(connections, conn)
	}
	s.connections = make(map[string]*Connection)
	s.mu.Unlock()

	for _, conn := range connections {
		conn.Close()
	}
}

// SetPingInterval sets the ping interval for keep-alive
func (s *Server) SetPingInterval(interval time.Duration) {
	s.pingInterval = interval
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}
