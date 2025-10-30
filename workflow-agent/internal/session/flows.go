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

import (
	"github.com/firebase/genkit/go/genkit"
)

import (
	"seata-go-ai-workflow-agent/pkg/errors"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// ConversationRequest represents a conversation request
type ConversationRequest struct {
	SessionID string `json:"session_id,omitempty"`
	ThreadID  string `json:"thread_id"`
	UserID    string `json:"user_id"`
	AgentID   string `json:"agent_id"`
	Message   string `json:"message"`
}

// ConversationResponse represents a conversation response
type ConversationResponse struct {
	SessionID string    `json:"session_id"`
	ThreadID  string    `json:"thread_id"`
	Messages  []Message `json:"messages"`
	Response  string    `json:"response,omitempty"`
}

// SessionFlows wraps session manager with genkit flows
type SessionFlows struct {
	manager *Manager
	g       *genkit.Genkit
}

// NewSessionFlows creates a new session flows instance
func NewSessionFlows(manager *Manager, g *genkit.Genkit) *SessionFlows {
	sf := &SessionFlows{
		manager: manager,
		g:       g,
	}

	sf.registerFlows()
	return sf
}

// registerFlows registers all session-related flows
func (sf *SessionFlows) registerFlows() {
	// Flow: Initialize or resume a conversation session
	genkit.DefineFlow(sf.g, "session-init", func(ctx context.Context, req ConversationRequest) (string, error) {
		if req.SessionID != "" {
			exists, err := sf.manager.storage.Exists(ctx, req.SessionID)
			if err != nil {
				logger.WithField("error", err).Warn("failed to check session")
			} else if exists {
				logger.WithField("sessionID", req.SessionID).Info("resuming session")
				return req.SessionID, nil
			}
		}

		session, err := sf.manager.CreateSession(ctx, req.UserID, req.AgentID)
		if err != nil {
			return "", err
		}

		logger.WithField("sessionID", session.ID).Info("session initialized")
		return session.ID, nil
	})

	// Flow: Add message to conversation
	genkit.DefineFlow(sf.g, "session-add-message", func(ctx context.Context, req ConversationRequest) (bool, error) {
		if req.SessionID == "" {
			return false, errors.InvalidArgumentError("session_id is required")
		}

		if err := sf.manager.AddMessage(ctx, req.SessionID, req.ThreadID, "user", req.Message); err != nil {
			return false, err
		}

		logger.WithFields(map[string]interface{}{
			"sessionID": req.SessionID,
			"threadID":  req.ThreadID,
		}).Debug("message added via flow")

		return true, nil
	})

	// Flow: Get conversation history
	genkit.DefineFlow(sf.g, "session-get-history", func(ctx context.Context, req struct {
		SessionID string `json:"session_id"`
		ThreadID  string `json:"thread_id"`
	}) ([]Message, error) {
		messages, err := sf.manager.GetMessages(ctx, req.SessionID, req.ThreadID)
		if err != nil {
			return nil, err
		}

		return messages, nil
	})

	// Flow: Complete conversation cycle (add user message + get history)
	genkit.DefineFlow(sf.g, "session-conversation", func(ctx context.Context, req ConversationRequest) (*ConversationResponse, error) {
		sessionID := req.SessionID
		if sessionID == "" {
			session, err := sf.manager.CreateSession(ctx, req.UserID, req.AgentID)
			if err != nil {
				return nil, err
			}
			sessionID = session.ID
		}

		if err := sf.manager.AddMessage(ctx, sessionID, req.ThreadID, "user", req.Message); err != nil {
			return nil, err
		}

		messages, err := sf.manager.GetMessages(ctx, sessionID, req.ThreadID)
		if err != nil {
			return nil, err
		}

		return &ConversationResponse{
			SessionID: sessionID,
			ThreadID:  req.ThreadID,
			Messages:  messages,
		}, nil
	})

	// Flow: Save session state
	genkit.DefineFlow(sf.g, "session-save-state", func(ctx context.Context, req struct {
		SessionID string                 `json:"session_id"`
		State     map[string]interface{} `json:"state"`
	}) (bool, error) {
		session, err := sf.manager.GetSession(ctx, req.SessionID)
		if err != nil {
			return false, err
		}

		for k, v := range req.State {
			session.SetState(k, v)
		}

		if err := sf.manager.UpdateSession(ctx, session); err != nil {
			return false, err
		}

		return true, nil
	})

	// Flow: Get session summary
	genkit.DefineFlow(sf.g, "session-summary", func(ctx context.Context, sessionID string) (*struct {
		SessionID    string `json:"session_id"`
		UserID       string `json:"user_id"`
		AgentID      string `json:"agent_id"`
		ThreadCount  int    `json:"thread_count"`
		MessageCount int    `json:"message_count"`
	}, error) {
		session, err := sf.manager.GetSession(ctx, sessionID)
		if err != nil {
			return nil, err
		}

		messageCount := 0
		for _, thread := range session.Threads {
			messageCount += thread.MessageCount()
		}

		return &struct {
			SessionID    string `json:"session_id"`
			UserID       string `json:"user_id"`
			AgentID      string `json:"agent_id"`
			ThreadCount  int    `json:"thread_count"`
			MessageCount int    `json:"message_count"`
		}{
			SessionID:    session.ID,
			UserID:       session.UserID,
			AgentID:      session.AgentID,
			ThreadCount:  session.ThreadCount(),
			MessageCount: messageCount,
		}, nil
	})

	logger.Info("session flows registered")
}

// GetManager returns the underlying session manager
func (sf *SessionFlows) GetManager() *Manager {
	return sf.manager
}
