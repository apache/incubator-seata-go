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
	"errors"
	"seata-go-ai-transport/common"
)

// Package initialization - register builders with the common registry
func init() {
	// Register SSE-enabled builders as the default for JSON-RPC
	// These support both regular and streaming operations
	common.RegisterClientBuilder(common.ProtocolJSONRPC, &SSEClientBuilder{})
	common.RegisterServerBuilder(common.ProtocolJSONRPC, &SSEServerBuilder{})
}

// Public constructors for direct usage

// NewJSONRPCClient creates a new JSON-RPC client directly
func NewJSONRPCClient(endpoint string) *Client {
	return NewClient(&ClientConfig{
		Endpoint: endpoint,
		Headers:  make(map[string]string),
	})
}

// NewJSONRPCServer creates a new JSON-RPC server directly
func NewJSONRPCServer(address string) *Server {
	return NewServer(&ServerConfig{
		Address: address,
	})
}

// Utility functions for working with JSON-RPC

// IsJSONRPCError checks if an error is a JSON-RPC error
func IsJSONRPCError(err error) bool {
	var JSONRPCError *JSONRPCError
	ok := errors.As(err, &JSONRPCError)
	return ok
}

// GetJSONRPCError extracts JSON-RPC error from error
func GetJSONRPCError(err error) (*JSONRPCError, bool) {
	var jsonErr *JSONRPCError
	ok := errors.As(err, &jsonErr)
	return jsonErr, ok
}

// CreateTransportError creates a transport error from JSON-RPC error
func CreateTransportError(jsonErr *JSONRPCError) *common.TransportError {
	return common.NewTransportError(
		common.ProtocolJSONRPC,
		jsonErr.Code,
		jsonErr.Message,
		jsonErr,
	)
}
