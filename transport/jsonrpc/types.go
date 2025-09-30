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
	"fmt"
)

// Version JSONRPC version constant
const Version = "2.0"

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface
func (e *JSONRPCError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("jsonrpc error [%d]: %s (data: %v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("jsonrpc error [%d]: %s", e.Code, e.Message)
}

// Standard JSON-RPC error codes
const (
	ParseErrorCode     = -32700
	InvalidRequestCode = -32600
	MethodNotFoundCode = -32601
	InvalidParamsCode  = -32602
	InternalErrorCode  = -32603
	ServerErrorCode    = -32000 // to -32099 range for server errors
)

// Standard JSON-RPC errors
var (
	ParseError     = &JSONRPCError{Code: ParseErrorCode, Message: "Parse error"}
	InvalidRequest = &JSONRPCError{Code: InvalidRequestCode, Message: "Invalid Request"}
	MethodNotFound = &JSONRPCError{Code: MethodNotFoundCode, Message: "Method not found"}
	InvalidParams  = &JSONRPCError{Code: InvalidParamsCode, Message: "Invalid params"}
	InternalError  = &JSONRPCError{Code: InternalErrorCode, Message: "Internal error"}
)

// NewJSONRPCError creates a new JSON-RPC error
func NewJSONRPCError(code int, message string, data interface{}) *JSONRPCError {
	return &JSONRPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewRequest creates a new JSON-RPC request
func NewRequest(method string, params interface{}, id interface{}) (*JSONRPCRequest, error) {
	var paramsBytes json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsBytes = data
	}

	return &JSONRPCRequest{
		JSONRPC: Version,
		Method:  method,
		Params:  paramsBytes,
		ID:      id,
	}, nil
}

// NewResponse creates a new JSON-RPC response with result
func NewResponse(result interface{}, id interface{}) (*JSONRPCResponse, error) {
	var resultBytes json.RawMessage
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		resultBytes = data
	}

	return &JSONRPCResponse{
		JSONRPC: Version,
		Result:  resultBytes,
		ID:      id,
	}, nil
}

// NewErrorResponse creates a new JSON-RPC error response
func NewErrorResponse(err *JSONRPCError, id interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: Version,
		Error:   err,
		ID:      id,
	}
}

// IsNotification checks if the request is a notification (no ID)
func (r *JSONRPCRequest) IsNotification() bool {
	return r.ID == nil
}

// UnmarshalParams unmarshals the request params into the given value
func (r *JSONRPCRequest) UnmarshalParams(v interface{}) error {
	if len(r.Params) == 0 {
		return nil
	}
	return json.Unmarshal(r.Params, v)
}

// UnmarshalResult unmarshals the response result into the given value
func (r *JSONRPCResponse) UnmarshalResult(v interface{}) error {
	if r.Error != nil {
		return r.Error
	}
	if len(r.Result) == 0 {
		return nil
	}
	return json.Unmarshal(r.Result, v)
}
