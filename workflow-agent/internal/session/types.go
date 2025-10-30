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
	"time"
)

// Message represents a single message in a conversation
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Thread represents a conversation thread within a session
type Thread struct {
	ID        string    `json:"id"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SessionState represents custom state data for a session
type SessionState map[string]interface{}

// Session represents a complete agent session with multiple threads
type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	AgentID   string                 `json:"agent_id"`
	Threads   map[string]*Thread     `json:"threads"`
	State     SessionState           `json:"state"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt time.Time              `json:"expires_at"`
}

// NewSession creates a new session with the given parameters
func NewSession(id, userID, agentID string, ttl time.Duration) *Session {
	now := time.Now()
	return &Session{
		ID:        id,
		UserID:    userID,
		AgentID:   agentID,
		Threads:   make(map[string]*Thread),
		State:     make(SessionState),
		Metadata:  make(map[string]interface{}),
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
}

// NewThread creates a new thread within a session
func NewThread(id string) *Thread {
	now := time.Now()
	return &Thread{
		ID:        id,
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewMessage creates a new message
func NewMessage(role, content string) Message {
	return Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// AddMessage adds a message to the thread
func (t *Thread) AddMessage(msg Message) {
	t.Messages = append(t.Messages, msg)
	t.UpdatedAt = time.Now()
}

// GetMessages returns all messages in the thread
func (t *Thread) GetMessages() []Message {
	return t.Messages
}

// GetLastMessage returns the last message in the thread
func (t *Thread) GetLastMessage() *Message {
	if len(t.Messages) == 0 {
		return nil
	}
	return &t.Messages[len(t.Messages)-1]
}

// MessageCount returns the number of messages in the thread
func (t *Thread) MessageCount() int {
	return len(t.Messages)
}

// AddThread adds a new thread to the session
func (s *Session) AddThread(thread *Thread) {
	s.Threads[thread.ID] = thread
	s.UpdatedAt = time.Now()
}

// GetThread retrieves a thread by ID
func (s *Session) GetThread(threadID string) *Thread {
	return s.Threads[threadID]
}

// GetOrCreateThread gets an existing thread or creates a new one
func (s *Session) GetOrCreateThread(threadID string) *Thread {
	if thread, exists := s.Threads[threadID]; exists {
		return thread
	}
	thread := NewThread(threadID)
	s.AddThread(thread)
	return thread
}

// SetState sets a state value
func (s *Session) SetState(key string, value interface{}) {
	s.State[key] = value
	s.UpdatedAt = time.Now()
}

// GetState gets a state value
func (s *Session) GetState(key string) (interface{}, bool) {
	val, ok := s.State[key]
	return val, ok
}

// DeleteState removes a state value
func (s *Session) DeleteState(key string) {
	delete(s.State, key)
	s.UpdatedAt = time.Now()
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// ExtendExpiry extends the session expiry time
func (s *Session) ExtendExpiry(ttl time.Duration) {
	s.ExpiresAt = time.Now().Add(ttl)
	s.UpdatedAt = time.Now()
}

// ThreadCount returns the number of threads in the session
func (s *Session) ThreadCount() int {
	return len(s.Threads)
}
