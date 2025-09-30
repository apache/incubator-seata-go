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
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"seata-go-ai-transport/common"
)

// Server implements the common.Server interface for gRPC
type Server struct {
	server         *grpc.Server
	address        string
	listener       net.Listener
	handlers       map[string]common.Handler
	streamHandlers map[string]common.StreamHandler
	methods        map[string]MethodInfo
	mu             sync.RWMutex
	started        int32
	closed         int32
}

// ServerConfig represents gRPC server configuration
type ServerConfig struct {
	Address string                `json:"address"`
	Options []grpc.ServerOption   `json:"-"`
	Methods map[string]MethodInfo `json:"-"`
}

var _ common.Server = (*Server)(nil)

// NewServer creates a new gRPC server
func NewServer(config *ServerConfig) (*Server, error) {
	opts := config.Options
	if opts == nil {
		opts = []grpc.ServerOption{}
	}

	server := &Server{
		server:         grpc.NewServer(opts...),
		address:        config.Address,
		handlers:       make(map[string]common.Handler),
		streamHandlers: make(map[string]common.StreamHandler),
		methods:        make(map[string]MethodInfo),
	}

	if config.Methods != nil {
		for method, info := range config.Methods {
			server.methods[method] = info
		}
	}

	return server, nil
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

	s.listener = listener
	return s.server.Serve(listener)
}

// Stop stops the server gracefully
func (s *Server) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return common.ErrServerClosed
	}

	// Create a channel to signal when graceful stop is complete
	done := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	// Wait for graceful stop or context cancellation
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// Force stop if context is cancelled
		s.server.Stop()
		return ctx.Err()
	}
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
	return common.ProtocolGRPC
}

// RegisterMethod registers a method with the server
func (s *Server) RegisterMethod(method string, info MethodInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.methods[method] = info
}

// CreateUnaryHandler creates a gRPC unary handler from a common handler
func (s *Server) CreateUnaryHandler(method string, handler common.Handler) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		// Get method info
		s.mu.RLock()
		methodInfo, exists := s.methods[method]
		s.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("method %s not registered", method)
		}

		// Convert proto message to JSON
		reqProto, ok := req.(proto.Message)
		if !ok {
			return nil, fmt.Errorf("request is not a proto message")
		}

		reqData, err := json.Marshal(reqProto)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		// Create common message
		msg := &common.Message{
			Method:  method,
			Payload: reqData,
			Headers: extractMetadata(ctx),
		}

		// Call handler
		resp, err := handler(ctx, msg)
		if err != nil {
			return nil, err
		}

		if resp.Error != nil {
			return nil, resp.Error
		}

		// Create output message
		output := proto.Clone(methodInfo.OutputType)
		if len(resp.Data) > 0 {
			if err := json.Unmarshal(resp.Data, output); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return output, nil
	}
}

// CreateStreamHandler creates a gRPC stream handler from a common stream handler
func (s *Server) CreateStreamHandler(method string, handler common.StreamHandler) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream) error {
		// Get method info
		s.mu.RLock()
		methodInfo, exists := s.methods[method]
		s.mu.RUnlock()

		if !exists {
			return fmt.Errorf("method %s not registered", method)
		}

		// Receive request
		input := proto.Clone(methodInfo.InputType)
		if err := stream.RecvMsg(input); err != nil {
			return fmt.Errorf("failed to receive message: %w", err)
		}

		// Convert to JSON
		reqData, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		// Create common message
		msg := &common.Message{
			Method:  method,
			Payload: reqData,
			Headers: extractMetadata(stream.Context()),
		}

		// Create stream writer
		streamWriter := NewGRPCStreamWriter(stream, methodInfo.OutputType)

		// Call handler
		return handler(stream.Context(), msg, streamWriter)
	}
}

// GetServer returns the underlying gRPC server
func (s *Server) GetServer() *grpc.Server {
	return s.server
}

// extractMetadata extracts gRPC metadata as headers
func extractMetadata(ctx context.Context) map[string]string {
	headers := make(map[string]string)

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for k, values := range md {
			if len(values) > 0 {
				headers[k] = values[0]
			}
		}
	}

	return headers
}
