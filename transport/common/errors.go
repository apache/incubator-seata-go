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
	"errors"
	"fmt"
)


var (
	// ErrProtocolNotSupported indicates the protocol is not supported
	ErrProtocolNotSupported = errors.New("protocol not supported")

	// ErrClientClosed indicates the client is already closed
	ErrClientClosed = errors.New("client is closed")

	// ErrServerClosed indicates the server is already closed
	ErrServerClosed = errors.New("server is closed")

	// ErrStreamClosed indicates the stream is already closed
	ErrStreamClosed = errors.New("stream is closed")

	// ErrInvalidConfig indicates invalid configuration
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrHandlerNotFound indicates handler not found for method
	ErrHandlerNotFound = errors.New("handler not found")

	// ErrTimeout indicates operation timeout
	ErrTimeout = errors.New("operation timeout")

	// ErrConnectionFailed indicates connection failed
	ErrConnectionFailed = errors.New("connection failed")
)

// TransportError represents a transport-specific error
type TransportError struct {
	Protocol Protocol
	Code     int
	Message  string
	Cause    error
}

// Error implements the error interface
func (e *TransportError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s transport error [%d]: %s: %v", e.Protocol, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s transport error [%d]: %s", e.Protocol, e.Code, e.Message)
}

// Unwrap returns the cause error
func (e *TransportError) Unwrap() error {
	return e.Cause
}

// NewTransportError creates a new transport error
func NewTransportError(protocol Protocol, code int, message string, cause error) *TransportError {
	return &TransportError{
		Protocol: protocol,
		Code:     code,
		Message:  message,
		Cause:    cause,
	}
}

// IsTransportError checks if an error is a TransportError
func IsTransportError(err error) bool {
	var te *TransportError
	return errors.As(err, &te)
}

// GetTransportError extracts TransportError from error
func GetTransportError(err error) (*TransportError, bool) {
	var te *TransportError
	ok := errors.As(err, &te)
	return te, ok
}
