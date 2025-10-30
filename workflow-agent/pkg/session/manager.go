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

	"github.com/google/uuid"
)

// SessionStatus represents the status of a session
type SessionStatus string

const (
	StatusPending    SessionStatus = "pending"
	StatusAnalyzing  SessionStatus = "analyzing"
	StatusDiscovering SessionStatus = "discovering"
	StatusGenerating SessionStatus = "generating"
	StatusCompleted  SessionStatus = "completed"
	StatusFailed     SessionStatus = "failed"
)

// ProgressEvent represents a progress update event
type ProgressEvent struct {
	Timestamp    time.Time         `json:"timestamp"`
	Step         int               `json:"step"`
	Action       string            `json:"action"`
	Message      string            `json:"message"`
	Status       SessionStatus     `json:"status"`
	Capabilities interface{}       `json:"capabilities,omitempty"` // []orchestration.DiscoveredCapability
	Workflow     *WorkflowSnapshot `json:"workflow,omitempty"`
}

// WorkflowSnapshot represents a snapshot of the workflow at a point in time
type WorkflowSnapshot struct {
	SeataWorkflow interface{} `json:"seata_workflow,omitempty"` // *orchestration.SeataStateMachine
	ReactFlow     interface{} `json:"react_flow,omitempty"`     // *orchestration.ReactFlowGraph
}

// Session represents an orchestration session
type Session struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Status      SessionStatus          `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Result      interface{}            `json:"result,omitempty"` // *orchestration.OrchestrationResult
	Events      []ProgressEvent        `json:"events"`

	// Internal fields
	mu          sync.RWMutex
	eventChan   chan ProgressEvent
	ctx         context.Context
	cancel      context.CancelFunc
	subscribers map[string]chan ProgressEvent
}

// Manager manages orchestration sessions
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewManager creates a new session manager
func NewManager(ttl time.Duration) *Manager {
	m := &Manager{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go m.cleanupExpiredSessions()

	return m
}

// CreateSession creates a new orchestration session
func (m *Manager) CreateSession(description string, contextData map[string]interface{}) *Session {
	id := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())

	session := &Session{
		ID:          id,
		Description: description,
		Context:     contextData,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Events:      make([]ProgressEvent, 0),
		eventChan:   make(chan ProgressEvent, 100),
		ctx:         ctx,
		cancel:      cancel,
		subscribers: make(map[string]chan ProgressEvent),
	}

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	// Start event dispatcher
	go session.dispatchEvents()

	return session
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, exists := m.sessions[id]
	return session, exists
}

// ListSessions returns all active sessions
func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// DeleteSession removes a session
func (m *Manager) DeleteSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[id]; exists {
		session.Close()
		delete(m.sessions, id)
	}
}

// cleanupExpiredSessions removes old completed sessions
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, session := range m.sessions {
			if session.Status == StatusCompleted || session.Status == StatusFailed {
				if now.Sub(session.UpdatedAt) > m.ttl {
					session.Close()
					delete(m.sessions, id)
				}
			}
		}
		m.mu.Unlock()
	}
}

// AddProgressEvent adds a progress event to the session
func (s *Session) AddProgressEvent(event ProgressEvent) {
	s.mu.Lock()
	event.Timestamp = time.Now()
	s.Events = append(s.Events, event)
	s.Status = event.Status
	s.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Send to event channel (non-blocking)
	select {
	case s.eventChan <- event:
	default:
		// Channel full, skip
	}
}

// Subscribe subscribes to progress events
func (s *Session) Subscribe() (string, chan ProgressEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscriberID := uuid.New().String()
	eventChan := make(chan ProgressEvent, 10)
	s.subscribers[subscriberID] = eventChan

	return subscriberID, eventChan
}

// Unsubscribe removes a subscriber
func (s *Session) Unsubscribe(subscriberID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, exists := s.subscribers[subscriberID]; exists {
		close(ch)
		delete(s.subscribers, subscriberID)
	}
}

// dispatchEvents dispatches events to all subscribers
func (s *Session) dispatchEvents() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.eventChan:
			s.mu.RLock()
			for _, ch := range s.subscribers {
				select {
				case ch <- event:
				default:
					// Subscriber slow, skip
				}
			}
			s.mu.RUnlock()
		}
	}
}

// SetResult sets the final orchestration result
// result should be *orchestration.OrchestrationResult
// Status should be set separately by the caller
func (s *Session) SetResult(result interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Result = result
	s.UpdatedAt = time.Now()
}

// Close closes the session and cleans up resources
func (s *Session) Close() {
	s.cancel()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ch := range s.subscribers {
		close(ch)
	}
	s.subscribers = make(map[string]chan ProgressEvent)
	close(s.eventChan)
}

// GetSnapshot returns a read-only snapshot of the session
func (s *Session) GetSnapshot() SessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SessionSnapshot{
		ID:          s.ID,
		Description: s.Description,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		EventCount:  len(s.Events),
	}
}

// GetCompleteHistory returns complete session data including all events
func (s *Session) GetCompleteHistory() SessionHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of events to avoid race conditions
	events := make([]ProgressEvent, len(s.Events))
	copy(events, s.Events)

	return SessionHistory{
		ID:          s.ID,
		Description: s.Description,
		Context:     s.Context,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		Events:      events,
		Result:      s.Result,
	}
}

// GetResult returns the result (thread-safe)
func (s *Session) GetResult() interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Result
}

// SessionSnapshot represents a lightweight session snapshot
type SessionSnapshot struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	Status      SessionStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	EventCount  int           `json:"event_count"`
}

// SessionHistory represents complete session history
type SessionHistory struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Status      SessionStatus          `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Events      []ProgressEvent        `json:"events"`
	Result      interface{}            `json:"result,omitempty"`
}
