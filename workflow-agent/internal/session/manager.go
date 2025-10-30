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
	"time"
)

import (
	"github.com/google/uuid"
)

import (
	"seata-go-ai-workflow-agent/pkg/errors"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// Manager manages agent sessions with genkit integration
type Manager struct {
	storage         Storage
	defaultTTL      time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// Config holds session manager configuration
type Config struct {
	Storage         Storage
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
	AutoCleanup     bool
}

// DefaultConfig returns default session manager configuration
func DefaultConfig() *Config {
	return &Config{
		Storage:         NewMemoryStorage(),
		DefaultTTL:      24 * time.Hour,
		CleanupInterval: 1 * time.Hour,
		AutoCleanup:     true,
	}
}

// NewManager creates a new session manager
func NewManager(cfg *Config) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if cfg.Storage == nil {
		cfg.Storage = NewMemoryStorage()
	}

	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 24 * time.Hour
	}

	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = 1 * time.Hour
	}

	m := &Manager{
		storage:         cfg.Storage,
		defaultTTL:      cfg.DefaultTTL,
		cleanupInterval: cfg.CleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	if cfg.AutoCleanup {
		go m.startAutoCleanup()
	}

	logger.WithFields(map[string]interface{}{
		"defaultTTL":      cfg.DefaultTTL,
		"cleanupInterval": cfg.CleanupInterval,
		"autoCleanup":     cfg.AutoCleanup,
	}).Info("session manager initialized")

	return m
}

// CreateSession creates a new session
func (m *Manager) CreateSession(ctx context.Context, userID, agentID string) (*Session, error) {
	return m.CreateSessionWithTTL(ctx, userID, agentID, m.defaultTTL)
}

// CreateSessionWithTTL creates a new session with custom TTL
func (m *Manager) CreateSessionWithTTL(ctx context.Context, userID, agentID string, ttl time.Duration) (*Session, error) {
	if userID == "" {
		return nil, errors.InvalidArgumentError("userID cannot be empty")
	}

	if agentID == "" {
		return nil, errors.InvalidArgumentError("agentID cannot be empty")
	}

	sessionID := uuid.New().String()
	session := NewSession(sessionID, userID, agentID, ttl)

	if err := m.storage.Save(ctx, session); err != nil {
		logger.WithField("error", err).Error("failed to create session")
		return nil, errors.Wrap(errors.ErrUnknown, "failed to create session", err)
	}

	logger.WithFields(map[string]interface{}{
		"sessionID": sessionID,
		"userID":    userID,
		"agentID":   agentID,
		"ttl":       ttl,
	}).Info("session created")

	return session, nil
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	session, err := m.storage.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// UpdateSession updates an existing session
func (m *Manager) UpdateSession(ctx context.Context, session *Session) error {
	if session == nil {
		return errors.InvalidArgumentError("session cannot be nil")
	}

	exists, err := m.storage.Exists(ctx, session.ID)
	if err != nil {
		return err
	}

	if !exists {
		return errors.NotFoundErrorf("session not found: %s", session.ID)
	}

	session.UpdatedAt = time.Now()

	if err := m.storage.Save(ctx, session); err != nil {
		logger.WithField("error", err).Error("failed to update session")
		return errors.Wrap(errors.ErrUnknown, "failed to update session", err)
	}

	logger.WithField("sessionID", session.ID).Debug("session updated")
	return nil
}

// DeleteSession removes a session
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	if err := m.storage.Delete(ctx, sessionID); err != nil {
		return err
	}

	logger.WithField("sessionID", sessionID).Info("session deleted")
	return nil
}

// AddMessage adds a message to a session thread
func (m *Manager) AddMessage(ctx context.Context, sessionID, threadID, role, content string) error {
	session, err := m.storage.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	thread := session.GetOrCreateThread(threadID)
	msg := NewMessage(role, content)
	thread.AddMessage(msg)

	if err := m.storage.Save(ctx, session); err != nil {
		logger.WithField("error", err).Error("failed to save message")
		return errors.Wrap(errors.ErrUnknown, "failed to save message", err)
	}

	logger.WithFields(map[string]interface{}{
		"sessionID": sessionID,
		"threadID":  threadID,
		"role":      role,
	}).Debug("message added")

	return nil
}

// GetMessages retrieves all messages from a thread
func (m *Manager) GetMessages(ctx context.Context, sessionID, threadID string) ([]Message, error) {
	session, err := m.storage.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	thread := session.GetThread(threadID)
	if thread == nil {
		return []Message{}, nil
	}

	return thread.GetMessages(), nil
}

// SetState sets a state value in the session
func (m *Manager) SetState(ctx context.Context, sessionID, key string, value interface{}) error {
	session, err := m.storage.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.SetState(key, value)

	if err := m.storage.Save(ctx, session); err != nil {
		logger.WithField("error", err).Error("failed to save state")
		return errors.Wrap(errors.ErrUnknown, "failed to save state", err)
	}

	logger.WithFields(map[string]interface{}{
		"sessionID": sessionID,
		"key":       key,
	}).Debug("state set")

	return nil
}

// GetState retrieves a state value from the session
func (m *Manager) GetState(ctx context.Context, sessionID, key string) (interface{}, error) {
	session, err := m.storage.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	val, exists := session.GetState(key)
	if !exists {
		return nil, errors.NotFoundErrorf("state key not found: %s", key)
	}

	return val, nil
}

// ListUserSessions returns all sessions for a user
func (m *Manager) ListUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	sessions, err := m.storage.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

// ListAgentSessions returns all sessions for an agent
func (m *Manager) ListAgentSessions(ctx context.Context, agentID string) ([]*Session, error) {
	sessions, err := m.storage.ListByAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

// ExtendSession extends the session expiry time
func (m *Manager) ExtendSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	session, err := m.storage.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.ExtendExpiry(ttl)

	if err := m.storage.Save(ctx, session); err != nil {
		logger.WithField("error", err).Error("failed to extend session")
		return errors.Wrap(errors.ErrUnknown, "failed to extend session", err)
	}

	logger.WithFields(map[string]interface{}{
		"sessionID": sessionID,
		"ttl":       ttl,
	}).Debug("session extended")

	return nil
}

// CleanupExpired removes all expired sessions
func (m *Manager) CleanupExpired(ctx context.Context) error {
	if err := m.storage.DeleteExpired(ctx); err != nil {
		logger.WithField("error", err).Error("failed to cleanup expired sessions")
		return err
	}

	return nil
}

// startAutoCleanup starts automatic cleanup of expired sessions
func (m *Manager) startAutoCleanup() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	logger.WithField("interval", m.cleanupInterval).Info("auto cleanup started")

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			if err := m.CleanupExpired(ctx); err != nil {
				logger.WithField("error", err).Warn("auto cleanup failed")
			}
		case <-m.stopCleanup:
			logger.Info("auto cleanup stopped")
			return
		}
	}
}

// Stop stops the session manager and cleanup goroutine
func (m *Manager) Stop() {
	close(m.stopCleanup)
	logger.Info("session manager stopped")
}
