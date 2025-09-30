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
	"seata-go-ai-workflow-agent/pkg/session/config"
	"seata-go-ai-workflow-agent/pkg/session/types"
	"time"
)

// DefaultManager provides a global session manager instance
var DefaultManager types.SessionManager

// Initialize initializes the default session manager with configuration
func Initialize(cfg config.Config) error {
	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		return err
	}

	DefaultManager = manager
	return nil
}

// CreateSession creates a new session using the default manager
func CreateSession(ctx context.Context, userID string, options ...types.SessionOption) (*types.SessionData, error) {
	if DefaultManager == nil {
		return nil, ErrManagerNotInitialized
	}
	return DefaultManager.CreateSession(ctx, userID, options...)
}

// GetSession retrieves a session by ID using the default manager
func GetSession(ctx context.Context, sessionID string) (*types.SessionData, error) {
	if DefaultManager == nil {
		return nil, ErrManagerNotInitialized
	}
	return DefaultManager.GetSession(ctx, sessionID)
}

// UpdateSession updates an existing session using the default manager
func UpdateSession(ctx context.Context, session *types.SessionData) error {
	if DefaultManager == nil {
		return ErrManagerNotInitialized
	}
	return DefaultManager.UpdateSession(ctx, session)
}

// DeleteSession removes a session using the default manager
func DeleteSession(ctx context.Context, sessionID string) error {
	if DefaultManager == nil {
		return ErrManagerNotInitialized
	}
	return DefaultManager.DeleteSession(ctx, sessionID)
}

// AddMessage adds a message to a session using the default manager
func AddMessage(ctx context.Context, sessionID string, message *types.Message) error {
	if DefaultManager == nil {
		return ErrManagerNotInitialized
	}
	return DefaultManager.AddMessage(ctx, sessionID, message)
}

// GetMessages retrieves messages from a session using the default manager
func GetMessages(ctx context.Context, sessionID string, offset, limit int) ([]types.Message, error) {
	if DefaultManager == nil {
		return nil, ErrManagerNotInitialized
	}
	return DefaultManager.GetMessages(ctx, sessionID, offset, limit)
}

// ListSessions returns sessions for a user using the default manager
func ListSessions(ctx context.Context, userID string, offset, limit int) ([]*types.SessionData, error) {
	if DefaultManager == nil {
		return nil, ErrManagerNotInitialized
	}
	return DefaultManager.ListSessions(ctx, userID, offset, limit)
}

// SetContext updates session context using the default manager
func SetContext(ctx context.Context, sessionID, key string, value interface{}) error {
	if DefaultManager == nil {
		return ErrManagerNotInitialized
	}
	return DefaultManager.SetContext(ctx, sessionID, key, value)
}

// GetContext retrieves session context value using the default manager
func GetContext(ctx context.Context, sessionID, key string) (interface{}, error) {
	if DefaultManager == nil {
		return nil, ErrManagerNotInitialized
	}
	return DefaultManager.GetContext(ctx, sessionID, key)
}

// ExtendTTL extends session expiration time using the default manager
func ExtendTTL(ctx context.Context, sessionID string, ttl time.Duration) error {
	if DefaultManager == nil {
		return ErrManagerNotInitialized
	}
	return DefaultManager.ExtendTTL(ctx, sessionID, ttl)
}

// Close closes the default session manager
func Close() error {
	if DefaultManager == nil {
		return ErrManagerNotInitialized
	}
	return DefaultManager.Close()
}
