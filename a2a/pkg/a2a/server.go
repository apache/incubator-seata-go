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

package a2a

import (
	"context"
	"crypto"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/auth/authenticator"
	"seata-go-ai-a2a/pkg/auth/jws"
	"seata-go-ai-a2a/pkg/handler"
	"seata-go-ai-a2a/pkg/task"
	"seata-go-ai-a2a/pkg/task/store"
	"seata-go-ai-a2a/pkg/transport/grpc"
	"seata-go-ai-a2a/pkg/transport/jsonrpc"
	"seata-go-ai-a2a/pkg/transport/rest"
	"seata-go-ai-a2a/pkg/transport/sse"
	"seata-go-ai-a2a/pkg/types"
)

// ServerConfig represents the configuration for the A2A server
type ServerConfig struct {
	// Basic Configuration
	AgentCard   *types.AgentCard `json:"agentCard"`
	ListenAddr  string           `json:"listenAddr,omitempty"`  // Default: ":8080"
	TLSCertPath string           `json:"tlsCertPath,omitempty"` // Optional TLS
	TLSKeyPath  string           `json:"tlsKeyPath,omitempty"`  // Optional TLS

	// Transport Configuration
	EnableGRPC    bool `json:"enableGRPC,omitempty"`    // Default: true
	EnableREST    bool `json:"enableREST,omitempty"`    // Default: true
	EnableJSONRPC bool `json:"enableJSONRPC,omitempty"` // Default: true
	EnableSSE     bool `json:"enableSSE,omitempty"`     // Default: true

	// gRPC specific
	GRPCAddr string `json:"grpcAddr,omitempty"` // Default: ":9090"

	// CORS Configuration
	CORSOrigins []string `json:"corsOrigins,omitempty"` // Default: allow all

	// Authentication Configuration
	Authenticators []*AuthenticatorConfig `json:"authenticators,omitempty"`

	// JWS Signing Configuration (for AgentCard signing)
	JWSPrivateKey crypto.PrivateKey `json:"-"` // Runtime only
	JWSKeyID      string            `json:"jwsKeyId,omitempty"`
	JWKSURL       string            `json:"jwksUrl,omitempty"`

	// Task Configuration
	TaskStore string `json:"taskStore,omitempty"` // "memory" or "redis"
	RedisURL  string `json:"redisUrl,omitempty"`  // For redis task store

	// Security & Rate Limiting
	EnableRateLimit bool `json:"enableRateLimit,omitempty"`
	RateLimit       int  `json:"rateLimit,omitempty"` // requests per minute
	EnableAuditLog  bool `json:"enableAuditLog,omitempty"`

	// Handler Interface Configuration
	MessageHandler    handler.MessageHandler    `json:"-"` // User-defined message processing handler
	AgentCardProvider handler.AgentCardProvider `json:"-"` // User-defined agent card provider

	// Lifecycle hooks
	OnTaskCreated   func(task *types.Task) `json:"-"`
	OnTaskCompleted func(task *types.Task) `json:"-"`
	OnTaskFailed    func(task *types.Task) `json:"-"`
}

// AuthenticatorConfig represents authenticator configuration
type AuthenticatorConfig struct {
	Type   string                 `json:"type"` // "jwt", "apikey", "basic", "mtls"
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

// Server represents the unified A2A server
type Server struct {
	config       *ServerConfig
	agentCard    *types.AgentCard
	taskManager  *task.TaskManager
	authManager  *AuthManager
	jwsSigner    *jws.Signer
	jwsValidator *jws.Validator

	// Transport servers
	grpcServer    *grpc.Server
	restServer    *rest.Server
	jsonrpcServer *jsonrpc.Server
	sseServer     *sse.Server
	httpServer    *http.Server

	// Lifecycle
	mu      sync.RWMutex
	started bool
	stopped bool
}

// NewServer creates a new A2A server with the given configuration
func NewServer(config *ServerConfig) (*Server, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	setConfigDefaults(config)

	server := &Server{
		config:       config,
		agentCard:    config.AgentCard,
		jwsSigner:    jws.NewSigner(),
		jwsValidator: jws.NewValidator(),
	}

	// Initialize components
	if err := server.initTaskManager(); err != nil {
		return nil, fmt.Errorf("failed to initialize task manager: %w", err)
	}

	if err := server.initAuthManager(); err != nil {
		return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
	}

	if err := server.initTransports(); err != nil {
		return nil, fmt.Errorf("failed to initialize transports: %w", err)
	}

	if err := server.signAgentCard(); err != nil {
		return nil, fmt.Errorf("failed to sign agent card: %w", err)
	}

	return server, nil
}

// NewServerWithHandlers creates a new A2A server with handler interfaces
func NewServerWithHandlers(agentCard *types.AgentCard, messageHandler handler.MessageHandler, agentCardProvider handler.AgentCardProvider) (*Server, error) {
	config := &ServerConfig{
		AgentCard:         agentCard,
		ListenAddr:        ":8080",
		GRPCAddr:          ":9090",
		EnableGRPC:        true,
		EnableREST:        true,
		EnableJSONRPC:     true,
		EnableSSE:         true,
		TaskStore:         "memory",
		MessageHandler:    messageHandler,
		AgentCardProvider: agentCardProvider,
	}

	return NewServer(config)
}

// Start starts the A2A server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}

	// Start transport servers
	if s.config.EnableGRPC && s.grpcServer != nil {
		if err := s.grpcServer.Start(); err != nil {
			return fmt.Errorf("failed to start gRPC server: %w", err)
		}
		log.Printf("A2A gRPC server started on %s", s.config.GRPCAddr)
	}

	// Create HTTP server for REST/JSON-RPC/SSE
	if s.config.EnableREST || s.config.EnableJSONRPC || s.config.EnableSSE {
		mux := http.NewServeMux()

		if s.config.EnableREST && s.restServer != nil {
			mux.Handle("/api/v1/", s.restServer)
			log.Println("A2A REST API enabled on /api/v1/")
		}

		if s.config.EnableJSONRPC && s.jsonrpcServer != nil {
			mux.Handle("/jsonrpc", s.jsonrpcServer)
			log.Println("A2A JSON-RPC enabled on /jsonrpc")
		}

		if s.config.EnableSSE && s.sseServer != nil {
			mux.Handle("/events", s.sseServer)
			log.Println("A2A SSE enabled on /events")
		}

		// AgentCard endpoint
		mux.HandleFunc("/agentCard", s.handleAgentCard)

		s.httpServer = &http.Server{
			Addr:    s.config.ListenAddr,
			Handler: mux,
		}

		go func() {
			var err error
			if s.config.TLSCertPath != "" && s.config.TLSKeyPath != "" {
				err = s.httpServer.ListenAndServeTLS(s.config.TLSCertPath, s.config.TLSKeyPath)
				log.Printf("A2A HTTPS server started on %s", s.config.ListenAddr)
			} else {
				err = s.httpServer.ListenAndServe()
				log.Printf("A2A HTTP server started on %s", s.config.ListenAddr)
			}
			if err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP server error: %v", err)
			}
		}()
	}

	s.started = true
	log.Printf("A2A Server started successfully")

	return nil
}

// Stop gracefully stops the A2A server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil
	}

	// Stop HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Stop gRPC server
	if s.grpcServer != nil {
		if err := s.grpcServer.Stop(); err != nil {
			log.Printf("gRPC server stop error: %v", err)
		}
	}

	// Close validators
	if s.jwsValidator != nil {
		s.jwsValidator.Close()
	}

	s.stopped = true
	log.Printf("A2A Server stopped")

	return nil
}

// GetAgentCard returns the agent card
func (s *Server) GetAgentCard() *types.AgentCard {
	return s.agentCard
}

// GetTaskManager returns the task manager
func (s *Server) GetTaskManager() *task.TaskManager {
	return s.taskManager
}

// initTaskManager initializes the task manager
func (s *Server) initTaskManager() error {
	var taskStore store.TaskStore

	switch s.config.TaskStore {
	case "redis":
		if s.config.RedisURL == "" {
			return fmt.Errorf("redis URL required for redis task store")
		}
		// For now, return error as Redis needs proper client configuration
		return fmt.Errorf("redis task store not implemented in this simplified version")
	case "memory":
		fallthrough
	default:
		taskStore = store.NewMemoryTaskStore()
	}

	config := task.DefaultConfig()

	// Check if handler interfaces are provided
	if s.config.MessageHandler != nil && s.config.AgentCardProvider != nil {
		// Create task manager with handler interfaces
		s.taskManager = task.NewTaskManagerWithHandlers(taskStore, config, s.config.MessageHandler, s.config.AgentCardProvider)
		log.Println("Task manager initialized with handler interfaces")
	} else {
		// Create standard task manager
		s.taskManager = task.NewTaskManager(taskStore, config)
		log.Println("Task manager initialized without handler interfaces")
	}

	// Task manager lifecycle hooks not available in current implementation
	// These would need to be implemented separately if needed

	return nil
}

// initAuthManager initializes the authentication manager
func (s *Server) initAuthManager() error {
	s.authManager = NewAuthManager()

	// Initialize authenticators based on config
	for _, authConfig := range s.config.Authenticators {
		authenticator, err := s.createAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator %s: %w", authConfig.Name, err)
		}
		s.authManager.AddAuthenticator(authenticator)
	}

	return nil
}

// createAuthenticator creates an authenticator based on configuration
func (s *Server) createAuthenticator(config *AuthenticatorConfig) (auth.Authenticator, error) {
	switch config.Type {
	case "jwt":
		jwtConfig := &authenticator.JWTAuthenticatorConfig{
			Name: config.Name,
		}

		// Extract JWT-specific config
		if jwksURL, ok := config.Config["jwksUrl"].(string); ok {
			jwtConfig.JWKSURL = jwksURL
		}
		if audience, ok := config.Config["audience"].(string); ok {
			jwtConfig.Audience = audience
		}
		if issuer, ok := config.Config["issuer"].(string); ok {
			jwtConfig.Issuer = issuer
		}
		if clockSkew, ok := config.Config["clockSkew"].(string); ok {
			if duration, err := time.ParseDuration(clockSkew); err == nil {
				jwtConfig.ClockSkew = duration
			}
		}

		return authenticator.NewJWTAuthenticator(jwtConfig), nil

	case "apikey":
		apiKeyConfig := &authenticator.APIKeyAuthenticatorConfig{
			Name: config.Name,
			Keys: make(map[string]*authenticator.APIKeyInfo),
		}

		// Extract API key-specific config
		if keys, ok := config.Config["apiKeys"].(map[string]interface{}); ok {
			for k, v := range keys {
				if keyValue, ok := v.(string); ok {
					apiKeyConfig.Keys[k] = &authenticator.APIKeyInfo{
						UserID: keyValue,
						Name:   keyValue,
						Scopes: []string{"read", "write"},
					}
				}
			}
		}

		// Set default schemes if not provided
		if apiKeyConfig.Schemes == nil {
			apiKeyConfig.Schemes = []*types.APIKeySecurityScheme{
				{
					Description: "API Key Authentication",
					Location:    "header",
					Name:        "X-API-Key",
				},
			}
		}

		return authenticator.NewAPIKeyAuthenticator(apiKeyConfig), nil

	case "basic":
		basicConfig := &authenticator.BasicAuthenticatorConfig{
			Name:  config.Name,
			Users: make(map[string]*authenticator.BasicUserInfo),
		}

		// Extract Basic auth-specific config
		if users, ok := config.Config["users"].(map[string]interface{}); ok {
			for username, password := range users {
				if passwordStr, ok := password.(string); ok {
					// Hash password with bcrypt
					hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(passwordStr), bcrypt.DefaultCost)
					basicConfig.Users[username] = &authenticator.BasicUserInfo{
						UserID:       username,
						Username:     username,
						PasswordHash: string(hashedPassword),
						Name:         username,
						Scopes:       []string{"read", "write"},
						CreatedAt:    time.Now(),
						IsActive:     true,
					}
				}
			}
		}

		return authenticator.NewBasicAuthenticator(basicConfig), nil

	case "mtls":
		mtlsConfig := &authenticator.MTLSAuthenticatorConfig{
			Name:               config.Name,
			TrustedCAs:         []string{},
			ClientCertificates: make(map[string]*authenticator.ClientCertInfo),
			RequireClientCert:  true,
			VerifyHostname:     true,
		}

		// Extract mTLS-specific config
		if caCerts, ok := config.Config["trustedCAs"].([]interface{}); ok {
			for _, cert := range caCerts {
				if certStr, ok := cert.(string); ok {
					mtlsConfig.TrustedCAs = append(mtlsConfig.TrustedCAs, certStr)
				}
			}
		}

		authenticator, err := authenticator.NewMTLSAuthenticator(mtlsConfig)
		if err != nil {
			return nil, err
		}
		return authenticator, nil

	default:
		return nil, fmt.Errorf("unknown authenticator type: %s", config.Type)
	}
}

// initTransports initializes all transport servers
func (s *Server) initTransports() error {
	// Initialize gRPC transport
	if s.config.EnableGRPC {
		grpcConfig := &grpc.ServerConfig{
			Address:     s.config.GRPCAddr,
			AgentCard:   s.agentCard,
			TaskManager: s.taskManager,
		}

		var err error
		s.grpcServer, err = grpc.NewServer(grpcConfig)
		if err != nil {
			return fmt.Errorf("failed to create gRPC server: %w", err)
		}
	}

	// Initialize REST transport
	if s.config.EnableREST {
		s.restServer = rest.NewServer()
		s.restServer.RegisterA2AHandlers(s.agentCard, s.taskManager)
	}

	// Initialize JSON-RPC transport
	if s.config.EnableJSONRPC {
		s.jsonrpcServer = jsonrpc.NewServer()
		s.jsonrpcServer.RegisterA2AHandlers(s.agentCard, s.taskManager)
		s.jsonrpcServer.SetCORSOrigins(s.config.CORSOrigins)
	}

	// Initialize SSE transport
	if s.config.EnableSSE {
		s.sseServer = sse.NewServer(s.taskManager)
		// SSE server doesn't have RegisterA2AHandlers method - it handles requests directly
	}

	return nil
}

// signAgentCard signs the agent card if JWS configuration is provided
func (s *Server) signAgentCard() error {
	if s.config.JWSPrivateKey == nil || s.config.JWSKeyID == "" || s.config.JWKSURL == "" {
		log.Println("JWS signing configuration not provided, agent card will not be signed")
		return nil
	}

	signature, err := s.jwsSigner.SignAgentCard(
		context.Background(),
		s.agentCard,
		s.config.JWSPrivateKey,
		s.config.JWSKeyID,
		s.config.JWKSURL,
	)
	if err != nil {
		return fmt.Errorf("failed to sign agent card: %w", err)
	}

	s.agentCard.Signatures = []*types.AgentCardSignature{signature}
	log.Printf("Agent card signed successfully with key ID: %s", s.config.JWSKeyID)

	return nil
}

// handleAgentCard handles AgentCard requests
func (s *Server) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Return the agent card directly as JSON
	if err := types.WriteJSONResponse(w, http.StatusOK, s.agentCard); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// validateConfig validates the server configuration
func validateConfig(config *ServerConfig) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	if config.AgentCard == nil {
		return fmt.Errorf("agent card is required")
	}

	if config.AgentCard.Name == "" {
		return fmt.Errorf("agent card name is required")
	}

	if config.AgentCard.ProtocolVersion == "" {
		return fmt.Errorf("agent card protocol version is required")
	}

	// Validate JWS config consistency
	jwsConfigProvided := (config.JWSPrivateKey != nil) || (config.JWSKeyID != "") || (config.JWKSURL != "")
	if jwsConfigProvided {
		if config.JWSPrivateKey == nil {
			return fmt.Errorf("JWS private key is required when JWS configuration is provided")
		}
		if config.JWSKeyID == "" {
			return fmt.Errorf("JWS key ID is required when JWS configuration is provided")
		}
		if config.JWKSURL == "" {
			return fmt.Errorf("JWKS URL is required when JWS configuration is provided")
		}
	}

	return nil
}

// setConfigDefaults sets default values for configuration
func setConfigDefaults(config *ServerConfig) {
	if config.ListenAddr == "" {
		config.ListenAddr = ":8080"
	}

	if config.GRPCAddr == "" {
		config.GRPCAddr = ":9090"
	}

	// Enable all transports by default
	if !config.EnableGRPC && !config.EnableREST && !config.EnableJSONRPC && !config.EnableSSE {
		config.EnableGRPC = true
		config.EnableREST = true
		config.EnableJSONRPC = true
		config.EnableSSE = true
	}

	if config.TaskStore == "" {
		config.TaskStore = "memory"
	}

	if config.RateLimit <= 0 {
		config.RateLimit = 1000 // 1000 requests per minute default
	}
}
