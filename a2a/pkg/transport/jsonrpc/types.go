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

	"seata-go-ai-a2a/pkg/types"
)

// JSONRPCVersion represents the JSON-RPC version
const JSONRPCVersion = "2.0"

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Error represents a JSON-RPC 2.0 error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC error codes
const (
	// JSON-RPC 2.0 standard errors
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603

	// A2A specific error codes (as defined in A2A specification)
	TaskNotFoundError                           = -32001
	TaskNotCancelableError                      = -32002
	UnsupportedOperationError                   = -32004
	ContentTypeNotSupportedError                = -32005
	InvalidAgentResponseError                   = -32006
	AuthenticatedExtendedCardNotConfiguredError = -32007
)

// MethodMapping maps A2A methods to JSON-RPC method names
var MethodMapping = map[string]string{
	"message/send":                        "message/send",
	"message/stream":                      "message/stream",
	"tasks/get":                           "tasks/get",
	"tasks/list":                          "tasks/list",
	"tasks/cancel":                        "tasks/cancel",
	"tasks/resubscribe":                   "tasks/resubscribe",
	"tasks/pushNotificationConfig/set":    "tasks/pushNotificationConfig/set",
	"tasks/pushNotificationConfig/get":    "tasks/pushNotificationConfig/get",
	"tasks/pushNotificationConfig/list":   "tasks/pushNotificationConfig/list",
	"tasks/pushNotificationConfig/delete": "tasks/pushNotificationConfig/delete",
	"agentCard/get":                       "agentCard/get",
}

// NewRequest creates a new JSON-RPC request
func NewRequest(method string, params interface{}, id interface{}) *Request {
	return &Request{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  params,
		ID:      id,
	}
}

// NewSuccessResponse creates a successful JSON-RPC response
func NewSuccessResponse(result interface{}, id interface{}) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		Result:  result,
		ID:      id,
	}
}

// NewErrorResponse creates an error JSON-RPC response
func NewErrorResponse(err *Error, id interface{}) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		Error:   err,
		ID:      id,
	}
}

// NewError creates a new JSON-RPC error
func NewError(code int, message string, data interface{}) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// ConvertA2AErrorToJSONRPCError converts an A2A error to JSON-RPC error format
func ConvertA2AErrorToJSONRPCError(a2aErr *types.A2AError) *Error {
	return &Error{
		Code:    a2aErr.Code,
		Message: a2aErr.Message,
		Data:    a2aErr.Data,
	}
}

// ParseRequest parses a JSON-RPC request from JSON bytes
func ParseRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, NewError(ParseError, "Parse error", err.Error())
	}

	if err := ValidateRequest(&req); err != nil {
		return nil, err
	}

	return &req, nil
}

// ValidateRequest validates a JSON-RPC request
func ValidateRequest(req *Request) *Error {
	if req.JSONRPC != JSONRPCVersion {
		return NewError(InvalidRequest, "Invalid JSON-RPC version", fmt.Sprintf("expected %s, got %s", JSONRPCVersion, req.JSONRPC))
	}

	if req.Method == "" {
		return NewError(InvalidRequest, "Missing method", nil)
	}

	if req.ID == nil {
		return NewError(InvalidRequest, "Missing id", nil)
	}

	return nil
}

// ValidateResponse validates a JSON-RPC response
func ValidateResponse(resp *Response) error {
	if resp.JSONRPC != JSONRPCVersion {
		return fmt.Errorf("invalid JSON-RPC version: expected %s, got %s", JSONRPCVersion, resp.JSONRPC)
	}

	if resp.Result == nil && resp.Error == nil {
		return fmt.Errorf("response must have either result or error")
	}

	if resp.Result != nil && resp.Error != nil {
		return fmt.Errorf("response cannot have both result and error")
	}

	return nil
}

// IsNotification returns true if the request is a notification (no ID)
func (r *Request) IsNotification() bool {
	return r.ID == nil
}

// GetMethodName returns the method name from the request
func (r *Request) GetMethodName() string {
	return r.Method
}

// GetParams returns the parameters from the request
func (r *Request) GetParams() interface{} {
	return r.Params
}

// GetID returns the request ID
func (r *Request) GetID() interface{} {
	return r.ID
}

// IsError returns true if the response contains an error
func (r *Response) IsError() bool {
	return r.Error != nil
}

// GetResult returns the result from the response
func (r *Response) GetResult() interface{} {
	return r.Result
}

// GetError returns the error from the response
func (r *Response) GetError() *Error {
	return r.Error
}

// GetID returns the response ID
func (r *Response) GetID() interface{} {
	return r.ID
}
