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

package transport

import (
	"seata-go-ai-transport/common"
	_ "seata-go-ai-transport/grpc"
	_ "seata-go-ai-transport/jsonrpc"
)

// Re-export common types for convenience
type (
	Protocol       = common.Protocol
	StreamType     = common.StreamType
	Message        = common.Message
	Response       = common.Response
	StreamResponse = common.StreamResponse
	Client         = common.Client
	Server         = common.Server
	Handler        = common.Handler
	StreamHandler  = common.StreamHandler
	StreamReader   = common.StreamReader
	StreamWriter   = common.StreamWriter
	Config         = common.Config
	ClientBuilder  = common.ClientBuilder
	ServerBuilder  = common.ServerBuilder
)

// Re-export protocol constants
const (
	ProtocolGRPC    = common.ProtocolGRPC
	ProtocolJSONRPC = common.ProtocolJSONRPC
)

// Re-export stream type constants
const (
	StreamTypeUnary        = common.StreamTypeUnary
	StreamTypeServerStream = common.StreamTypeServerStream
	StreamTypeClientStream = common.StreamTypeClientStream
	StreamTypeBiDirStream  = common.StreamTypeBiDirStream
)

// Re-export registry functions
var (
	RegisterClientBuilder = common.RegisterClientBuilder
	RegisterServerBuilder = common.RegisterServerBuilder
	CreateClient          = common.CreateClient
	CreateServer          = common.CreateServer
	GetSupportedProtocols = common.GetSupportedProtocols
)

// Re-export utility functions
var (
	MarshalMessage            = common.MarshalMessage
	UnmarshalResponse         = common.UnmarshalResponse
	UnmarshalStreamResponse   = common.UnmarshalStreamResponse
	CreateResponse            = common.CreateResponse
	CreateStreamResponse      = common.CreateStreamResponse
	CreateErrorResponse       = common.CreateErrorResponse
	CreateErrorStreamResponse = common.CreateErrorStreamResponse
)

// Re-export common errors
var (
	ErrProtocolNotSupported = common.ErrProtocolNotSupported
	ErrClientClosed         = common.ErrClientClosed
	ErrServerClosed         = common.ErrServerClosed
	ErrStreamClosed         = common.ErrStreamClosed
	ErrInvalidConfig        = common.ErrInvalidConfig
	ErrHandlerNotFound      = common.ErrHandlerNotFound
	ErrTimeout              = common.ErrTimeout
	ErrConnectionFailed     = common.ErrConnectionFailed
)

// Re-export error functions
var (
	NewTransportError = common.NewTransportError
	IsTransportError  = common.IsTransportError
	GetTransportError = common.GetTransportError
)
