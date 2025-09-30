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

package session

import (
	"context"
	"fmt"
	"seata-go-ai-workflow-agent/pkg/session/config"
	"seata-go-ai-workflow-agent/pkg/session/store"
	"seata-go-ai-workflow-agent/pkg/session/types"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager implements the SessionManager interface
type Manager struct {
	store         types.SessionStore
	config        config.SessionConfig
	cleanupTicker *time.Ticker
	cleanupStop   chan struct{}
	mu            sync.RWMutex
}

// NewManager creates a new session manager with the specified store
func NewManager(store types.SessionStore, cfg config.SessionConfig) *Manager {
	manager := &Manager{
		store:       store,
		config:      cfg,
		cleanupStop: make(chan struct{}),
	}

	return manager
}

// NewManagerFromConfig creates a new session manager from configuration
func NewManagerFromConfig(cfg config.Config) (*Manager, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create store based on type
	var sessionStore types.SessionStore
	var err error

	switch cfg.Store.Type {
	case config.StoreTypeMemory:
		sessionStore = store.NewMemoryStore(cfg.Store.Memory)
	case config.StoreTypeRedis:
		sessionStore, err = store.NewRedisStore(cfg.Store.Redis)
		if err != nil {
			return nil, fmt.Errorf("failed to create Redis store: %w", err)
		}
	case config.StoreTypeMySQL:
		sessionStore, err = store.NewMySQLStore(cfg.Store.MySQL)
		if err != nil {
			return nil, fmt.Errorf("failed to create MySQL store: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported store type: %s", cfg.Store.Type)
	}

	manager := NewManager(sessionStore, cfg.Session)

	// Start cleanup routine if enabled
	if cfg.Cleanup.Enabled && cfg.Cleanup.Interval > 0 {
		manager.StartCleanup(cfg.Cleanup.Interval)
	}

	return manager, nil
}

// CreateSession creates a new session
func (m *Manager) CreateSession(ctx context.Context, userID string, options ...types.SessionOption) (*types.SessionData, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Apply options to default config
	sessionConfig := types.DefaultSessionConfig()
	for _, option := range options {
		option(sessionConfig)
	}

	// Override with manager defaults if not set
	if sessionConfig.TTL == 0 {
		sessionConfig.TTL = m.config.DefaultTTL
	}
	if sessionConfig.MaxMessages == 0 {
		sessionConfig.MaxMessages = m.config.MaxMessageCount
	}

	// Generate session ID
	sessionID := generateSessionID()

	// Create session data
	session := types.NewSessionData(sessionID, userID)
	session.Metadata = sessionConfig.Metadata
	session.Context = sessionConfig.Context

	// Set TTL if specified
	if sessionConfig.TTL > 0 {
		session.SetTTL(sessionConfig.TTL)
	}

	// Store session
	if err := m.store.Set(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(ctx context.Context, sessionID string) (*types.SessionData, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	session, err := m.store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// UpdateSession updates an existing session
func (m *Manager) UpdateSession(ctx context.Context, session *types.SessionData) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	// Validate message count limit
	if len(session.Messages) > m.config.MaxMessageCount {
		return fmt.Errorf("session exceeds maximum message count: %d", m.config.MaxMessageCount)
	}

	return m.store.Set(ctx, session)
}

// DeleteSession removes a session
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	return m.store.Delete(ctx, sessionID)
}

// AddMessage adds a message to a session
func (m *Manager) AddMessage(ctx context.Context, sessionID string, message *types.Message) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Get current session
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Check message count limit
	if len(session.Messages) >= m.config.MaxMessageCount {
		return fmt.Errorf("session message limit reached: %d", m.config.MaxMessageCount)
	}

	// Generate message ID if not set
	if message.ID == "" {
		message.ID = generateMessageID()
	}

	// Set timestamp if not set
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// Add message to session
	session.AddMessage(message)

	// Update session
	return m.UpdateSession(ctx, session)
}

// GetMessages retrieves messages from a session with pagination
func (m *Manager) GetMessages(ctx context.Context, sessionID string, offset, limit int) ([]types.Message, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	messages := session.Messages

	// Apply pagination
	end := offset + limit
	if offset > len(messages) {
		return []types.Message{}, nil
	}
	if end > len(messages) {
		end = len(messages)
	}

	return messages[offset:end], nil
}

// ListSessions returns sessions for a user
func (m *Manager) ListSessions(ctx context.Context, userID string, offset, limit int) ([]*types.SessionData, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	sessionIDs, err := m.store.List(ctx, userID, offset, limit)
	if err != nil {
		return nil, err
	}

	var sessions []*types.SessionData
	for _, sessionID := range sessionIDs {
		session, err := m.store.Get(ctx, sessionID)
		if err != nil {
			// Skip sessions that cannot be retrieved (might be expired)
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// SetContext updates session context
func (m *Manager) SetContext(ctx context.Context, sessionID, key string, value interface{}) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if key == "" {
		return fmt.Errorf("context key cannot be empty")
	}

	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.SetContext(key, value)
	return m.UpdateSession(ctx, session)
}

// GetContext retrieves session context value
func (m *Manager) GetContext(ctx context.Context, sessionID, key string) (interface{}, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}
	if key == "" {
		return nil, fmt.Errorf("context key cannot be empty")
	}

	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	value, exists := session.GetContext(key)
	if !exists {
		return nil, fmt.Errorf("context key not found: %s", key)
	}

	return value, nil
}

// ExtendTTL extends session expiration time
func (m *Manager) ExtendTTL(ctx context.Context, sessionID string, ttl time.Duration) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if ttl <= 0 {
		return fmt.Errorf("TTL must be positive")
	}

	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.SetTTL(ttl)
	return m.UpdateSession(ctx, session)
}

// SessionExists checks if a session exists
func (m *Manager) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	if sessionID == "" {
		return false, fmt.Errorf("session ID cannot be empty")
	}

	return m.store.Exists(ctx, sessionID)
}

// CountUserSessions returns the number of sessions for a user
func (m *Manager) CountUserSessions(ctx context.Context, userID string) (int64, error) {
	if userID == "" {
		return 0, fmt.Errorf("user ID cannot be empty")
	}

	return m.store.Count(ctx, userID)
}

// StartCleanup starts the automatic cleanup routine
func (m *Manager) StartCleanup(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}

	m.cleanupTicker = time.NewTicker(interval)
	go m.cleanupWorker()
}

// StopCleanup stops the automatic cleanup routine
func (m *Manager) StopCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
		m.cleanupTicker = nil
	}

	close(m.cleanupStop)
	m.cleanupStop = make(chan struct{})
}

// Cleanup manually triggers session cleanup
func (m *Manager) Cleanup(ctx context.Context) (int64, error) {
	return m.store.Cleanup(ctx)
}

// Close closes the session manager and releases resources
func (m *Manager) Close() error {
	m.StopCleanup()
	return m.store.Close()
}

// Health checks the health of the session store
func (m *Manager) Health(ctx context.Context) error {
	return m.store.Health(ctx)
}

// cleanupWorker runs the cleanup routine
func (m *Manager) cleanupWorker() {
	if m.cleanupTicker == nil {
		return
	}

	for {
		select {
		case <-m.cleanupTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			m.store.Cleanup(ctx)
			cancel()
		case <-m.cleanupStop:
			return
		}
	}
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return "sess_" + uuid.New().String()
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return "msg_" + uuid.New().String()
}
