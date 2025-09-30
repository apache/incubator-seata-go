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

package handler

import (
	"context"
	"time"

	"seata-go-ai-a2a/pkg/types"
)

// MessageHandler message processing interface
// Developers implement this interface to define agent's message processing logic
type MessageHandler interface {
	// HandleMessage core method for processing messages
	HandleMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error)

	// HandleCancel cancel task processing (optional implementation)
	HandleCancel(ctx context.Context, taskID string) error
}

// MessageRequest encapsulates all information for processing request
type MessageRequest struct {
	// Original message
	Message *types.Message

	// Task context information
	TaskID    string
	ContextID string

	// Processing configuration
	Configuration *MessageConfiguration

	// Historical messages
	History []*types.Message

	// Request metadata
	Metadata map[string]any

	// Task operations handler
	TaskOperations TaskOperations
}

// MessageConfiguration message processing configuration
type MessageConfiguration struct {
	// Blocking mode - true: wait for completion; false: return immediately
	Blocking bool

	// Streaming mode - true: return event stream; false: return single result
	Streaming bool

	// Historical message length limit
	HistoryLength int

	// Accepted output modes
	AcceptedOutputModes []string

	// Push notification configuration
	PushNotification *types.PushNotificationConfig

	// Request timeout
	Timeout time.Duration
}

// MessageResponse processing result
type MessageResponse struct {
	// Processing mode
	Mode ResponseMode

	// Direct result for non-streaming mode
	Result *ResponseResult

	// Event stream for streaming mode
	EventStream <-chan types.StreamResponse

	// Error information
	Error error
}

// ResponseMode response mode
type ResponseMode int

const (
	ResponseModeMessage ResponseMode = iota // Direct message reply
	ResponseModeTask                        // Task reply
	ResponseModeStream                      // Streaming reply
)

// ResponseResult response result
type ResponseResult struct {
	// Result type - Message or Task
	Data interface{}

	// Task state update (only when Data is Task)
	TaskState types.TaskState

	// Output artifacts
	Artifacts []*types.Artifact
}

// TaskOperations task operations interface
// Provides task management capabilities for MessageHandler
type TaskOperations interface {
	// UpdateTaskState update task state
	UpdateTaskState(state types.TaskState, message *types.Message) error

	// AddArtifact add output artifact
	AddArtifact(artifact *types.Artifact) error

	// GetTaskHistory get task history
	GetTaskHistory() ([]*types.Message, error)

	// GetTaskMetadata get task metadata
	GetTaskMetadata() (map[string]any, error)

	// SendEvent send event to stream (streaming mode)
	SendEvent(event types.StreamResponse) error

	// GetTaskID get current task ID
	GetTaskID() string

	// GetContextID get current context ID
	GetContextID() string
}

// AgentCardProvider agent card provider interface
// Developers implement this interface to provide agent's capability description
type AgentCardProvider interface {
	// GetAgentCard get agent card
	GetAgentCard(ctx context.Context) (*types.AgentCard, error)

	// GetAgentCardForUser get user-specific agent card (supports extended card after authentication)
	GetAgentCardForUser(ctx context.Context, userInfo *UserInfo) (*types.AgentCard, error)
}

// UserInfo user information
type UserInfo struct {
	UserID   string
	Scopes   []string
	Metadata map[string]any
}
