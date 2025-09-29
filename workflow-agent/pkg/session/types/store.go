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
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session expired")
	ErrInvalidSession   = errors.New("invalid session data")
	ErrStoreUnavailable = errors.New("store unavailable")
)

// SessionStore defines the interface for session storage backends
type SessionStore interface {
	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (*SessionData, error)

	// Set stores or updates a session
	Set(ctx context.Context, session *SessionData) error

	// Delete removes a session
	Delete(ctx context.Context, sessionID string) error

	// Exists checks if a session exists
	Exists(ctx context.Context, sessionID string) (bool, error)

	// List returns session IDs for a user with pagination
	List(ctx context.Context, userID string, offset, limit int) ([]string, error)

	// Count returns the total number of sessions for a user
	Count(ctx context.Context, userID string) (int64, error)

	// Cleanup removes expired sessions
	Cleanup(ctx context.Context) (int64, error)

	// Close closes the store and releases resources
	Close() error

	// Health checks store connectivity
	Health(ctx context.Context) error
}

// SessionManager provides high-level session operations
type SessionManager interface {
	// CreateSession creates a new session
	CreateSession(ctx context.Context, userID string, options ...SessionOption) (*SessionData, error)

	// GetSession retrieves a session by ID
	GetSession(ctx context.Context, sessionID string) (*SessionData, error)

	// UpdateSession updates an existing session
	UpdateSession(ctx context.Context, session *SessionData) error

	// DeleteSession removes a session
	DeleteSession(ctx context.Context, sessionID string) error

	// AddMessage adds a message to a session
	AddMessage(ctx context.Context, sessionID string, message *Message) error

	// GetMessages retrieves messages from a session with pagination
	GetMessages(ctx context.Context, sessionID string, offset, limit int) ([]Message, error)

	// ListSessions returns sessions for a user
	ListSessions(ctx context.Context, userID string, offset, limit int) ([]*SessionData, error)

	// SetContext updates session context
	SetContext(ctx context.Context, sessionID, key string, value interface{}) error

	// GetContext retrieves session context value
	GetContext(ctx context.Context, sessionID, key string) (interface{}, error)

	// ExtendTTL extends session expiration time
	ExtendTTL(ctx context.Context, sessionID string, ttl time.Duration) error

	// SessionExists checks if a session exists
	SessionExists(ctx context.Context, sessionID string) (bool, error)

	// Close closes the session manager
	Close() error
}

// SessionOption configures session creation
type SessionOption func(*SessionConfig)

// SessionConfig holds session configuration
type SessionConfig struct {
	TTL         time.Duration
	MaxMessages int
	Metadata    map[string]string
	Context     map[string]interface{}
}

// DefaultSessionConfig returns default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		TTL:         24 * time.Hour,
		MaxMessages: 1000,
		Metadata:    make(map[string]string),
		Context:     make(map[string]interface{}),
	}
}

// WithTTL sets session time-to-live
func WithTTL(ttl time.Duration) SessionOption {
	return func(config *SessionConfig) {
		config.TTL = ttl
	}
}

// WithMaxMessages sets maximum message count
func WithMaxMessages(count int) SessionOption {
	return func(config *SessionConfig) {
		config.MaxMessages = count
	}
}

// WithMetadata sets session metadata
func WithMetadata(metadata map[string]string) SessionOption {
	return func(config *SessionConfig) {
		config.Metadata = metadata
	}
}

// WithContext sets session context
func WithContext(context map[string]interface{}) SessionOption {
	return func(config *SessionConfig) {
		config.Context = context
	}
}

// StoreStats represents store statistics
type StoreStats struct {
	TotalSessions   int64      `json:"total_sessions"`
	ActiveSessions  int64      `json:"active_sessions"`
	ExpiredSessions int64      `json:"expired_sessions"`
	AverageMessages float64    `json:"average_messages"`
	OldestSession   *time.Time `json:"oldest_session,omitempty"`
	NewestSession   *time.Time `json:"newest_session,omitempty"`
	MemoryUsage     int64      `json:"memory_usage,omitempty"`
	LastCleanup     *time.Time `json:"last_cleanup,omitempty"`
}

// StoreInfo provides store information
type StoreInfo struct {
	Type    string        `json:"type"`
	Version string        `json:"version"`
	Status  string        `json:"status"`
	Uptime  time.Duration `json:"uptime"`
	Stats   StoreStats    `json:"stats"`
}
