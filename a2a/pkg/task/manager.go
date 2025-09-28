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

package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"seata-go-ai-a2a/pkg/handler"
	"seata-go-ai-a2a/pkg/task/store"
	"seata-go-ai-a2a/pkg/types"
)

// TaskManager manages the complete lifecycle of A2A tasks
// It provides task creation, status updates, artifact management, and event streaming
type TaskManager struct {
	store         store.TaskStore
	eventBus      *EventBus
	config        *Config
	logger        types.Logger
	mu            sync.RWMutex
	closing       bool
	closeOnce     sync.Once
	cleanupStopCh chan struct{}
	cleanupDone   chan struct{}

	// Handler interfaces (optional)
	messageHandler    handler.MessageHandler
	agentCardProvider handler.AgentCardProvider
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

// UserInfo user information
type UserInfo struct {
	UserID   string
	Scopes   []string
	Metadata map[string]any
}

// Config configures the TaskManager
type Config struct {
	// Maximum number of messages to keep in task history (0 = unlimited)
	MaxHistoryLength int

	// Maximum number of artifacts per task (0 = unlimited)
	MaxArtifacts int

	// Default timeout for operations
	DefaultTimeout time.Duration

	// Cleanup interval for terminated tasks
	CleanupInterval time.Duration

	// TTL for terminated tasks in memory
	TerminatedTaskTTL time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxHistoryLength:  100,
		MaxArtifacts:      50,
		DefaultTimeout:    30 * time.Second,
		CleanupInterval:   5 * time.Minute,
		TerminatedTaskTTL: 1 * time.Hour,
	}
}

// NewTaskManager creates a new task manager with the given store and configuration
func NewTaskManager(taskStore store.TaskStore, config *Config) *TaskManager {
	return NewTaskManagerWithLogger(taskStore, config, &types.DefaultLogger{})
}

// NewTaskManagerWithLogger creates a new task manager with a custom logger
func NewTaskManagerWithLogger(taskStore store.TaskStore, config *Config, logger types.Logger) *TaskManager {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = &types.DefaultLogger{}
	}

	eventBus := NewEventBus()

	tm := &TaskManager{
		store:         taskStore,
		eventBus:      eventBus,
		config:        config,
		logger:        logger,
		cleanupStopCh: make(chan struct{}),
		cleanupDone:   make(chan struct{}),
	}

	// Start cleanup routine
	go tm.cleanupRoutine()

	return tm
}

// NewTaskManagerWithHandlers creates a new task manager with handler interfaces
func NewTaskManagerWithHandlers(taskStore store.TaskStore, config *Config, messageHandler handler.MessageHandler, agentCardProvider handler.AgentCardProvider) *TaskManager {
	return NewTaskManagerWithHandlersAndLogger(taskStore, config, messageHandler, agentCardProvider, &types.DefaultLogger{})
}

// NewTaskManagerWithHandlersAndLogger creates a new task manager with handler interfaces and custom logger
func NewTaskManagerWithHandlersAndLogger(taskStore store.TaskStore, config *Config, messageHandler handler.MessageHandler, agentCardProvider handler.AgentCardProvider, logger types.Logger) *TaskManager {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = &types.DefaultLogger{}
	}

	eventBus := NewEventBus()

	tm := &TaskManager{
		store:             taskStore,
		eventBus:          eventBus,
		config:            config,
		logger:            logger,
		messageHandler:    messageHandler,
		agentCardProvider: agentCardProvider,
		cleanupStopCh:     make(chan struct{}),
		cleanupDone:       make(chan struct{}),
	}

	// Start cleanup routine
	go tm.cleanupRoutine()

	return tm
}

// CreateTask creates a new task from an initial message
func (tm *TaskManager) CreateTask(ctx context.Context, contextID string, message *types.Message, metadata map[string]any) (*types.Task, error) {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	// Generate task ID
	taskID := uuid.New().String()

	// Create initial status
	status := &types.TaskStatus{
		State:     types.TaskStateSubmitted,
		Update:    message,
		Timestamp: time.Now(),
	}

	// Create task with initial message in history
	task := &types.Task{
		ID:        taskID,
		ContextID: contextID,
		Status:    status,
		History:   []*types.Message{message},
		Kind:      "task",
		Metadata:  metadata,
	}

	// Store the task
	if err := tm.store.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Publish task creation event
	taskResponse := &types.TaskStreamResponse{Task: task}
	tm.eventBus.PublishToTask(taskID, taskResponse)

	return task, nil
}

// GetTask retrieves a task by ID
func (tm *TaskManager) GetTask(ctx context.Context, taskID string) (*types.Task, error) {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	task, err := tm.store.GetTask(ctx, taskID)
	if err != nil {
		// Check if it's a "not found" error and convert to A2A error
		if err.Error() == fmt.Sprintf("task not found: %s", taskID) ||
			err.Error() == fmt.Sprintf("task with ID %s not found", taskID) {
			return nil, types.NewTaskNotFoundError(taskID)
		}
		return nil, err
	}
	return task, nil
}

// UpdateTaskStatus updates the status of a task and publishes the event
func (tm *TaskManager) UpdateTaskStatus(ctx context.Context, taskID string, newState types.TaskState, updateMessage *types.Message) error {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	// Get current task
	task, err := tm.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Validate state transition
	if !tm.isValidStateTransition(task.Status.State, newState) {
		return fmt.Errorf("invalid state transition from %s to %s", task.Status.State.String(), newState.String())
	}

	// Create new status
	newStatus := &types.TaskStatus{
		State:     newState,
		Update:    updateMessage,
		Timestamp: time.Now(),
	}

	// Update task status in store
	if err := tm.store.UpdateTaskStatus(ctx, taskID, newStatus); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Add update message to history if provided
	if updateMessage != nil {
		if err := tm.AddMessage(ctx, taskID, updateMessage); err != nil {
			// Log error but don't fail the status update
			tm.logger.Warn(ctx, "Failed to add update message to history",
				"task_id", taskID,
				"error", err,
			)
		}
	}

	// Create and publish status update event
	statusEvent := &types.TaskStatusUpdateEvent{
		TaskID:    taskID,
		ContextID: task.ContextID,
		Status:    newStatus,
		Final:     newState.IsTerminal(),
	}

	statusResponse := &types.StatusUpdateStreamResponse{StatusUpdate: statusEvent}
	tm.eventBus.PublishToTask(taskID, statusResponse)

	return nil
}

// AddArtifact adds an artifact to a task and publishes the event
func (tm *TaskManager) AddArtifact(ctx context.Context, taskID string, artifact *types.Artifact, append, lastChunk bool) error {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	// Get task to validate it exists
	task, err := tm.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check artifact limit
	if tm.config.MaxArtifacts > 0 && len(task.Artifacts) >= tm.config.MaxArtifacts {
		return fmt.Errorf("maximum artifacts limit (%d) reached for task", tm.config.MaxArtifacts)
	}

	// Add artifact to store
	if err := tm.store.AddArtifact(ctx, taskID, artifact); err != nil {
		return fmt.Errorf("failed to add artifact: %w", err)
	}

	// Create and publish artifact update event
	artifactEvent := &types.TaskArtifactUpdateEvent{
		TaskID:    taskID,
		ContextID: task.ContextID,
		Artifact:  artifact,
		Append:    append,
		LastChunk: lastChunk,
	}

	artifactResponse := &types.ArtifactUpdateStreamResponse{ArtifactUpdate: artifactEvent}
	tm.eventBus.PublishToTask(taskID, artifactResponse)

	return nil
}

// AddMessage adds a message to task history
func (tm *TaskManager) AddMessage(ctx context.Context, taskID string, message *types.Message) error {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	// Check history limit if configured
	if tm.config.MaxHistoryLength > 0 {
		history, err := tm.store.GetHistory(ctx, taskID, 0) // Get all history
		if err != nil {
			return fmt.Errorf("failed to check history length: %w", err)
		}

		if len(history) >= tm.config.MaxHistoryLength {
			return fmt.Errorf("maximum history length (%d) reached for task", tm.config.MaxHistoryLength)
		}
	}

	return tm.store.AddMessage(ctx, taskID, message)
}

// CancelTask cancels a running task
func (tm *TaskManager) CancelTask(ctx context.Context, taskID string) (*types.Task, error) {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	// Get current task
	task, err := tm.store.GetTask(ctx, taskID)
	if err != nil {
		// Check if it's a "not found" error and convert to A2A error
		if err.Error() == fmt.Sprintf("task not found: %s", taskID) ||
			err.Error() == fmt.Sprintf("task with ID %s not found", taskID) {
			return nil, types.NewTaskNotFoundError(taskID)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Check if task can be cancelled
	if task.Status.State.IsTerminal() {
		return nil, types.NewTaskNotCancelableError(taskID, task.Status.State)
	}

	// Update status to cancelled
	if err := tm.UpdateTaskStatus(ctx, taskID, types.TaskStateCancelled, nil); err != nil {
		return nil, fmt.Errorf("failed to cancel task: %w", err)
	}

	// Return updated task
	return tm.store.GetTask(ctx, taskID)
}

// SubscribeToTask creates a subscription to task events
func (tm *TaskManager) SubscribeToTask(ctx context.Context, taskID string) (*TaskSubscription, error) {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	// Verify task exists
	task, err := tm.store.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Create subscription
	subscription := NewTaskSubscription(taskID, tm.eventBus)

	// If task is already completed, send current state and close
	if task.Status.State.IsTerminal() || task.Status.State.IsInterrupted() {
		go func() {
			defer subscription.Close()

			// Send current task state
			taskResponse := &types.TaskStreamResponse{Task: task}
			select {
			case subscription.events <- taskResponse:
			case <-time.After(5 * time.Second):
				// Timeout sending to closed subscription
			}
		}()
	}

	return subscription, nil
}

// GetTaskHistory retrieves task history with optional limit
func (tm *TaskManager) GetTaskHistory(ctx context.Context, taskID string, limit int) ([]*types.Message, error) {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	return tm.store.GetHistory(ctx, taskID, limit)
}

// ListTasksByContext lists all tasks for a context
func (tm *TaskManager) ListTasksByContext(ctx context.Context, contextID string) ([]*types.Task, error) {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	return tm.store.ListTasksByContext(ctx, contextID)
}

// ListActiveTasks lists all non-terminal tasks
func (tm *TaskManager) ListActiveTasks(ctx context.Context) ([]*types.Task, error) {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	activeStates := []types.TaskState{
		types.TaskStateSubmitted,
		types.TaskStateWorking,
		types.TaskStateInputRequired,
		types.TaskStateAuthRequired,
	}

	return tm.store.ListTasksByState(ctx, activeStates)
}

// ListTasks lists tasks with optional filtering and pagination
func (tm *TaskManager) ListTasks(ctx context.Context, req *types.ListTasksRequest) (*types.ListTasksResponse, error) {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	var tasks []*types.Task
	var err error

	// If context ID is specified, filter by context
	if req.ContextID != "" {
		tasks, err = tm.store.ListTasksByContext(ctx, req.ContextID)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks by context: %w", err)
		}
	} else if len(req.States) > 0 {
		// Filter by states if specified
		tasks, err = tm.store.ListTasksByState(ctx, req.States)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks by state: %w", err)
		}
	} else {
		// List all tasks (would need a new store method for this)
		// For now, list active tasks as fallback
		activeStates := []types.TaskState{
			types.TaskStateSubmitted,
			types.TaskStateWorking,
			types.TaskStateInputRequired,
			types.TaskStateAuthRequired,
			types.TaskStateCompleted,
			types.TaskStateFailed,
			types.TaskStateCancelled,
			types.TaskStateRejected,
		}
		tasks, err = tm.store.ListTasksByState(ctx, activeStates)
		if err != nil {
			return nil, fmt.Errorf("failed to list all tasks: %w", err)
		}
	}

	// Apply pagination if requested
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 50 // Default page size
	}

	// Simple pagination implementation
	// In a real implementation, you'd want proper cursor-based pagination
	startIndex := 0
	if req.PageToken != "" {
		// Parse page token (simplified - in practice you'd want proper encoding)
		fmt.Sscanf(req.PageToken, "%d", &startIndex)
	}

	endIndex := startIndex + pageSize
	var nextPageToken string

	if endIndex < len(tasks) {
		nextPageToken = fmt.Sprintf("%d", endIndex)
		tasks = tasks[startIndex:endIndex]
	} else if startIndex < len(tasks) {
		tasks = tasks[startIndex:]
	} else {
		tasks = []*types.Task{}
	}

	return &types.ListTasksResponse{
		Tasks:         tasks,
		NextPageToken: nextPageToken,
	}, nil
}

// Close shuts down the task manager
func (tm *TaskManager) Close(ctx context.Context) error {
	var err error
	tm.closeOnce.Do(func() {
		tm.mu.Lock()
		tm.closing = true
		tm.mu.Unlock()

		// Stop cleanup routine gracefully
		close(tm.cleanupStopCh)

		// Wait for cleanup routine to finish or timeout
		select {
		case <-tm.cleanupDone:
			// Cleanup routine finished gracefully
		case <-ctx.Done():
			// Context cancelled, proceed with cleanup
			tm.logger.Warn(ctx, "Cleanup routine did not finish within context timeout")
		case <-time.After(5 * time.Second):
			// Fallback timeout
			tm.logger.Warn(ctx, "Cleanup routine did not finish within timeout")
		}

		// Close event bus
		tm.eventBus.Close()

		// Close store
		if closeErr := tm.store.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close store: %w", closeErr)
		}
	})

	return err
}

// HealthCheck checks the health of the task manager
func (tm *TaskManager) HealthCheck(ctx context.Context) error {
	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return fmt.Errorf("task manager is closing")
	}
	tm.mu.RUnlock()

	return tm.store.HealthCheck(ctx)
}

// isValidStateTransition validates task state transitions
func (tm *TaskManager) isValidStateTransition(from, to types.TaskState) bool {
	// Allow any transition from unspecified
	if from == types.TaskStateUnspecified {
		return true
	}

	// No transitions from terminal states (except cancellation handling)
	if from.IsTerminal() && to != from {
		return false
	}

	// Define valid transitions
	validTransitions := map[types.TaskState][]types.TaskState{
		types.TaskStateSubmitted: {
			types.TaskStateWorking,
			types.TaskStateRejected,
			types.TaskStateCancelled,
			types.TaskStateAuthRequired,
		},
		types.TaskStateWorking: {
			types.TaskStateCompleted,
			types.TaskStateFailed,
			types.TaskStateCancelled,
			types.TaskStateInputRequired,
		},
		types.TaskStateInputRequired: {
			types.TaskStateWorking,
			types.TaskStateCancelled,
			types.TaskStateFailed,
		},
		types.TaskStateAuthRequired: {
			types.TaskStateSubmitted,
			types.TaskStateWorking,
			types.TaskStateCancelled,
			types.TaskStateRejected,
		},
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedStates {
		if to == allowed {
			return true
		}
	}

	return false
}

// cleanupRoutine periodically cleans up terminated tasks
func (tm *TaskManager) cleanupRoutine() {
	defer close(tm.cleanupDone)

	ticker := time.NewTicker(tm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.performCleanup()
		case <-tm.cleanupStopCh:
			// Perform final cleanup before stopping
			tm.performCleanup()
			return
		}
	}
}

// performCleanup cleans up old terminated tasks
func (tm *TaskManager) performCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), tm.config.DefaultTimeout)
	defer cancel()

	tm.mu.RLock()
	if tm.closing {
		tm.mu.RUnlock()
		return
	}
	tm.mu.RUnlock()

	// Get all terminal tasks
	terminalStates := []types.TaskState{
		types.TaskStateCompleted,
		types.TaskStateFailed,
		types.TaskStateCancelled,
		types.TaskStateRejected,
	}

	tasks, err := tm.store.ListTasksByState(ctx, terminalStates)
	if err != nil {
		tm.logger.Warn(ctx, "Failed to list terminal tasks for cleanup",
			"error", err,
		)
		return
	}

	cutoff := time.Now().Add(-tm.config.TerminatedTaskTTL)

	for _, task := range tasks {
		if task.Status != nil && task.Status.Timestamp.Before(cutoff) {
			// Clean up old task subscriptions
			tm.eventBus.CleanupTask(task.ID)
		}
	}
}

// ProcessMessage processes a message through the handler interface
func (tm *TaskManager) ProcessMessage(ctx context.Context, req *handler.MessageRequest) (*handler.MessageResponse, error) {
	tm.mu.RLock()
	handler := tm.messageHandler
	tm.mu.RUnlock()

	if handler == nil {
		return nil, fmt.Errorf("no message handler configured")
	}

	// Process through user handler
	return handler.HandleMessage(ctx, req)
}

// ProcessCancel processes a cancellation request through the handler interface
func (tm *TaskManager) ProcessCancel(ctx context.Context, taskID string) error {
	tm.mu.RLock()
	handler := tm.messageHandler
	tm.mu.RUnlock()

	if handler == nil {
		return fmt.Errorf("no message handler configured")
	}

	// Process through user handler
	return handler.HandleCancel(ctx, taskID)
}

// GetAgentCard retrieves the agent card using the configured provider
func (tm *TaskManager) GetAgentCard(ctx context.Context) (*types.AgentCard, error) {
	tm.mu.RLock()
	provider := tm.agentCardProvider
	tm.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("no agent card provider configured")
	}

	// Get card through provider
	return provider.GetAgentCard(ctx)
}
