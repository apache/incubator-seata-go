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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"seata-go-ai-a2a/pkg/types"
)

// RedisTaskStore implements TaskStore interface using Redis storage
// This implementation supports distributed deployments and persistence
type RedisTaskStore struct {
	client redis.Cmdable
	prefix string
	ttl    time.Duration
}

// RedisTaskStoreConfig configures the Redis task store
type RedisTaskStoreConfig struct {
	// Redis client (can be *redis.Client or *redis.ClusterClient)
	Client redis.Cmdable

	// Key prefix for Redis keys (default: "a2a:tasks:")
	KeyPrefix string

	// TTL for terminal tasks (default: 24 hours)
	TerminalTaskTTL time.Duration
}

// NewRedisTaskStore creates a new Redis-backed task store
func NewRedisTaskStore(config RedisTaskStoreConfig) *RedisTaskStore {
	prefix := config.KeyPrefix
	if prefix == "" {
		prefix = "a2a:tasks:"
	}

	ttl := config.TerminalTaskTTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	return &RedisTaskStore{
		client: config.Client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// Redis key patterns
const (
	keyPatternTask      = "%stask:%s"           // a2a:tasks:task:{taskID}
	keyPatternArtifacts = "%stask:%s:artifacts" // a2a:tasks:task:{taskID}:artifacts
	keyPatternHistory   = "%stask:%s:history"   // a2a:tasks:task:{taskID}:history
	keyPatternByContext = "%scontext:%s"        // a2a:tasks:context:{contextID}
	keyPatternByState   = "%sstate:%s"          // a2a:tasks:state:{state}
)

// CreateTask stores a new task in Redis
func (r *RedisTaskStore) CreateTask(ctx context.Context, task *types.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	taskKey := fmt.Sprintf(keyPatternTask, r.prefix, task.ID)

	// Check if task already exists
	exists, err := r.client.Exists(ctx, taskKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check task existence: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// Serialize task to JSON
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Store task data
	pipe.Set(ctx, taskKey, taskData, 0)

	// Add to context index
	if task.ContextID != "" {
		contextKey := fmt.Sprintf(keyPatternByContext, r.prefix, task.ContextID)
		pipe.SAdd(ctx, contextKey, task.ID)
	}

	// Add to state index
	if task.Status != nil {
		stateKey := fmt.Sprintf(keyPatternByState, r.prefix, task.Status.State.String())
		pipe.SAdd(ctx, stateKey, task.ID)
	}

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID from Redis
func (r *RedisTaskStore) GetTask(ctx context.Context, taskID string) (*types.Task, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	taskKey := fmt.Sprintf(keyPatternTask, r.prefix, taskID)

	taskData, err := r.client.Get(ctx, taskKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("task with ID %s not found", taskID)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task types.Task
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// UpdateTask updates an existing task in Redis
func (r *RedisTaskStore) UpdateTask(ctx context.Context, task *types.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	taskKey := fmt.Sprintf(keyPatternTask, r.prefix, task.ID)

	// Check if task exists
	exists, err := r.client.Exists(ctx, taskKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check task existence: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("task with ID %s not found", task.ID)
	}

	// Get current task to update indexes
	currentTask, err := r.GetTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get current task: %w", err)
	}

	// Serialize updated task
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	pipe := r.client.Pipeline()

	// Update task data
	var ttl time.Duration
	if task.Status != nil && task.Status.State.IsTerminal() {
		ttl = r.ttl
	}
	pipe.Set(ctx, taskKey, taskData, ttl)

	// Update state index if state changed
	if currentTask.Status != nil && task.Status != nil &&
		currentTask.Status.State != task.Status.State {

		// Remove from old state index
		oldStateKey := fmt.Sprintf(keyPatternByState, r.prefix, currentTask.Status.State.String())
		pipe.SRem(ctx, oldStateKey, task.ID)

		// Add to new state index
		newStateKey := fmt.Sprintf(keyPatternByState, r.prefix, task.Status.State.String())
		pipe.SAdd(ctx, newStateKey, task.ID)
	}

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// DeleteTask removes a task from Redis
func (r *RedisTaskStore) DeleteTask(ctx context.Context, taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	// Get task first to clean up indexes
	task, err := r.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	pipe := r.client.Pipeline()

	// Delete main task key
	taskKey := fmt.Sprintf(keyPatternTask, r.prefix, taskID)
	pipe.Del(ctx, taskKey)

	// Delete artifacts
	artifactsKey := fmt.Sprintf(keyPatternArtifacts, r.prefix, taskID)
	pipe.Del(ctx, artifactsKey)

	// Delete history
	historyKey := fmt.Sprintf(keyPatternHistory, r.prefix, taskID)
	pipe.Del(ctx, historyKey)

	// Remove from context index
	if task.ContextID != "" {
		contextKey := fmt.Sprintf(keyPatternByContext, r.prefix, task.ContextID)
		pipe.SRem(ctx, contextKey, taskID)
	}

	// Remove from state index
	if task.Status != nil {
		stateKey := fmt.Sprintf(keyPatternByState, r.prefix, task.Status.State.String())
		pipe.SRem(ctx, stateKey, taskID)
	}

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// UpdateTaskStatus updates the status of an existing task
func (r *RedisTaskStore) UpdateTaskStatus(ctx context.Context, taskID string, status *types.TaskStatus) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if status == nil {
		return fmt.Errorf("status cannot be nil")
	}

	// Get current task
	task, err := r.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// Update status
	task.Status = status

	// Update the task
	return r.UpdateTask(ctx, task)
}

// AddArtifact adds an artifact to a task
func (r *RedisTaskStore) AddArtifact(ctx context.Context, taskID string, artifact *types.Artifact) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if artifact == nil {
		return fmt.Errorf("artifact cannot be nil")
	}

	// Get current task
	task, err := r.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// Add artifact to task
	task.Artifacts = append(task.Artifacts, artifact)

	// Update the task
	return r.UpdateTask(ctx, task)
}

// GetArtifacts retrieves all artifacts for a task
func (r *RedisTaskStore) GetArtifacts(ctx context.Context, taskID string) ([]*types.Artifact, error) {
	task, err := r.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return task.Artifacts, nil
}

// AddMessage adds a message to task history
func (r *RedisTaskStore) AddMessage(ctx context.Context, taskID string, message *types.Message) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Get current task
	task, err := r.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// Add message to history
	task.History = append(task.History, message)

	// Update the task
	return r.UpdateTask(ctx, task)
}

// GetHistory retrieves task history with optional limit
func (r *RedisTaskStore) GetHistory(ctx context.Context, taskID string, limit int) ([]*types.Message, error) {
	task, err := r.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	history := task.History
	if limit > 0 && len(history) > limit {
		// Return the most recent messages
		history = history[len(history)-limit:]
	}

	return history, nil
}

// ListTasksByContext lists all tasks for a specific context
func (r *RedisTaskStore) ListTasksByContext(ctx context.Context, contextID string) ([]*types.Task, error) {
	if contextID == "" {
		return nil, fmt.Errorf("context ID cannot be empty")
	}

	contextKey := fmt.Sprintf(keyPatternByContext, r.prefix, contextID)
	taskIDs, err := r.client.SMembers(ctx, contextKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get task IDs for context: %w", err)
	}

	return r.getTasksByIDs(ctx, taskIDs)
}

// ListTasksByState lists all tasks with specified states
func (r *RedisTaskStore) ListTasksByState(ctx context.Context, states []types.TaskState) ([]*types.Task, error) {
	if len(states) == 0 {
		return nil, fmt.Errorf("at least one state must be specified")
	}

	var allTaskIDs []string

	for _, state := range states {
		stateKey := fmt.Sprintf(keyPatternByState, r.prefix, state.String())
		taskIDs, err := r.client.SMembers(ctx, stateKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get task IDs for state %s: %w", state.String(), err)
		}
		allTaskIDs = append(allTaskIDs, taskIDs...)
	}

	// Remove duplicates
	taskIDSet := make(map[string]bool)
	var uniqueTaskIDs []string
	for _, taskID := range allTaskIDs {
		if !taskIDSet[taskID] {
			taskIDSet[taskID] = true
			uniqueTaskIDs = append(uniqueTaskIDs, taskID)
		}
	}

	return r.getTasksByIDs(ctx, uniqueTaskIDs)
}

// HealthCheck checks the health of Redis connection
func (r *RedisTaskStore) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close cleans up Redis connections
func (r *RedisTaskStore) Close() error {
	// If the client is a *redis.Client, close it
	if client, ok := r.client.(*redis.Client); ok {
		return client.Close()
	}
	// For cluster clients or other implementations, no-op
	return nil
}

// Helper method to get multiple tasks by IDs
func (r *RedisTaskStore) getTasksByIDs(ctx context.Context, taskIDs []string) ([]*types.Task, error) {
	if len(taskIDs) == 0 {
		return []*types.Task{}, nil
	}

	// Build keys for pipeline
	keys := make([]string, len(taskIDs))
	for i, taskID := range taskIDs {
		keys[i] = fmt.Sprintf(keyPatternTask, r.prefix, taskID)
	}

	// Use pipeline for batch get
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Parse results
	var tasks []*types.Task
	for i, cmd := range cmds {
		result, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				// Task not found, skip
				continue
			}
			return nil, fmt.Errorf("failed to get task %s: %w", taskIDs[i], err)
		}

		var task types.Task
		if err := json.Unmarshal([]byte(result), &task); err != nil {
			return nil, fmt.Errorf("failed to unmarshal task %s: %w", taskIDs[i], err)
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}
