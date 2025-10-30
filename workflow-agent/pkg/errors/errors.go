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

package errors

import (
	"fmt"
)

// ErrorCode represents an error code
type ErrorCode int

const (
	ErrUnknown ErrorCode = iota
	ErrConfig
	ErrConnection
	ErrRegistration
	ErrHeartbeat
	ErrDiscovery
	ErrInvalidArgument
	ErrNotFound
	ErrAlreadyExists
	ErrTimeout
	ErrUnauthorized
	ErrGeneration
	ErrOrchestration
	ErrMaxRetriesExceeded
)

var errorMessages = map[ErrorCode]string{
	ErrUnknown:            "unknown error",
	ErrConfig:             "configuration error",
	ErrConnection:         "connection error",
	ErrRegistration:       "registration error",
	ErrHeartbeat:          "heartbeat error",
	ErrDiscovery:          "discovery error",
	ErrInvalidArgument:    "invalid argument",
	ErrNotFound:           "not found",
	ErrAlreadyExists:      "already exists",
	ErrTimeout:            "timeout",
	ErrUnauthorized:       "unauthorized",
	ErrGeneration:         "generation error",
	ErrOrchestration:      "orchestration error",
	ErrMaxRetriesExceeded: "max retries exceeded",
}

// Error represents a custom error with code and context
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", errorMessages[e.Code], e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", errorMessages[e.Code], e.Message)
}

// Unwrap implements the unwrap interface for error chains
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is implements the Is interface for error comparison
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// New creates a new error with the given code and message
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Newf creates a new error with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Wrapf wraps an error with formatted message
func Wrapf(code ErrorCode, cause error, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// Predefined error constructors

func ConfigError(message string) *Error {
	return New(ErrConfig, message)
}

func ConfigErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrConfig, format, args...)
}

func ConnectionError(message string) *Error {
	return New(ErrConnection, message)
}

func ConnectionErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrConnection, format, args...)
}

func RegistrationError(message string) *Error {
	return New(ErrRegistration, message)
}

func RegistrationErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrRegistration, format, args...)
}

func HeartbeatError(message string) *Error {
	return New(ErrHeartbeat, message)
}

func HeartbeatErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrHeartbeat, format, args...)
}

func DiscoveryError(message string) *Error {
	return New(ErrDiscovery, message)
}

func DiscoveryErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrDiscovery, format, args...)
}

func InvalidArgumentError(message string) *Error {
	return New(ErrInvalidArgument, message)
}

func InvalidArgumentErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrInvalidArgument, format, args...)
}

func NotFoundError(message string) *Error {
	return New(ErrNotFound, message)
}

func NotFoundErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrNotFound, format, args...)
}

func AlreadyExistsError(message string) *Error {
	return New(ErrAlreadyExists, message)
}

func AlreadyExistsErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrAlreadyExists, format, args...)
}

func TimeoutError(message string) *Error {
	return New(ErrTimeout, message)
}

func TimeoutErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrTimeout, format, args...)
}

func UnauthorizedError(message string) *Error {
	return New(ErrUnauthorized, message)
}

func UnauthorizedErrorf(format string, args ...interface{}) *Error {
	return Newf(ErrUnauthorized, format, args...)
}

// GetCode extracts the error code from an error
func GetCode(err error) ErrorCode {
	if err == nil {
		return ErrUnknown
	}
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return ErrUnknown
}

// IsCode checks if an error has the given code
func IsCode(err error, code ErrorCode) bool {
	return GetCode(err) == code
}
