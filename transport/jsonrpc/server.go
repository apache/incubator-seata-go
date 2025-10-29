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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"seata-go-ai-transport/common"
)

// Server implements the common.Server interface for JSON-RPC
type Server struct {
	address        string
	server         *http.Server
	handlers       map[string]common.Handler
	streamHandlers map[string]common.StreamHandler
	mu             sync.RWMutex
	started        int32
	closed         int32
}

// ServerConfig represents JSON-RPC server configuration
type ServerConfig struct {
	Address      string        `json:"address"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

var _ common.Server = (*Server)(nil)

// NewServer creates a new JSON-RPC server
func NewServer(config *ServerConfig) *Server {
	mux := http.NewServeMux()
	server := &Server{
		address:        config.Address,
		handlers:       make(map[string]common.Handler),
		streamHandlers: make(map[string]common.StreamHandler),
		server: &http.Server{
			Addr:         config.Address,
			Handler:      mux,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			IdleTimeout:  config.IdleTimeout,
		},
	}

	// Register JSON-RPC endpoint
	mux.HandleFunc("/", server.handleJSONRPC)

	return server
}

// Serve starts the server
func (s *Server) Serve() error {
	if !atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		return fmt.Errorf("server already started")
	}

	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		atomic.StoreInt32(&s.started, 0)
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}

	return s.server.Serve(listener)
}

// Stop stops the server gracefully
func (s *Server) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return common.ErrServerClosed
	}

	return s.server.Shutdown(ctx)
}

// RegisterHandler registers a handler for a method
func (s *Server) RegisterHandler(method string, handler common.Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

// RegisterStreamHandler registers a streaming handler
func (s *Server) RegisterStreamHandler(method string, handler common.StreamHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streamHandlers[method] = handler
}

// Protocol returns the underlying protocol
func (s *Server) Protocol() common.Protocol {
	return common.ProtocolJSONRPC
}

// handleJSONRPC handles JSON-RPC requests
func (s *Server) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, ParseError, nil)
		return
	}

	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, ParseError, nil)
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != Version {
		s.writeErrorResponse(w, InvalidRequest, req.ID)
		return
	}

	// Handle request
	s.handleRequest(w, r, &req)
}

// handleRequest handles a single JSON-RPC request
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request, req *JSONRPCRequest) {
	// Get handler
	s.mu.RLock()
	handler, exists := s.handlers[req.Method]
	s.mu.RUnlock()

	if !exists {
		if !req.IsNotification() {
			s.writeErrorResponse(w, MethodNotFound, req.ID)
		}
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

	// Call handler
	resp, err := handler(r.Context(), msg)
	if err != nil {
		if !req.IsNotification() {
			// Convert error to JSON-RPC error
			var jsonErr *JSONRPCError
			if transportErr, ok := common.GetTransportError(err); ok {
				jsonErr = NewJSONRPCError(transportErr.Code, transportErr.Message, transportErr.Cause)
			} else {
				jsonErr = NewJSONRPCError(InternalErrorCode, err.Error(), nil)
			}
			s.writeErrorResponse(w, jsonErr, req.ID)
		}
		return
	}

	// For notifications, don't send response
	if req.IsNotification() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Create response
	var result interface{}
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			s.writeErrorResponse(w, NewJSONRPCError(InternalErrorCode, "Failed to unmarshal result", err.Error()), req.ID)
			return
		}
	}

	jsonResp, err := NewResponse(result, req.ID)
	if err != nil {
		s.writeErrorResponse(w, NewJSONRPCError(InternalErrorCode, "Failed to create response", err.Error()), req.ID)
		return
	}

	// Set response headers
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}

	// Write response
	respData, err := json.Marshal(jsonResp)
	if err != nil {
		s.writeErrorResponse(w, NewJSONRPCError(InternalErrorCode, "Failed to marshal response", err.Error()), req.ID)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}

// writeErrorResponse writes a JSON-RPC error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, jsonErr *JSONRPCError, id interface{}) {
	resp := NewErrorResponse(jsonErr, id)
	data, err := json.Marshal(resp)
	if err != nil {
		// If we can't marshal the error response, send a basic HTTP error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK) // JSON-RPC errors are still HTTP 200
	w.Write(data)
}
