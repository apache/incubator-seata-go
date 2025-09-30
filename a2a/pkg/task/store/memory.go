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

package store

import (
	"context"
	"fmt"
	"seata-go-ai-a2a/pkg/types"
	"sync"
)

// MemoryTaskStore implements TaskStore interface using in-memory storage
// This implementation is thread-safe and suitable for development and testing
type MemoryTaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*types.Task
}

// NewMemoryTaskStore creates a new in-memory task store
func NewMemoryTaskStore() *MemoryTaskStore {
	return &MemoryTaskStore{
		tasks: make(map[string]*types.Task),
	}
}

// CreateTask stores a new task in memory
func (m *MemoryTaskStore) CreateTask(ctx context.Context, task *types.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// Deep copy the task to avoid external mutations
	taskCopy := m.copyTask(task)
	m.tasks[task.ID] = taskCopy

	return nil
}

// GetTask retrieves a task by ID
func (m *MemoryTaskStore) GetTask(ctx context.Context, taskID string) (*types.Task, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", taskID)
	}

	// Return a copy to prevent external mutations
	return m.copyTask(task), nil
}

// UpdateTask updates an existing task
func (m *MemoryTaskStore) UpdateTask(ctx context.Context, task *types.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[task.ID]; !exists {
		return fmt.Errorf("task with ID %s not found", task.ID)
	}

	// Deep copy the task to avoid external mutations
	taskCopy := m.copyTask(task)
	m.tasks[task.ID] = taskCopy

	return nil
}

// DeleteTask removes a task from memory
func (m *MemoryTaskStore) DeleteTask(ctx context.Context, taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[taskID]; !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	delete(m.tasks, taskID)
	return nil
}

// UpdateTaskStatus updates the status of an existing task
func (m *MemoryTaskStore) UpdateTaskStatus(ctx context.Context, taskID string, status *types.TaskStatus) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if status == nil {
		return fmt.Errorf("status cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	// Update the status with a copy
	task.Status = m.copyTaskStatus(status)
	return nil
}

// AddArtifact adds an artifact to a task
func (m *MemoryTaskStore) AddArtifact(ctx context.Context, taskID string, artifact *types.Artifact) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if artifact == nil {
		return fmt.Errorf("artifact cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	// Add a copy of the artifact
	artifactCopy := m.copyArtifact(artifact)
	task.Artifacts = append(task.Artifacts, artifactCopy)
	return nil
}

// GetArtifacts retrieves all artifacts for a task
func (m *MemoryTaskStore) GetArtifacts(ctx context.Context, taskID string) ([]*types.Artifact, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", taskID)
	}

	// Return copies of artifacts
	artifacts := make([]*types.Artifact, len(task.Artifacts))
	for i, artifact := range task.Artifacts {
		artifacts[i] = m.copyArtifact(artifact)
	}

	return artifacts, nil
}

// AddMessage adds a message to task history
func (m *MemoryTaskStore) AddMessage(ctx context.Context, taskID string, message *types.Message) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	// Add a copy of the message
	messageCopy := m.copyMessage(message)
	task.History = append(task.History, messageCopy)
	return nil
}

// GetHistory retrieves task history with optional limit
func (m *MemoryTaskStore) GetHistory(ctx context.Context, taskID string, limit int) ([]*types.Message, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", taskID)
	}

	history := task.History
	if limit > 0 && len(history) > limit {
		// Return the most recent messages
		history = history[len(history)-limit:]
	}

	// Return copies of messages
	messages := make([]*types.Message, len(history))
	for i, message := range history {
		messages[i] = m.copyMessage(message)
	}

	return messages, nil
}

// ListTasksByContext lists all tasks for a specific context
func (m *MemoryTaskStore) ListTasksByContext(ctx context.Context, contextID string) ([]*types.Task, error) {
	if contextID == "" {
		return nil, fmt.Errorf("context ID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*types.Task
	for _, task := range m.tasks {
		if task.ContextID == contextID {
			tasks = append(tasks, m.copyTask(task))
		}
	}

	return tasks, nil
}

// ListTasksByState lists all tasks with specified states
func (m *MemoryTaskStore) ListTasksByState(ctx context.Context, states []types.TaskState) ([]*types.Task, error) {
	if len(states) == 0 {
		return nil, fmt.Errorf("at least one state must be specified")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	stateSet := make(map[types.TaskState]bool)
	for _, state := range states {
		stateSet[state] = true
	}

	var tasks []*types.Task
	for _, task := range m.tasks {
		if task.Status != nil && stateSet[task.Status.State] {
			tasks = append(tasks, m.copyTask(task))
		}
	}

	return tasks, nil
}

// HealthCheck checks the health of the memory store
func (m *MemoryTaskStore) HealthCheck(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Memory store is always healthy if accessible
	return nil
}

// Close cleans up the memory store
func (m *MemoryTaskStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all tasks
	m.tasks = make(map[string]*types.Task)
	return nil
}

// Helper methods for deep copying

func (m *MemoryTaskStore) copyTask(task *types.Task) *types.Task {
	if task == nil {
		return nil
	}

	copy := &types.Task{
		ID:        task.ID,
		ContextID: task.ContextID,
		Status:    m.copyTaskStatus(task.Status),
		Metadata:  m.copyMetadata(task.Metadata),
	}

	// Copy artifacts
	if task.Artifacts != nil {
		copy.Artifacts = make([]*types.Artifact, len(task.Artifacts))
		for i, artifact := range task.Artifacts {
			copy.Artifacts[i] = m.copyArtifact(artifact)
		}
	}

	// Copy history
	if task.History != nil {
		copy.History = make([]*types.Message, len(task.History))
		for i, message := range task.History {
			copy.History[i] = m.copyMessage(message)
		}
	}

	return copy
}

func (m *MemoryTaskStore) copyTaskStatus(status *types.TaskStatus) *types.TaskStatus {
	if status == nil {
		return nil
	}

	return &types.TaskStatus{
		State:     status.State,
		Update:    m.copyMessage(status.Update),
		Timestamp: status.Timestamp,
	}
}

func (m *MemoryTaskStore) copyArtifact(artifact *types.Artifact) *types.Artifact {
	if artifact == nil {
		return nil
	}

	copy := &types.Artifact{
		ArtifactID:  artifact.ArtifactID,
		Name:        artifact.Name,
		Description: artifact.Description,
		Metadata:    m.copyMetadata(artifact.Metadata),
		Extensions:  m.copyStringSlice(artifact.Extensions),
	}

	// Copy parts
	if artifact.Parts != nil {
		copy.Parts = make([]types.Part, len(artifact.Parts))
		for i, part := range artifact.Parts {
			copy.Parts[i] = m.copyPart(part)
		}
	}

	return copy
}

func (m *MemoryTaskStore) copyMessage(message *types.Message) *types.Message {
	if message == nil {
		return nil
	}

	copy := &types.Message{
		MessageID:  message.MessageID,
		ContextID:  message.ContextID,
		TaskID:     message.TaskID,
		Role:       message.Role,
		Metadata:   m.copyMetadata(message.Metadata),
		Extensions: m.copyStringSlice(message.Extensions),
	}

	// Copy content parts
	if message.Parts != nil {
		copy.Parts = make([]types.Part, len(message.Parts))
		for i, part := range message.Parts {
			copy.Parts[i] = m.copyPart(part)
		}
	}

	return copy
}

func (m *MemoryTaskStore) copyPart(part types.Part) types.Part {
	if part == nil {
		return nil
	}

	switch p := part.(type) {
	case *types.TextPart:
		return &types.TextPart{
			Text:     p.Text,
			Metadata: m.copyMetadata(p.Metadata),
		}
	case *types.FilePart:
		return &types.FilePart{
			Content:  m.copyFileContent(p.Content),
			MimeType: p.MimeType,
			Name:     p.Name,
			Metadata: m.copyMetadata(p.Metadata),
		}
	case *types.DataPart:
		return &types.DataPart{
			Data:     m.copyMetadata(p.Data),
			Metadata: m.copyMetadata(p.Metadata),
		}
	default:
		// Return the original part if type is unknown
		return part
	}
}

func (m *MemoryTaskStore) copyFileContent(content types.FileContent) types.FileContent {
	if content == nil {
		return nil
	}

	switch c := content.(type) {
	case *types.FileWithURI:
		return &types.FileWithURI{URI: c.URI}
	case *types.FileWithBytes:
		bytes := make([]byte, len(c.Bytes))
		copy(bytes, c.Bytes)
		return &types.FileWithBytes{Bytes: bytes}
	default:
		return content
	}
}

func (m *MemoryTaskStore) copyMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}

	copy := make(map[string]any, len(metadata))
	for k, v := range metadata {
		copy[k] = v
	}
	return copy
}

func (m *MemoryTaskStore) copyStringSlice(slice []string) []string {
	if slice == nil {
		return nil
	}

	copy := make([]string, len(slice))
	for i, s := range slice {
		copy[i] = s
	}
	return copy
}
