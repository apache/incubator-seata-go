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

package types

import (
	"fmt"

	pb "seata-go-ai-a2a/pkg/proto/v1"
)

// TaskState represents the state of a task in the A2A protocol
type TaskState int

const (
	TaskStateUnspecified TaskState = iota
	TaskStateSubmitted
	TaskStateWorking
	TaskStateCompleted
	TaskStateFailed
	TaskStateCancelled
	TaskStateInputRequired
	TaskStateRejected
	TaskStateAuthRequired
)

// String returns the string representation of TaskState
func (t TaskState) String() string {
	switch t {
	case TaskStateUnspecified:
		return "TASK_STATE_UNSPECIFIED"
	case TaskStateSubmitted:
		return "TASK_STATE_SUBMITTED"
	case TaskStateWorking:
		return "TASK_STATE_WORKING"
	case TaskStateCompleted:
		return "TASK_STATE_COMPLETED"
	case TaskStateFailed:
		return "TASK_STATE_FAILED"
	case TaskStateCancelled:
		return "TASK_STATE_CANCELLED"
	case TaskStateInputRequired:
		return "TASK_STATE_INPUT_REQUIRED"
	case TaskStateRejected:
		return "TASK_STATE_REJECTED"
	case TaskStateAuthRequired:
		return "TASK_STATE_AUTH_REQUIRED"
	default:
		return "TASK_STATE_UNKNOWN"
	}
}

// IsTerminal returns true if the task state is terminal (no further transitions expected)
func (t TaskState) IsTerminal() bool {
	switch t {
	case TaskStateCompleted, TaskStateFailed, TaskStateCancelled, TaskStateRejected:
		return true
	default:
		return false
	}
}

// IsInterrupted returns true if the task state is interrupted (waiting for input or auth)
func (t TaskState) IsInterrupted() bool {
	switch t {
	case TaskStateInputRequired, TaskStateAuthRequired:
		return true
	default:
		return false
	}
}

// Role represents the role of a message sender in A2A protocol
type Role int

const (
	RoleUnspecified Role = iota
	RoleUser
	RoleAgent
)

// String returns the string representation of Role
func (r Role) String() string {
	switch r {
	case RoleUnspecified:
		return "ROLE_UNSPECIFIED"
	case RoleUser:
		return "ROLE_USER"
	case RoleAgent:
		return "ROLE_AGENT"
	default:
		return "ROLE_UNKNOWN"
	}
}

// TaskStateToProto converts TaskState to pb.TaskState
func TaskStateToProto(state TaskState) pb.TaskState {
	switch state {
	case TaskStateUnspecified:
		return pb.TaskState_TASK_STATE_UNSPECIFIED
	case TaskStateSubmitted:
		return pb.TaskState_TASK_STATE_SUBMITTED
	case TaskStateWorking:
		return pb.TaskState_TASK_STATE_WORKING
	case TaskStateCompleted:
		return pb.TaskState_TASK_STATE_COMPLETED
	case TaskStateFailed:
		return pb.TaskState_TASK_STATE_FAILED
	case TaskStateCancelled:
		return pb.TaskState_TASK_STATE_CANCELLED
	case TaskStateInputRequired:
		return pb.TaskState_TASK_STATE_INPUT_REQUIRED
	case TaskStateRejected:
		return pb.TaskState_TASK_STATE_REJECTED
	case TaskStateAuthRequired:
		return pb.TaskState_TASK_STATE_AUTH_REQUIRED
	default:
		return pb.TaskState_TASK_STATE_UNSPECIFIED
	}
}

// TaskStateFromProto converts pb.TaskState to TaskState
func TaskStateFromProto(state pb.TaskState) TaskState {
	switch state {
	case pb.TaskState_TASK_STATE_UNSPECIFIED:
		return TaskStateUnspecified
	case pb.TaskState_TASK_STATE_SUBMITTED:
		return TaskStateSubmitted
	case pb.TaskState_TASK_STATE_WORKING:
		return TaskStateWorking
	case pb.TaskState_TASK_STATE_COMPLETED:
		return TaskStateCompleted
	case pb.TaskState_TASK_STATE_FAILED:
		return TaskStateFailed
	case pb.TaskState_TASK_STATE_CANCELLED:
		return TaskStateCancelled
	case pb.TaskState_TASK_STATE_INPUT_REQUIRED:
		return TaskStateInputRequired
	case pb.TaskState_TASK_STATE_REJECTED:
		return TaskStateRejected
	case pb.TaskState_TASK_STATE_AUTH_REQUIRED:
		return TaskStateAuthRequired
	default:
		return TaskStateUnspecified
	}
}

// ParseTaskState parses a string into a TaskState
func ParseTaskState(s string) (TaskState, error) {
	switch s {
	case "TASK_STATE_UNSPECIFIED":
		return TaskStateUnspecified, nil
	case "TASK_STATE_SUBMITTED":
		return TaskStateSubmitted, nil
	case "TASK_STATE_WORKING":
		return TaskStateWorking, nil
	case "TASK_STATE_COMPLETED":
		return TaskStateCompleted, nil
	case "TASK_STATE_FAILED":
		return TaskStateFailed, nil
	case "TASK_STATE_CANCELLED":
		return TaskStateCancelled, nil
	case "TASK_STATE_INPUT_REQUIRED":
		return TaskStateInputRequired, nil
	case "TASK_STATE_REJECTED":
		return TaskStateRejected, nil
	case "TASK_STATE_AUTH_REQUIRED":
		return TaskStateAuthRequired, nil
	default:
		return TaskStateUnspecified, fmt.Errorf("unknown task state: %s", s)
	}
}

// RoleToProto converts Role to pb.Role
func RoleToProto(role Role) pb.Role {
	switch role {
	case RoleUnspecified:
		return pb.Role_ROLE_UNSPECIFIED
	case RoleUser:
		return pb.Role_ROLE_USER
	case RoleAgent:
		return pb.Role_ROLE_AGENT
	default:
		return pb.Role_ROLE_UNSPECIFIED
	}
}

// RoleFromProto converts pb.Role to Role
func RoleFromProto(role pb.Role) Role {
	switch role {
	case pb.Role_ROLE_UNSPECIFIED:
		return RoleUnspecified
	case pb.Role_ROLE_USER:
		return RoleUser
	case pb.Role_ROLE_AGENT:
		return RoleAgent
	default:
		return RoleUnspecified
	}
}

// A2A specific error codes as defined in the A2A specification
const (
	// TaskNotFoundError indicates that the specified task was not found
	TaskNotFoundError = -32001
	// TaskNotCancelableError indicates that the task cannot be cancelled in its current state
	TaskNotCancelableError = -32002
	// PushNotificationNotSupportedError indicates that push notifications are not supported
	PushNotificationNotSupportedError = -32003
	// UnsupportedOperationError indicates that the requested operation is not supported
	UnsupportedOperationError = -32004
	// ContentTypeNotSupportedError indicates that the content type is not supported
	ContentTypeNotSupportedError = -32005
	// InvalidAgentResponseError indicates that the agent response is invalid
	InvalidAgentResponseError = -32006
	// AuthenticatedExtendedCardNotConfiguredError indicates that authenticated extended card is not configured
	AuthenticatedExtendedCardNotConfiguredError = -32007
)

// A2AError represents an A2A specific error with the defined error codes
type A2AError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error implements the error interface
func (e *A2AError) Error() string {
	return fmt.Sprintf("A2A Error %d: %s", e.Code, e.Message)
}

// NewTaskNotFoundError creates a new TaskNotFoundError
func NewTaskNotFoundError(taskID string) *A2AError {
	return &A2AError{
		Code:    TaskNotFoundError,
		Message: fmt.Sprintf("Task not found: %s", taskID),
		Data:    map[string]string{"task_id": taskID},
	}
}

// NewTaskNotCancelableError creates a new TaskNotCancelableError
func NewTaskNotCancelableError(taskID string, currentState TaskState) *A2AError {
	return &A2AError{
		Code:    TaskNotCancelableError,
		Message: fmt.Sprintf("Task %s cannot be cancelled in state %s", taskID, currentState.String()),
		Data: map[string]any{
			"task_id":       taskID,
			"current_state": currentState.String(),
		},
	}
}

// NewUnsupportedOperationError creates a new UnsupportedOperationError
func NewUnsupportedOperationError(operation string) *A2AError {
	return &A2AError{
		Code:    UnsupportedOperationError,
		Message: fmt.Sprintf("Operation not supported: %s", operation),
		Data:    map[string]string{"operation": operation},
	}
}

// NewContentTypeNotSupportedError creates a new ContentTypeNotSupportedError
func NewContentTypeNotSupportedError(contentType string) *A2AError {
	return &A2AError{
		Code:    ContentTypeNotSupportedError,
		Message: fmt.Sprintf("Content type not supported: %s", contentType),
		Data:    map[string]string{"content_type": contentType},
	}
}

// NewInvalidAgentResponseError creates a new InvalidAgentResponseError
func NewInvalidAgentResponseError(reason string) *A2AError {
	return &A2AError{
		Code:    InvalidAgentResponseError,
		Message: fmt.Sprintf("Invalid agent response: %s", reason),
		Data:    map[string]string{"reason": reason},
	}
}

// NewPushNotificationNotSupportedError creates a new PushNotificationNotSupportedError
func NewPushNotificationNotSupportedError() *A2AError {
	return &A2AError{
		Code:    PushNotificationNotSupportedError,
		Message: "Push notifications not supported",
	}
}

// NewAuthenticatedExtendedCardNotConfiguredError creates a new AuthenticatedExtendedCardNotConfiguredError
func NewAuthenticatedExtendedCardNotConfiguredError() *A2AError {
	return &A2AError{
		Code:    AuthenticatedExtendedCardNotConfiguredError,
		Message: "Authenticated extended card not configured",
	}
}
