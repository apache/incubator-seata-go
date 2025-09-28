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
