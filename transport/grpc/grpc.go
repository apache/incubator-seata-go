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
	"google.golang.org/protobuf/proto"

	"seata-go-ai-transport/common"
)

// Package initialization - register builders with the common registry
func init() {
	common.RegisterClientBuilder(common.ProtocolGRPC, &ClientBuilder{})
	common.RegisterServerBuilder(common.ProtocolGRPC, &ServerBuilder{})
}

// Public constructors for direct usage

// NewGRPCClient creates a new gRPC client directly
func NewGRPCClient(target string, isInsecure bool) (*Client, error) {
	config := NewGRPCClientConfig(target, isInsecure)
	return NewClient(config)
}

// NewGRPCServer creates a new gRPC server directly
func NewGRPCServer(address string) (*Server, error) {
	config := NewGRPCServerConfig(address)
	return NewServer(config)
}

// Utility functions for method registration

// CreateMethodInfo creates MethodInfo for a gRPC method
func CreateMethodInfo(fullName string, isStreaming bool, inputType, outputType proto.Message) MethodInfo {
	return MethodInfo{
		FullName:    fullName,
		IsStreaming: isStreaming,
		InputType:   inputType,
		OutputType:  outputType,
	}
}

// CreateUnaryMethodInfo creates MethodInfo for a unary method
func CreateUnaryMethodInfo(fullName string, inputType, outputType proto.Message) MethodInfo {
	return CreateMethodInfo(fullName, false, inputType, outputType)
}

// CreateStreamingMethodInfo creates MethodInfo for a streaming method
func CreateStreamingMethodInfo(fullName string, inputType, outputType proto.Message) MethodInfo {
	return CreateMethodInfo(fullName, true, inputType, outputType)
}

// Helper functions for error handling

// IsGRPCError checks if an error is a gRPC error
func IsGRPCError(err error) bool {
	// This can be extended to check for specific gRPC error types
	if err == nil {
		return false
	}
	// Check if it's a gRPC status error
	_, ok := err.(interface {
		GRPCStatus() interface{}
	})
	return ok
}

// CreateTransportError creates a transport error from gRPC error
func CreateTransportError(err error) *common.TransportError {
	return common.NewTransportError(
		common.ProtocolGRPC,
		-1, // gRPC errors don't have simple integer codes
		err.Error(),
		err,
	)
}
