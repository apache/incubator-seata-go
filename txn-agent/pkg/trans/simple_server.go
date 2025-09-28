package trans

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"seata-go-ai-txn-agent/pkg/agent"
	"seata-go-ai-txn-agent/pkg/utils"
)

// SimpleServer represents a simplified WebSocket server for single connection
type SimpleServer struct {
	agentConfig   agent.AgentConfig
	agentInstance *agent.SagaWorkflowAgent // Single persistent agent instance
	router        *gin.Engine
	logger        *utils.Logger
	port          string
	conn          *websocket.Conn
}

var simpleUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

// NewSimpleServer creates a new simplified WebSocket server
func NewSimpleServer(port string, agentConfig agent.AgentConfig) *SimpleServer {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	logger := utils.GetLogger("simple-websocket-server")

	// Create single persistent agent instance
	agentInstance, err := agent.NewSagaWorkflowAgent(agentConfig)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create agent instance")
	}

	logger.Info("Created persistent WorkflowAgent instance")

	server := &SimpleServer{
		agentConfig:   agentConfig,
		agentInstance: agentInstance,
		router:        router,
		logger:        logger,
		port:          port,
	}

	server.setupRoutes()
	return server
}

// setupRoutes sets up HTTP routes
func (s *SimpleServer) setupRoutes() {
	// Enable CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	s.router.Use(cors.New(config))

	// WebSocket endpoint
	s.router.GET("/ws", s.handleWebSocket)

	// Health check endpoint
	s.router.GET("/health", s.handleHealth)

	// API endpoints
	api := s.router.Group("/api/v1")
	{
		api.GET("/status", s.handleStatus)
	}
}

// handleWebSocket handles WebSocket connections
func (s *SimpleServer) handleWebSocket(c *gin.Context) {
	conn, err := simpleUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upgrade connection")
		return
	}

	s.conn = conn
	s.logger.WithField("remote_addr", c.Request.RemoteAddr).Info("New WebSocket connection")

	// Handle connection in goroutine
	go s.handleConnection(conn)
}

// handleConnection handles a WebSocket connection
func (s *SimpleServer) handleConnection(conn *websocket.Conn) {
	defer func() {
		conn.Close()
		s.conn = nil
		s.logger.Info("WebSocket connection closed")
	}()

	// Send welcome message
	welcomeMsg := WebSocketMessage{
		Type: "system",
		Data: map[string]interface{}{
			"message":   "Connected to Seata Workflow Agent",
			"timestamp": time.Now().UTC(),
		},
		Timestamp: time.Now().UTC(),
	}
	s.sendMessage(conn, welcomeMsg)

	for {
		var message WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.WithError(err).Error("Unexpected WebSocket close")
			}
			break
		}

		s.logger.WithField("message_type", message.Type).Debug("Received message")

		// Handle different message types
		switch message.Type {
		case MessageTypeUserInput:
			s.handleUserInput(conn, message)
		case MessageTypeClearChat:
			s.handleClearChat(conn, message)
		case "ping":
			s.handlePing(conn, message)
		default:
			s.logger.WithField("message_type", message.Type).Warn("Unknown message type")
		}
	}
}

// handleUserInput processes user input messages
func (s *SimpleServer) handleUserInput(conn *websocket.Conn, message WebSocketMessage) {
	// Send typing indicator
	typingMsg := WebSocketMessage{
		Type: MessageTypeTyping,
		Data: map[string]interface{}{
			"isTyping": true,
		},
		Timestamp: time.Now().UTC(),
	}
	s.sendMessage(conn, typingMsg)

	// Extract user input
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		s.sendError(conn, "Invalid message data format")
		return
	}

	content, ok := data["content"].(string)
	if !ok {
		s.sendError(conn, "Missing or invalid content field")
		return
	}

	s.logger.WithField("user_input", content).Info("Processing user input")

	// Use persistent agent instance to process message
	for i := 0; i < len(s.agentInstance.GetConversationHistory()); i++ {
		fmt.Println(strconv.Itoa(i), s.agentInstance.GetConversationHistory()[i])
	}

	response, err := s.agentInstance.SendMessage(context.Background(), content)
	if err != nil {
		s.logger.WithError(err).Error("Failed to process message")
		s.sendError(conn, fmt.Sprintf("Failed to process message: %v", err))
		return
	}

	// Stop typing indicator
	stopTypingMsg := WebSocketMessage{
		Type: MessageTypeTyping,
		Data: map[string]interface{}{
			"isTyping": false,
		},
		Timestamp: time.Now().UTC(),
	}
	s.sendMessage(conn, stopTypingMsg)

	// Send agent response
	responseMsg := WebSocketMessage{
		Type:      MessageTypeAgentResponse,
		Data:      response,
		Timestamp: time.Now().UTC(),
	}
	s.sendMessage(conn, responseMsg)
}

// handleClearChat processes clear chat messages
func (s *SimpleServer) handleClearChat(conn *websocket.Conn, message WebSocketMessage) {
	s.logger.Info("Clearing chat history for agent")

	// Clear agent conversation history
	s.agentInstance.ClearConversation()

	// Send confirmation
	confirmMsg := WebSocketMessage{
		Type: "chat_cleared",
		Data: map[string]interface{}{
			"message":   "Chat history cleared",
			"timestamp": time.Now().UTC(),
		},
		Timestamp: time.Now().UTC(),
	}
	s.sendMessage(conn, confirmMsg)
}

// handlePing processes ping messages
func (s *SimpleServer) handlePing(conn *websocket.Conn, message WebSocketMessage) {
	pongMsg := WebSocketMessage{
		Type: "pong",
		Data: map[string]interface{}{
			"timestamp": time.Now().UTC(),
		},
		Timestamp: time.Now().UTC(),
	}
	s.sendMessage(conn, pongMsg)
}

// sendMessage sends a message to the WebSocket connection
func (s *SimpleServer) sendMessage(conn *websocket.Conn, message WebSocketMessage) {
	if err := conn.WriteJSON(message); err != nil {
		s.logger.WithError(err).Error("Failed to send message")
	}
}

// sendError sends an error message
func (s *SimpleServer) sendError(conn *websocket.Conn, errorMsg string) {
	errorMessage := WebSocketMessage{
		Type: MessageTypeError,
		Data: map[string]interface{}{
			"message":   errorMsg,
			"timestamp": time.Now().UTC(),
		},
		Timestamp: time.Now().UTC(),
	}
	s.sendMessage(conn, errorMessage)
}

// handleHealth handles health check requests
func (s *SimpleServer) handleHealth(c *gin.Context) {
	connected := s.conn != nil
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"connected": connected,
	})
}

// handleStatus handles status requests
func (s *SimpleServer) handleStatus(c *gin.Context) {
	connected := s.conn != nil
	c.JSON(http.StatusOK, gin.H{
		"server": gin.H{
			"status":  "running",
			"version": "1.0.0",
		},
		"websocket": gin.H{
			"connected": connected,
		},
	})
}

// Start starts the WebSocket server
func (s *SimpleServer) Start(ctx context.Context) error {
	s.logger.WithField("port", s.port).Info("Starting simple WebSocket server")

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", s.port),
		Handler: s.router,
	}

	// Start server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	s.logger.Info("Shutting down server...")

	// Close WebSocket connection if exists
	if s.conn != nil {
		s.conn.Close()
	}

	// Shutdown server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
