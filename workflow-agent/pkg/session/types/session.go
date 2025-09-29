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
	"encoding/json"
	"time"
)

// Message represents a single message in a conversation
type Message struct {
	ID        string            `json:"id"`
	Role      string            `json:"role"` // user, assistant, system
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// SessionData holds the complete session state
type SessionData struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	Messages     []Message              `json:"messages"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	IsActive     bool                   `json:"is_active"`
	MessageCount int                    `json:"message_count"`
}

// NewMessage creates a new message with current timestamp
func NewMessage(id, role, content string) *Message {
	return &Message{
		ID:        id,
		Role:      role,
		Content:   content,
		Metadata:  make(map[string]string),
		Timestamp: time.Now(),
	}
}

// NewSessionData creates a new session data with initialized fields
func NewSessionData(id, userID string) *SessionData {
	now := time.Now()
	return &SessionData{
		ID:           id,
		UserID:       userID,
		Messages:     make([]Message, 0),
		Context:      make(map[string]interface{}),
		Metadata:     make(map[string]string),
		CreatedAt:    now,
		UpdatedAt:    now,
		IsActive:     true,
		MessageCount: 0,
	}
}

// AddMessage appends a message to the session and updates metadata
func (s *SessionData) AddMessage(msg *Message) {
	s.Messages = append(s.Messages, *msg)
	s.MessageCount = len(s.Messages)
	s.UpdatedAt = time.Now()
}

// SetContext updates the session context
func (s *SessionData) SetContext(key string, value interface{}) {
	if s.Context == nil {
		s.Context = make(map[string]interface{})
	}
	s.Context[key] = value
	s.UpdatedAt = time.Now()
}

// GetContext retrieves a value from session context
func (s *SessionData) GetContext(key string) (interface{}, bool) {
	value, exists := s.Context[key]
	return value, exists
}

// SetMetadata updates session metadata
func (s *SessionData) SetMetadata(key, value string) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	s.Metadata[key] = value
	s.UpdatedAt = time.Now()
}

// IsExpired checks if the session has expired
func (s *SessionData) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// SetTTL sets the session expiration time
func (s *SessionData) SetTTL(ttl time.Duration) {
	expiresAt := time.Now().Add(ttl)
	s.ExpiresAt = &expiresAt
	s.UpdatedAt = time.Now()
}

// GetLastMessage returns the most recent message
func (s *SessionData) GetLastMessage() *Message {
	if len(s.Messages) == 0 {
		return nil
	}
	return &s.Messages[len(s.Messages)-1]
}

// GetMessagesByRole returns all messages with the specified role
func (s *SessionData) GetMessagesByRole(role string) []Message {
	var filtered []Message
	for _, msg := range s.Messages {
		if msg.Role == role {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// ToJSON serializes session data to JSON
func (s *SessionData) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON deserializes session data from JSON
func (s *SessionData) FromJSON(data []byte) error {
	return json.Unmarshal(data, s)
}

// Clone creates a deep copy of the session data
func (s *SessionData) Clone() *SessionData {
	data, _ := s.ToJSON()
	cloned := &SessionData{}
	cloned.FromJSON(data)
	return cloned
}
