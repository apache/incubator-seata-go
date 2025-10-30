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
	"sync"
	"time"
)

import (
	"seata-go-ai-workflow-agent/pkg/errors"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// MemoryStorage provides in-memory session storage
type MemoryStorage struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		sessions: make(map[string]*Session),
	}
}

// Save persists a session to memory
func (m *MemoryStorage) Save(ctx context.Context, session *Session) error {
	if session == nil {
		return errors.InvalidArgumentError("session cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Deep copy the session to avoid external modifications
	sessionCopy := m.copySession(session)
	m.sessions[session.ID] = sessionCopy

	logger.WithFields(map[string]interface{}{
		"sessionID": session.ID,
		"userID":    session.UserID,
		"threads":   len(session.Threads),
	}).Debug("session saved to memory")

	return nil
}

// Get retrieves a session from memory by ID
func (m *MemoryStorage) Get(ctx context.Context, sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, errors.NotFoundErrorf("session not found: %s", sessionID)
	}

	if session.IsExpired() {
		logger.WithField("sessionID", sessionID).Debug("session expired")
		return nil, errors.NotFoundErrorf("session expired: %s", sessionID)
	}

	// Return a copy to prevent external modifications
	return m.copySession(session), nil
}

// Delete removes a session from memory
func (m *MemoryStorage) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		return errors.NotFoundErrorf("session not found: %s", sessionID)
	}

	delete(m.sessions, sessionID)

	logger.WithField("sessionID", sessionID).Debug("session deleted from memory")
	return nil
}

// List returns all sessions for a user
func (m *MemoryStorage) List(ctx context.Context, userID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*Session
	for _, session := range m.sessions {
		if session.UserID == userID && !session.IsExpired() {
			sessions = append(sessions, m.copySession(session))
		}
	}

	logger.WithFields(map[string]interface{}{
		"userID": userID,
		"count":  len(sessions),
	}).Debug("sessions listed for user")

	return sessions, nil
}

// ListByAgent returns all sessions for an agent
func (m *MemoryStorage) ListByAgent(ctx context.Context, agentID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*Session
	for _, session := range m.sessions {
		if session.AgentID == agentID && !session.IsExpired() {
			sessions = append(sessions, m.copySession(session))
		}
	}

	logger.WithFields(map[string]interface{}{
		"agentID": agentID,
		"count":   len(sessions),
	}).Debug("sessions listed for agent")

	return sessions, nil
}

// DeleteExpired removes all expired sessions
func (m *MemoryStorage) DeleteExpired(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	now := time.Now()

	for id, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, id)
			count++
		}
	}

	if count > 0 {
		logger.WithField("count", count).Info("expired sessions deleted")
	}

	return nil
}

// Exists checks if a session exists in memory
func (m *MemoryStorage) Exists(ctx context.Context, sessionID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return false, nil
	}

	return !session.IsExpired(), nil
}

// copySession creates a deep copy of a session
func (m *MemoryStorage) copySession(src *Session) *Session {
	if src == nil {
		return nil
	}

	dst := &Session{
		ID:        src.ID,
		UserID:    src.UserID,
		AgentID:   src.AgentID,
		Threads:   make(map[string]*Thread),
		State:     make(SessionState),
		Metadata:  make(map[string]interface{}),
		CreatedAt: src.CreatedAt,
		UpdatedAt: src.UpdatedAt,
		ExpiresAt: src.ExpiresAt,
	}

	// Copy threads
	for id, thread := range src.Threads {
		threadCopy := &Thread{
			ID:        thread.ID,
			Messages:  make([]Message, len(thread.Messages)),
			CreatedAt: thread.CreatedAt,
			UpdatedAt: thread.UpdatedAt,
		}
		copy(threadCopy.Messages, thread.Messages)
		dst.Threads[id] = threadCopy
	}

	// Copy state
	for k, v := range src.State {
		dst.State[k] = v
	}

	// Copy metadata
	for k, v := range src.Metadata {
		dst.Metadata[k] = v
	}

	return dst
}

// Count returns the total number of sessions in storage
func (m *MemoryStorage) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// Clear removes all sessions from storage
func (m *MemoryStorage) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]*Session)
	logger.Info("all sessions cleared from memory")
}
