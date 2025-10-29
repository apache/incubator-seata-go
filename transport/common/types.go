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

package common

import (
	"context"
	"io"
)

// Protocol represents the transport protocol type
type Protocol int

const (
	ProtocolGRPC Protocol = iota
	ProtocolJSONRPC
)

func (p Protocol) String() string {
	switch p {
	case ProtocolGRPC:
		return "grpc"
	case ProtocolJSONRPC:
		return "jsonrpc"
	default:
		return "unknown"
	}
}

// StreamType represents the streaming capability
type StreamType int

const (
	StreamTypeUnary StreamType = iota
	StreamTypeServerStream
	StreamTypeClientStream
	StreamTypeBiDirStream
)

// Message represents a generic transport message
type Message struct {
	Method  string
	Payload []byte
	Headers map[string]string
}

// Response represents a generic transport response
type Response struct {
	Data    []byte
	Headers map[string]string
	Error   error
}

// StreamResponse represents a streaming response
type StreamResponse struct {
	Data    []byte
	Headers map[string]string
	Done    bool
	Error   error
}

// Client represents a transport client interface
type Client interface {
	// Call makes a unary RPC call
	Call(ctx context.Context, msg *Message) (*Response, error)

	// Stream makes a streaming RPC call
	Stream(ctx context.Context, msg *Message) (StreamReader, error)

	// Protocol returns the underlying protocol
	Protocol() Protocol

	// Close closes the client connection
	Close() error
}

// Server represents a transport server interface
type Server interface {
	// Serve starts the server
	Serve() error

	// Stop stops the server gracefully
	Stop(ctx context.Context) error

	// RegisterHandler registers a handler for a method
	RegisterHandler(method string, handler Handler)

	// RegisterStreamHandler registers a streaming handler
	RegisterStreamHandler(method string, handler StreamHandler)

	// Protocol returns the underlying protocol
	Protocol() Protocol
}

// Handler represents a unary RPC handler
type Handler func(ctx context.Context, req *Message) (*Response, error)

// StreamHandler represents a streaming RPC handler
type StreamHandler func(ctx context.Context, req *Message, stream StreamWriter) error

// StreamReader provides methods to read from a stream
type StreamReader interface {
	// Recv receives the next message from the stream
	Recv() (*StreamResponse, error)

	// Close closes the stream
	Close() error
}

// StreamWriter provides methods to write to a stream
type StreamWriter interface {
	// Send sends a message to the stream
	Send(resp *StreamResponse) error

	// Close closes the stream
	Close() error
}

// Config represents transport configuration
type Config struct {
	Protocol Protocol               `json:"protocol"`
	Address  string                 `json:"address"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// ClientBuilder builds transport clients
type ClientBuilder interface {
	// Build creates a new client with the given config
	Build(config *Config) (Client, error)

	// Protocol returns the protocol this builder supports
	Protocol() Protocol
}

// ServerBuilder builds transport servers
type ServerBuilder interface {
	// Build creates a new server with the given config
	Build(config *Config) (Server, error)

	// Protocol returns the protocol this builder supports
	Protocol() Protocol
}

var _ io.Closer = (StreamReader)(nil)
var _ io.Closer = (StreamWriter)(nil)
