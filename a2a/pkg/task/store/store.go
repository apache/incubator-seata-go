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

	"seata-go-ai-a2a/pkg/types"
)

// TaskStore defines the interface for task persistence operations
// All implementations must be thread-safe and support concurrent access
type TaskStore interface {
	// Task CRUD operations
	CreateTask(ctx context.Context, task *types.Task) error
	GetTask(ctx context.Context, taskID string) (*types.Task, error)
	UpdateTask(ctx context.Context, task *types.Task) error
	DeleteTask(ctx context.Context, taskID string) error

	// Task status operations
	UpdateTaskStatus(ctx context.Context, taskID string, status *types.TaskStatus) error

	// Task artifact operations
	AddArtifact(ctx context.Context, taskID string, artifact *types.Artifact) error
	GetArtifacts(ctx context.Context, taskID string) ([]*types.Artifact, error)

	// Task history operations
	AddMessage(ctx context.Context, taskID string, message *types.Message) error
	GetHistory(ctx context.Context, taskID string, limit int) ([]*types.Message, error)

	// Query operations
	ListTasksByContext(ctx context.Context, contextID string) ([]*types.Task, error)
	ListTasksByState(ctx context.Context, states []types.TaskState) ([]*types.Task, error)

	// Health check and lifecycle
	HealthCheck(ctx context.Context) error
	Close() error
}
