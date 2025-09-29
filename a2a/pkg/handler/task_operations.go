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
	"fmt"
	"sync"

	"seata-go-ai-a2a/pkg/types"
)

// taskOperationsImpl concrete implementation of TaskOperations interface
// It encapsulates interaction with TaskManager
type taskOperationsImpl struct {
	taskManager types.TaskManagerInterface
	taskID      string
	contextID   string

	// Event channel for streaming mode
	eventChan chan types.StreamResponse
	mu        sync.RWMutex
}

// NewTaskOperations creates TaskOperations implementation
func NewTaskOperations(taskManager types.TaskManagerInterface, taskID, contextID string, eventChan chan types.StreamResponse) TaskOperations {
	return &taskOperationsImpl{
		taskManager: taskManager,
		taskID:      taskID,
		contextID:   contextID,
		eventChan:   eventChan,
	}
}

// UpdateTaskState update task state
func (t *taskOperationsImpl) UpdateTaskState(state types.TaskState, message *types.Message) error {
	if t.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}

	// Update task state
	err := t.taskManager.UpdateTaskStatus(nil, t.taskID, state, message)
	if err != nil {
		return fmt.Errorf("failed to update task state: %w", err)
	}

	// If in streaming mode, send status update event
	if t.eventChan != nil {
		event := &types.TaskStatusUpdateEvent{
			TaskID:    t.taskID,
			ContextID: t.contextID,
			Status: &types.TaskStatus{
				State:  state,
				Update: message,
			},
			Final: isTerminalState(state),
		}

		return t.SendEvent(&types.StatusUpdateStreamResponse{
			StatusUpdate: event,
		})
	}

	return nil
}

// AddArtifact add output artifact
func (t *taskOperationsImpl) AddArtifact(artifact *types.Artifact) error {
	if t.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}

	// Get task and add artifact
	task, err := t.taskManager.GetTask(nil, t.taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("task not found: %s", t.taskID)
	}

	// Use AddArtifact method from TaskManagerInterface
	err = t.taskManager.AddArtifact(nil, t.taskID, artifact, false, true)
	if err != nil {
		return fmt.Errorf("failed to add artifact: %w", err)
	}

	// If in streaming mode, send artifact update event
	if t.eventChan != nil {
		event := &types.TaskArtifactUpdateEvent{
			TaskID:    t.taskID,
			ContextID: t.contextID,
			Artifact:  artifact,
			Append:    false,
			LastChunk: true,
		}

		return t.SendEvent(&types.ArtifactUpdateStreamResponse{
			ArtifactUpdate: event,
		})
	}

	return nil
}

// GetTaskHistory get task history
func (t *taskOperationsImpl) GetTaskHistory() ([]*types.Message, error) {
	if t.taskManager == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}

	// Get task details
	task, err := t.taskManager.GetTask(nil, t.taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if task == nil {
		return nil, fmt.Errorf("task not found: %s", t.taskID)
	}

	return task.History, nil
}

// GetTaskMetadata get task metadata
func (t *taskOperationsImpl) GetTaskMetadata() (map[string]any, error) {
	if t.taskManager == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}

	// Get task details
	task, err := t.taskManager.GetTask(nil, t.taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if task == nil {
		return nil, fmt.Errorf("task not found: %s", t.taskID)
	}

	// Return metadata as map
	if task.Metadata != nil {
		return task.Metadata, nil
	}

	return make(map[string]any), nil
}

// SendEvent send event to stream (streaming mode)
func (t *taskOperationsImpl) SendEvent(event types.StreamResponse) error {
	if t.eventChan == nil {
		return fmt.Errorf("streaming not enabled")
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	select {
	case t.eventChan <- event:
		return nil
	default:
		return fmt.Errorf("event channel is full or closed")
	}
}

// GetTaskID get current task ID
func (t *taskOperationsImpl) GetTaskID() string {
	return t.taskID
}

// GetContextID get current context ID
func (t *taskOperationsImpl) GetContextID() string {
	return t.contextID
}

// isTerminalState check if state is terminal
func isTerminalState(state types.TaskState) bool {
	switch state {
	case types.TaskStateCompleted,
		types.TaskStateFailed,
		types.TaskStateCancelled,
		types.TaskStateRejected:
		return true
	default:
		return false
	}
}

// CloseEventChannel close event channel (used when streaming mode ends)
func (t *taskOperationsImpl) CloseEventChannel() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.eventChan != nil {
		close(t.eventChan)
		t.eventChan = nil
	}
}
