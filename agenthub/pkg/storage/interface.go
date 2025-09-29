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

package storage

import (
	"context"
	"time"
)

// Storage defines the generic storage interface following K8s patterns
type Storage interface {
	// Create creates a new resource
	Create(ctx context.Context, resource Resource) error

	// Get retrieves a resource by ID
	Get(ctx context.Context, id string) (Resource, error)

	// Update updates an existing resource
	Update(ctx context.Context, resource Resource) error

	// Delete deletes a resource by ID
	Delete(ctx context.Context, id string) error

	// List lists resources with optional filters
	List(ctx context.Context, filters map[string]interface{}) ([]Resource, error)

	// Watch watches for changes to resources
	Watch(ctx context.Context, filters map[string]interface{}) (<-chan Event, error)

	// Close closes the storage connection
	Close() error
}

// Resource represents a generic storage resource
type Resource interface {
	GetID() string
	GetKind() string
	GetVersion() string
	GetMetadata() map[string]interface{}
	SetMetadata(key string, value interface{})
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// Event represents a storage event
type Event struct {
	Type     EventType `json:"type"`
	Resource Resource  `json:"resource"`
	Time     time.Time `json:"time"`
}

// EventType represents the type of storage event
type EventType string

const (
	EventTypeCreate EventType = "CREATE"
	EventTypeUpdate EventType = "UPDATE"
	EventTypeDelete EventType = "DELETE"
)

// HealthChecker provides health check functionality for storage
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// Migrator provides database migration functionality
type Migrator interface {
	Migrate(ctx context.Context) error
	Rollback(ctx context.Context, version string) error
}

// TransactionManager provides transaction support
type TransactionManager interface {
	BeginTransaction(ctx context.Context) (Transaction, error)
}

// Transaction represents a storage transaction
type Transaction interface {
	Commit() error
	Rollback() error
	Storage() Storage
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type       string                 `json:"type"`
	Connection string                 `json:"connection"`
	Options    map[string]interface{} `json:"options"`
}

// Factory creates storage instances
type Factory interface {
	Create(config StorageConfig) (Storage, error)
	GetSupportedTypes() []string
}
