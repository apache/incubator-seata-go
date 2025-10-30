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
)

// Storage defines the interface for session persistence
type Storage interface {
	// Save persists a session
	Save(ctx context.Context, session *Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Delete removes a session
	Delete(ctx context.Context, sessionID string) error

	// List returns all sessions for a user
	List(ctx context.Context, userID string) ([]*Session, error)

	// ListByAgent returns all sessions for an agent
	ListByAgent(ctx context.Context, agentID string) ([]*Session, error)

	// DeleteExpired removes all expired sessions
	DeleteExpired(ctx context.Context) error

	// Exists checks if a session exists
	Exists(ctx context.Context, sessionID string) (bool, error)
}
