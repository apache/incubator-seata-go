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
	"context"
	"fmt"
	"sync"

	"seata-go-ai-txn-agent/pkg/agent"
	"seata-go-ai-txn-agent/pkg/utils"
)

// BroadcastMessage represents a message to be broadcast
type BroadcastMessage struct {
	Message WebSocketMessage
	Client  *Client
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	Broadcast chan BroadcastMessage

	// Register requests from the clients
	Register chan *Client

	// Unregister requests from clients
	Unregister chan *Client

	// Agent for processing messages
	agent *agent.SagaWorkflowAgent

	// Logger
	logger *utils.Logger

	// Mutex for thread safety
	mu sync.RWMutex

	// Session storage
	sessions map[string]*agent.SagaWorkflowAgent
}

// NewHub creates a new Hub
func NewHub(agentConfig agent.AgentConfig) *Hub {
	// Create a default agent
	defaultAgent, err := agent.NewSagaWorkflowAgent(agentConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create default agent: %w", err))
	}

	return &Hub{
		Broadcast:  make(chan BroadcastMessage),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		agent:      defaultAgent,
		logger:     utils.GetLogger("websocket-hub"),
		sessions:   make(map[string]*agent.SagaWorkflowAgent),
	}
}

// Run starts the hub
func (h *Hub) Run(ctx context.Context) {
	h.logger.Info("WebSocket hub started")

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("WebSocket hub shutting down")
			return

		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case broadcastMsg := <-h.Broadcast:
			h.handleMessage(ctx, broadcastMsg)
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true
	h.logger.WithField("client_id", client.ID).WithField("session_id", client.SessionID).Info("Client registered")

	// Send welcome message
	welcomeMsg := ConnectedMessage{
		SessionID: client.SessionID,
		Message:   "🚀 欢迎使用 Seata Go Workflow Agent！我将帮助您设计分布式事务工作流。",
	}
	client.SendMessage(MessageTypeConnected, welcomeMsg)
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.Send)
		h.logger.WithField("client_id", client.ID).Info("Client unregistered")
	}
}

// handleMessage handles incoming messages from clients
func (h *Hub) handleMessage(ctx context.Context, broadcastMsg BroadcastMessage) {
	client := broadcastMsg.Client
	message := broadcastMsg.Message

	h.logger.WithFields(map[string]interface{}{
		"client_id":    client.ID,
		"message_type": message.Type,
	}).Debug("Handling message")

	switch message.Type {
	case MessageTypeUserInput:
		h.handleUserInput(ctx, client, message)
	case MessageTypeClearChat:
		h.handleClearChat(client, message)
	case MessageTypeGetHistory:
		h.handleGetHistory(client, message)
	default:
		h.logger.WithField("message_type", message.Type).Warn("Unknown message type")
	}
}

// handleUserInput handles user input messages
func (h *Hub) handleUserInput(ctx context.Context, client *Client, message WebSocketMessage) {
	inputData, ok := message.Data.(map[string]interface{})
	if !ok {
		client.SendMessage(MessageTypeError, ErrorMessage{
			Message: "Invalid input data format",
			Code:    "INVALID_FORMAT",
		})
		return
	}

	content, ok := inputData["content"].(string)
	if !ok || content == "" {
		client.SendMessage(MessageTypeError, ErrorMessage{
			Message: "Content is required",
			Code:    "MISSING_CONTENT",
		})
		return
	}

	// Send typing indicator
	client.SendMessage(MessageTypeTyping, TypingMessage{IsTyping: true})

	// Get or create agent for this session
	sessionAgent := h.getSessionAgent(client.SessionID)

	// Process message with agent
	response, err := sessionAgent.SendMessage(ctx, content)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process message with agent")
		client.SendMessage(MessageTypeError, ErrorMessage{
			Message: fmt.Sprintf("处理消息失败: %v", err),
			Code:    "PROCESSING_ERROR",
		})
		client.SendMessage(MessageTypeTyping, TypingMessage{IsTyping: false})
		return
	}

	// Send response
	responseData := AgentResponseMessage{
		AgentResponse: response,
		SessionID:     client.SessionID,
	}
	client.SendMessage(MessageTypeAgentResponse, responseData)
	client.SendMessage(MessageTypeTyping, TypingMessage{IsTyping: false})
}

// handleClearChat handles clear chat messages
func (h *Hub) handleClearChat(client *Client, message WebSocketMessage) {
	sessionAgent := h.getSessionAgent(client.SessionID)
	sessionAgent.ClearConversation()

	client.SendMessage(MessageTypeConnected, ConnectedMessage{
		SessionID: client.SessionID,
		Message:   "对话历史已清空，让我们重新开始吧！",
	})
}

// handleGetHistory handles get history messages
func (h *Hub) handleGetHistory(client *Client, message WebSocketMessage) {
	sessionAgent := h.getSessionAgent(client.SessionID)
	history := sessionAgent.GetConversationHistory()

	client.SendMessage(MessageTypeAgentResponse, ChatHistoryMessage{
		Messages:  history,
		SessionID: client.SessionID,
	})
}

// getSessionAgent gets or creates an agent for the given session
func (h *Hub) getSessionAgent(sessionID string) *agent.SagaWorkflowAgent {
	h.mu.Lock()
	defer h.mu.Unlock()

	if sessionAgent, exists := h.sessions[sessionID]; exists {
		return sessionAgent
	}

	// Create new agent for this session using the same config as the hub's agent
	// Note: We'll use the default config for now since GetConfig() is not available
	config := agent.AgentConfig{
		BaseURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:     "sk-63476c61c99b4cd78c0e8d1348560761",
		Model:      "qwen-max",
		Timeout:    60,
		MaxRetries: 3,
	}

	sessionAgent, err := agent.NewSagaWorkflowAgent(config)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create session agent")
		return h.agent // Fallback to default agent
	}

	h.sessions[sessionID] = sessionAgent
	h.logger.WithField("session_id", sessionID).Info("Created new session agent")

	return sessionAgent
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
