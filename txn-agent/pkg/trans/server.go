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
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"seata-go-ai-txn-agent/pkg/agent"
	"seata-go-ai-txn-agent/pkg/utils"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

// Server represents the WebSocket server
type Server struct {
	hub    *Hub
	router *gin.Engine
	logger *utils.Logger
	port   string
}

// NewServer creates a new WebSocket server
func NewServer(port string, agentConfig agent.AgentConfig) *Server {
	hub := NewHub(agentConfig)

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	server := &Server{
		hub:    hub,
		router: router,
		logger: utils.GetLogger("websocket-server"),
		port:   port,
	}

	server.setupRoutes()
	return server
}

// setupRoutes sets up HTTP routes
func (s *Server) setupRoutes() {
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
		api.POST("/sessions", s.handleCreateSession)
	}

	// Serve static files (for frontend if needed)
	s.router.Static("/static", "./static")
	s.router.StaticFile("/", "./static/index.html")
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upgrade connection")
		return
	}

	clientID := uuid.New().String()
	client := NewClient(clientID, sessionID, conn, s.hub)

	s.logger.WithFields(map[string]interface{}{
		"client_id":   clientID,
		"session_id":  sessionID,
		"remote_addr": c.Request.RemoteAddr,
	}).Info("New WebSocket connection")

	// Register client
	s.hub.Register <- client

	// Start goroutines
	ctx := c.Request.Context()
	go client.WritePump(ctx)
	go client.ReadPump(ctx)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"clients":   s.hub.GetClientCount(),
	})
}

// handleStatus handles status requests
func (s *Server) handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": gin.H{
			"status":  "running",
			"version": "1.0.0",
			"uptime":  time.Since(time.Now()).String(),
		},
		"websocket": gin.H{
			"clients": s.hub.GetClientCount(),
		},
	})
}

// handleCreateSession handles session creation requests
func (s *Server) handleCreateSession(c *gin.Context) {
	sessionID := uuid.New().String()
	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"created_at": time.Now().UTC(),
	})
}

// Start starts the WebSocket server
func (s *Server) Start(ctx context.Context) error {
	// Start hub
	go s.hub.Run(ctx)

	s.logger.WithField("port", s.port).Info("Starting WebSocket server")

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

	// Shutdown server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

// GetHub returns the hub instance
func (s *Server) GetHub() *Hub {
	return s.hub
}
