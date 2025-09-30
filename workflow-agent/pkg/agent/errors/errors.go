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
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// AgentError represents a structured error with context
type AgentError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Retryable bool                   `json:"retryable"`
	Component string                 `json:"component"`
	SessionID string                 `json:"session_id,omitempty"`
	Stack     []StackFrame           `json:"stack,omitempty"`
	Cause     error                  `json:"-"`
}

// StackFrame represents a stack frame
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// Error implements the error interface
func (e *AgentError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Component, e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AgentError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *AgentError) Is(target error) bool {
	if t, ok := target.(*AgentError); ok {
		return e.Code == t.Code && e.Component == t.Component
	}
	return false
}

// JSON returns the error as JSON
func (e *AgentError) JSON() []byte {
	data, _ := json.Marshal(e)
	return data
}

// Common error codes
const (
	// Core agent errors
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeExecutionTimeout = "EXECUTION_TIMEOUT"
	ErrCodeMaxStepsExceeded = "MAX_STEPS_EXCEEDED"
	ErrCodeConcurrencyLimit = "CONCURRENCY_LIMIT"
	ErrCodeSessionNotFound  = "SESSION_NOT_FOUND"
	ErrCodeAgentNotFound    = "AGENT_NOT_FOUND"
	ErrCodeInvalidState     = "INVALID_STATE"

	// Reasoning errors
	ErrCodeReasoningFailed  = "REASONING_FAILED"
	ErrCodeModelUnavailable = "MODEL_UNAVAILABLE"
	ErrCodeInvalidResponse  = "INVALID_RESPONSE"
	ErrCodeParsingFailed    = "PARSING_FAILED"
	ErrCodePromptTooLong    = "PROMPT_TOO_LONG"

	// Tool errors
	ErrCodeToolNotFound           = "TOOL_NOT_FOUND"
	ErrCodeToolExecutionFailed    = "TOOL_EXECUTION_FAILED"
	ErrCodeInvalidParameters      = "INVALID_PARAMETERS"
	ErrCodeToolTimeout            = "TOOL_TIMEOUT"
	ErrCodeToolRegistrationFailed = "TOOL_REGISTRATION_FAILED"

	// State errors
	ErrCodeStateLoadFailed   = "STATE_LOAD_FAILED"
	ErrCodeStateSaveFailed   = "STATE_SAVE_FAILED"
	ErrCodeStateDeleteFailed = "STATE_DELETE_FAILED"
	ErrCodeStateCorrupted    = "STATE_CORRUPTED"

	// Configuration errors
	ErrCodeInvalidConfig          = "INVALID_CONFIG"
	ErrCodeConfigLoadFailed       = "CONFIG_LOAD_FAILED"
	ErrCodeConfigValidationFailed = "CONFIG_VALIDATION_FAILED"

	// System errors
	ErrCodeSystemError        = "SYSTEM_ERROR"
	ErrCodeResourceExhausted  = "RESOURCE_EXHAUSTED"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// Component names
const (
	ComponentAgent     = "agent"
	ComponentReasoning = "reasoning"
	ComponentTools     = "tools"
	ComponentState     = "state"
	ComponentParser    = "parser"
	ComponentConfig    = "config"
	ComponentSession   = "session"
)

// NewAgentError creates a new agent error
func NewAgentError(component, code, message string) *AgentError {
	return &AgentError{
		Code:      code,
		Message:   message,
		Component: component,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
		Stack:     captureStack(2),
	}
}

// Wrap wraps an existing error with agent error context
func Wrap(err error, component, code, message string) *AgentError {
	if err == nil {
		return nil
	}

	agentErr := &AgentError{
		Code:      code,
		Message:   message,
		Component: component,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
		Stack:     captureStack(2),
		Cause:     err,
	}

	// If the cause is already an AgentError, inherit some properties
	if ae, ok := err.(*AgentError); ok {
		if agentErr.SessionID == "" {
			agentErr.SessionID = ae.SessionID
		}
	}

	return agentErr
}

// WithDetails adds details to the error
func (e *AgentError) WithDetails(details map[string]interface{}) *AgentError {
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithDetail adds a single detail to the error
func (e *AgentError) WithDetail(key string, value interface{}) *AgentError {
	e.Details[key] = value
	return e
}

// WithSessionID sets the session ID
func (e *AgentError) WithSessionID(sessionID string) *AgentError {
	e.SessionID = sessionID
	return e
}

// WithRetryable marks the error as retryable
func (e *AgentError) WithRetryable(retryable bool) *AgentError {
	e.Retryable = retryable
	return e
}

// IsRetryable checks if the error is retryable
func (e *AgentError) IsRetryable() bool {
	return e.Retryable
}

// captureStack captures the current stack trace
func captureStack(skip int) []StackFrame {
	var frames []StackFrame

	for i := skip; i < skip+10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		frames = append(frames, StackFrame{
			Function: fn.Name(),
			File:     file,
			Line:     line,
		})
	}

	return frames
}

// Predefined errors for common scenarios

// Agent errors
func NewInvalidRequestError(message string) *AgentError {
	return NewAgentError(ComponentAgent, ErrCodeInvalidRequest, message)
}

func NewExecutionTimeoutError(timeout time.Duration) *AgentError {
	return NewAgentError(ComponentAgent, ErrCodeExecutionTimeout,
		fmt.Sprintf("execution timed out after %v", timeout))
}

func NewMaxStepsExceededError(maxSteps int) *AgentError {
	return NewAgentError(ComponentAgent, ErrCodeMaxStepsExceeded,
		fmt.Sprintf("maximum steps exceeded: %d", maxSteps))
}

func NewConcurrencyLimitError(limit int) *AgentError {
	return NewAgentError(ComponentAgent, ErrCodeConcurrencyLimit,
		fmt.Sprintf("concurrency limit reached: %d", limit))
}

func NewSessionNotFoundError(sessionID string) *AgentError {
	return NewAgentError(ComponentAgent, ErrCodeSessionNotFound,
		fmt.Sprintf("session not found: %s", sessionID)).WithSessionID(sessionID)
}

// Reasoning errors
func NewReasoningFailedError(err error) *AgentError {
	return Wrap(err, ComponentReasoning, ErrCodeReasoningFailed, "reasoning failed")
}

func NewModelUnavailableError(model string) *AgentError {
	return NewAgentError(ComponentReasoning, ErrCodeModelUnavailable,
		fmt.Sprintf("model unavailable: %s", model)).WithDetail("model", model)
}

func NewParsingFailedError(err error) *AgentError {
	return Wrap(err, ComponentParser, ErrCodeParsingFailed, "response parsing failed")
}

// Tool errors
func NewToolNotFoundError(toolName string) *AgentError {
	return NewAgentError(ComponentTools, ErrCodeToolNotFound,
		fmt.Sprintf("tool not found: %s", toolName)).WithDetail("tool_name", toolName)
}

func NewToolExecutionFailedError(toolName string, err error) *AgentError {
	return Wrap(err, ComponentTools, ErrCodeToolExecutionFailed,
		fmt.Sprintf("tool execution failed: %s", toolName)).WithDetail("tool_name", toolName)
}

func NewInvalidParametersError(message string) *AgentError {
	return NewAgentError(ComponentTools, ErrCodeInvalidParameters, message)
}

func NewToolTimeoutError(toolName string, timeout time.Duration) *AgentError {
	return NewAgentError(ComponentTools, ErrCodeToolTimeout,
		fmt.Sprintf("tool %s timed out after %v", toolName, timeout)).
		WithDetails(map[string]interface{}{
			"tool_name": toolName,
			"timeout":   timeout.String(),
		})
}

// State errors
func NewStateLoadFailedError(sessionID string, err error) *AgentError {
	return Wrap(err, ComponentState, ErrCodeStateLoadFailed,
		fmt.Sprintf("failed to load state for session %s", sessionID)).
		WithSessionID(sessionID)
}

func NewStateSaveFailedError(sessionID string, err error) *AgentError {
	return Wrap(err, ComponentState, ErrCodeStateSaveFailed,
		fmt.Sprintf("failed to save state for session %s", sessionID)).
		WithSessionID(sessionID)
}

func NewStateCorruptedError(sessionID string) *AgentError {
	return NewAgentError(ComponentState, ErrCodeStateCorrupted,
		fmt.Sprintf("state corrupted for session %s", sessionID)).
		WithSessionID(sessionID)
}

// Configuration errors
func NewInvalidConfigError(message string) *AgentError {
	return NewAgentError(ComponentConfig, ErrCodeInvalidConfig, message)
}

func NewConfigLoadFailedError(err error) *AgentError {
	return Wrap(err, ComponentConfig, ErrCodeConfigLoadFailed, "failed to load configuration")
}

// System errors
func NewSystemError(err error) *AgentError {
	return Wrap(err, "system", ErrCodeSystemError, "system error occurred")
}

func NewResourceExhaustedError(resource string) *AgentError {
	return NewAgentError("system", ErrCodeResourceExhausted,
		fmt.Sprintf("resource exhausted: %s", resource)).WithDetail("resource", resource)
}

// Error classification helpers

// IsTemporary checks if an error is temporary and should be retried
func IsTemporary(err error) bool {
	if agentErr, ok := err.(*AgentError); ok {
		return agentErr.IsRetryable() || isTemporaryCode(agentErr.Code)
	}
	return false
}

// IsFatal checks if an error is fatal and should not be retried
func IsFatal(err error) bool {
	if agentErr, ok := err.(*AgentError); ok {
		return isFatalCode(agentErr.Code)
	}
	return false
}

// isTemporaryCode checks if an error code represents a temporary error
func isTemporaryCode(code string) bool {
	temporaryCodes := map[string]bool{
		ErrCodeExecutionTimeout:   true,
		ErrCodeToolTimeout:        true,
		ErrCodeModelUnavailable:   true,
		ErrCodeServiceUnavailable: true,
		ErrCodeResourceExhausted:  true,
		ErrCodeSystemError:        true,
	}
	return temporaryCodes[code]
}

// isFatalCode checks if an error code represents a fatal error
func isFatalCode(code string) bool {
	fatalCodes := map[string]bool{
		ErrCodeInvalidRequest:         true,
		ErrCodeInvalidParameters:      true,
		ErrCodeInvalidConfig:          true,
		ErrCodeConfigValidationFailed: true,
		ErrCodeStateCorrupted:         true,
		ErrCodeMaxStepsExceeded:       true,
	}
	return fatalCodes[code]
}

// ErrorCollector collects multiple errors
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// Error returns a combined error message
func (ec *ErrorCollector) Error() error {
	if !ec.HasErrors() {
		return nil
	}

	if len(ec.errors) == 1 {
		return ec.errors[0]
	}

	var messages []string
	for _, err := range ec.errors {
		messages = append(messages, err.Error())
	}

	return NewAgentError("collector", "MULTIPLE_ERRORS",
		fmt.Sprintf("multiple errors occurred: %v", messages)).
		WithDetail("error_count", len(ec.errors))
}
