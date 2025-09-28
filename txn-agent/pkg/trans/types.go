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

package trans

import (
	"time"

	"seata-go-ai-txn-agent/pkg/agent"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Client to server messages
	MessageTypeUserInput  MessageType = "user_input"
	MessageTypeClearChat  MessageType = "clear_chat"
	MessageTypeGetHistory MessageType = "get_history"

	// Server to client messages
	MessageTypeAgentResponse MessageType = "agent_response"
	MessageTypeError         MessageType = "error"
	MessageTypeConnected     MessageType = "connected"
	MessageTypeTyping        MessageType = "typing"
)

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      MessageType `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	ID        string      `json:"id,omitempty"`
}

// UserInputMessage represents user input data
type UserInputMessage struct {
	Content   string `json:"content"`
	SessionID string `json:"session_id,omitempty"`
}

// AgentResponseMessage represents agent response data
type AgentResponseMessage struct {
	*agent.AgentResponse
	SessionID string `json:"session_id,omitempty"`
}

// ErrorMessage represents error data
type ErrorMessage struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ConnectedMessage represents connection confirmation
type ConnectedMessage struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// TypingMessage represents typing indicator
type TypingMessage struct {
	IsTyping bool `json:"is_typing"`
}

// ChatHistoryMessage represents chat history
type ChatHistoryMessage struct {
	Messages  []agent.ConversationMessage `json:"messages"`
	SessionID string                      `json:"session_id"`
}
