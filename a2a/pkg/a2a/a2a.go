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

// Package a2a provides a unified SDK for the Application-to-Application (A2A) protocol.
//
// This package implements a complete A2A protocol stack with:
//   - Multi-transport support (gRPC, REST, JSON-RPC, SSE)
//   - Comprehensive authentication (JWT, API Key, Basic, mTLS)
//   - JWS signature validation for Agent Cards
//   - Task management with pluggable storage
//   - Rate limiting and audit logging
//   - Simple APIs for quick setup
//   - Production-ready security features
//
// Quick Start - Server:
//
//	// Simple server
//	server, err := a2a.QuickServer("My Agent", "A helpful AI agent", 8080)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer server.Stop()
//
//	// Server with authentication
//	apiKeys := map[string]string{"user1": "secret123"}
//	server, err := a2a.QuickServerWithAuth("My Agent", "Description", 8080, apiKeys)
//
//	// Advanced server
//	server, err := a2a.NewSimpleServer("My Agent", "Description").
//	    WithPort(8080).
//	    WithJWTAuth("https://auth.example.com/.well-known/jwks.json", "my-app", "issuer").
//	    WithStreamingSupport().
//	    WithRedisStore("redis://localhost:6379").
//	    BuildAndStart()
//
// Quick Start - Client:
//
//	// Simple client
//	client, err := a2a.QuickClient("http://localhost:8080")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Send a message
//	message := a2a.TextMessage("Hello, how can you help me?")
//	task, err := client.SendMessage(context.Background(), message, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Wait for completion
//	for {
//	    updatedTask, err := client.GetTask(context.Background(), task.ID)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    if updatedTask.Status.State.IsTerminal() {
//	        if updatedTask.Status.State == types.TaskStateCompleted {
//	            log.Printf("Task completed: %+v", updatedTask.Artifacts)
//	        }
//	        break
//	    }
//
//	    time.Sleep(time.Second)
//	}
//
// Agent Card Creation:
//
//	agentCard := a2a.NewAgentCard("My Agent", "A helpful AI agent").
//	    WithURL("http://localhost:8080").
//	    WithProvider("My Company", "https://company.com").
//	    WithStreamingSupport().
//	    AddAPIKeyAuth("apikey", "API Key Authentication", "header", "X-API-Key").
//	    AddSkill("help", "General Help", "Provides general assistance",
//	             []string{"help", "support"}, []string{"How can I help you?"}).
//	    Build()
//
// Security Features:
//
//	// Authentication middleware
//	authManager := a2a.NewAuthManager()
//	authMiddleware := a2a.NewAuthMiddleware(authManager, jwsValidator, false)
//
//	// Apply to HTTP handler
//	handler = authMiddleware.HTTPMiddleware()(handler)
//
//	// Apply to gRPC server
//	grpcServer := grpc.NewServer(
//	    grpc.UnaryInterceptor(authMiddleware.GRPCUnaryInterceptor()),
//	    grpc.StreamInterceptor(authMiddleware.GRPCStreamInterceptor()),
//	)
//
// JWS Validation:
//
//	// Validate agent card signatures
//	jwsValidator := jws.NewValidator()
//	jwsMiddleware := a2a.NewJWSMiddleware(jwsValidator, trustedJWKSURLs, true)
//	err := jwsMiddleware.ValidateAgentCard(agentCard)
package a2a

import (
	"context"
	"crypto"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	// Import all necessary subpackages for convenience
	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/auth/authenticator"
	"seata-go-ai-a2a/pkg/auth/jws"
	"seata-go-ai-a2a/pkg/task"
	"seata-go-ai-a2a/pkg/task/store"
	"seata-go-ai-a2a/pkg/types"
)

// Version information
const (
	Version         = "1.0.0"
	ProtocolVersion = "1.0"
	SDKName         = "grpc-a2a-sdk"
)

// A2A represents the main SDK interface
type A2A struct {
	server *Server
	client *Client
}

// SDKConfig represents the main SDK configuration
type SDKConfig struct {
	// Server configuration (optional)
	ServerConfig *ServerConfig `json:"serverConfig,omitempty"`

	// Client configuration (optional)
	ClientConfig *ClientConfig `json:"clientConfig,omitempty"`

	// Global settings
	LogLevel      string `json:"logLevel,omitempty"` // "debug", "info", "warn", "error"
	EnableMetrics bool   `json:"enableMetrics,omitempty"`
}

// NewA2A creates a new A2A SDK instance
func NewA2A(config *SDKConfig) (*A2A, error) {
	if config == nil {
		return nil, fmt.Errorf("SDK config is required")
	}

	a2a := &A2A{}

	// Initialize server if configured
	if config.ServerConfig != nil {
		server, err := NewServer(config.ServerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create server: %w", err)
		}
		a2a.server = server
	}

	// Initialize client if configured
	if config.ClientConfig != nil {
		client, err := NewClient(config.ClientConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
		a2a.client = client
	}

	return a2a, nil
}

// GetServer returns the server instance (may be nil)
func (a *A2A) GetServer() *Server {
	return a.server
}

// GetClient returns the client instance (may be nil)
func (a *A2A) GetClient() *Client {
	return a.client
}

// Start starts the SDK (starts server if present)
func (a *A2A) Start() error {
	if a.server != nil {
		return a.server.Start()
	}
	return nil
}

// Stop stops the SDK (stops server and closes client)
func (a *A2A) Stop() error {
	var errs []error

	if a.server != nil {
		if err := a.server.Stop(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.client != nil {
		if err := a.client.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping SDK: %v", errs)
	}

	return nil
}

// Factory functions for common components

// CreateJWTAuthenticator creates a JWT authenticator with the given configuration
func CreateJWTAuthenticator(name, jwksURL, audience, issuer string) auth.Authenticator {
	config := &authenticator.JWTAuthenticatorConfig{
		Name:     name,
		JWKSURL:  jwksURL,
		Audience: audience,
		Issuer:   issuer,
	}
	return authenticator.NewJWTAuthenticator(config)
}

// CreateAPIKeyAuthenticator creates an API key authenticator
func CreateAPIKeyAuthenticator(name string, apiKeys map[string]string) auth.Authenticator {
	// Convert simple map to APIKeyInfo map
	keys := make(map[string]*authenticator.APIKeyInfo)
	for key, userID := range apiKeys {
		keys[key] = &authenticator.APIKeyInfo{
			UserID: userID,
			Name:   userID,
			Scopes: []string{"read", "write"},
		}
	}

	config := &authenticator.APIKeyAuthenticatorConfig{
		Name: name,
		Keys: keys,
		Schemes: []*types.APIKeySecurityScheme{
			{
				Description: "API Key Authentication",
				Location:    "header",
				Name:        "X-API-Key",
			},
		},
	}
	return authenticator.NewAPIKeyAuthenticator(config)
}

// CreateBasicAuthenticator creates a basic auth authenticator
func CreateBasicAuthenticator(name string, users map[string]string) auth.Authenticator {
	// Convert simple map to BasicUserInfo map
	userInfos := make(map[string]*authenticator.BasicUserInfo)
	for username, password := range users {
		// Hash password with bcrypt
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		userInfos[username] = &authenticator.BasicUserInfo{
			UserID:       username,
			Username:     username,
			PasswordHash: string(hashedPassword),
			Name:         username,
			Scopes:       []string{"read", "write"},
			CreatedAt:    time.Now(),
		}
	}

	config := &authenticator.BasicAuthenticatorConfig{
		Name:  name,
		Users: userInfos,
	}
	return authenticator.NewBasicAuthenticator(config)
}

// CreateJWSSigner creates a JWS signer for signing agent cards
func CreateJWSSigner() *jws.Signer {
	return jws.NewSigner()
}

// CreateJWSValidator creates a JWS validator for validating agent cards
func CreateJWSValidator() *jws.Validator {
	return jws.NewValidator()
}

// CreateTaskManager creates a task manager with the specified store
func CreateTaskManager(storeType, connectionString string) (*task.TaskManager, error) {
	var taskStore store.TaskStore

	switch storeType {
	case "memory":
		taskStore = store.NewMemoryTaskStore()
	case "redis":
		if connectionString == "" {
			return nil, fmt.Errorf("connection string required for redis store")
		}
		// For Redis store, need to create client first
		// This is simplified - in production, parse connection string properly
		return nil, fmt.Errorf("redis task store requires proper redis client configuration")
	default:
		return nil, fmt.Errorf("unsupported store type: %s", storeType)
	}

	config := task.DefaultConfig()

	return task.NewTaskManager(taskStore, config), nil
}

// Convenience functions for common agent card configurations

// CreateBasicAgentCard creates a basic agent card with minimal configuration
func CreateBasicAgentCard(name, description, url string) *types.AgentCard {
	return NewAgentCard(name, description).
		WithURL(url).
		Build()
}

// CreateAPIKeyAgentCard creates an agent card with API key authentication
func CreateAPIKeyAgentCard(name, description, url string) *types.AgentCard {
	return NewAgentCard(name, description).
		WithURL(url).
		AddAPIKeyAuth("apikey", "API Key Authentication", "header", "X-API-Key").
		RequireAuth(map[string][]string{"apikey": {}}).
		Build()
}

// CreateJWTAgentCard creates an agent card with JWT authentication
func CreateJWTAgentCard(name, description, url string) *types.AgentCard {
	return NewAgentCard(name, description).
		WithURL(url).
		AddBearerAuth("jwt", "JWT Authentication", "JWT").
		RequireAuth(map[string][]string{"jwt": {}}).
		Build()
}

// CreateStreamingAgentCard creates an agent card with streaming support
func CreateStreamingAgentCard(name, description, url string) *types.AgentCard {
	card := NewAgentCard(name, description).
		WithURL(url).
		WithPushNotifications().
		AddInterface(url+"/events", "sse").
		Build()

	// Enable streaming manually
	card.Capabilities.Streaming = true
	return card
}

// SignAgentCard signs an agent card with JWS
func SignAgentCard(agentCard *types.AgentCard, privateKey crypto.PrivateKey, keyID, jwksURL string) error {
	signer := CreateJWSSigner()
	signature, err := signer.SignAgentCard(context.Background(), agentCard, privateKey, keyID, jwksURL)
	if err != nil {
		return err
	}

	agentCard.Signatures = []*types.AgentCardSignature{signature}
	return nil
}

// ValidateAgentCardJWS validates the JWS signature of an agent card
func ValidateAgentCardJWS(agentCard *types.AgentCard, trustedJWKSURLs []string) error {
	validator := CreateJWSValidator()
	defer validator.Close()

	middleware := NewJWSMiddleware(validator, trustedJWKSURLs, true)
	return middleware.ValidateAgentCard(agentCard)
}

// Error types and utilities

// SDKError represents an SDK-specific error
type SDKError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *SDKError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Common error codes
const (
	ErrCodeInvalidConfig       = "INVALID_CONFIG"
	ErrCodeServerStartFailed   = "SERVER_START_FAILED"
	ErrCodeClientConnFailed    = "CLIENT_CONNECTION_FAILED"
	ErrCodeAuthFailed          = "AUTHENTICATION_FAILED"
	ErrCodeJWSValidationFailed = "JWS_VALIDATION_FAILED"
)

// NewSDKError creates a new SDK error
func NewSDKError(code, message, details string) *SDKError {
	return &SDKError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Info provides SDK information
func Info() map[string]interface{} {
	return map[string]interface{}{
		"sdk_name":             SDKName,
		"sdk_version":          Version,
		"protocol_version":     ProtocolVersion,
		"supported_transports": []string{"grpc", "rest", "jsonrpc", "sse"},
		"supported_auth":       []string{"jwt", "apikey", "basic", "mtls"},
		"features": map[string]bool{
			"streaming":           true,
			"push_notifications":  true,
			"jws_validation":      true,
			"rate_limiting":       true,
			"audit_logging":       true,
			"multiple_transports": true,
			"pluggable_storage":   true,
		},
	}
}

// LogSDKInfo logs SDK information
func LogSDKInfo() {
	info := Info()
	log.Printf("A2A SDK %s initialized", info["sdk_version"])
	log.Printf("Protocol version: %s", info["protocol_version"])
	log.Printf("Supported transports: %v", info["supported_transports"])
	log.Printf("Supported authentication: %v", info["supported_auth"])
}

// Default configurations for common use cases

// DefaultServerConfig returns a default server configuration
func DefaultServerConfig(name, description string) *ServerConfig {
	return &ServerConfig{
		AgentCard: &types.AgentCard{
			ProtocolVersion:    ProtocolVersion,
			Name:               name,
			Description:        description,
			Version:            "1.0.0",
			PreferredTransport: "rest",
			Capabilities: &types.AgentCapabilities{
				Streaming:              false,
				PushNotifications:      false,
				StateTransitionHistory: true,
			},
		},
		ListenAddr:      ":8080",
		GRPCAddr:        ":9090",
		EnableGRPC:      true,
		EnableREST:      true,
		EnableJSONRPC:   true,
		EnableSSE:       false,
		TaskStore:       "memory",
		EnableRateLimit: false,
		EnableAuditLog:  false,
		Authenticators:  []*AuthenticatorConfig{},
	}
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig(agentURL string) *ClientConfig {
	return &ClientConfig{
		AgentURL:           agentURL,
		PreferredTransport: "rest",
		ValidateJWS:        false,
		SkipAgentCardFetch: false,
		Streaming:          false,
		MaxRetries:         3,
	}
}

// Production-ready configurations

// ProductionServerConfig returns a production-ready server configuration
func ProductionServerConfig(name, description, url string) *ServerConfig {
	config := DefaultServerConfig(name, description)
	config.AgentCard.URL = url
	config.EnableRateLimit = true
	config.RateLimit = 1000 // 1000 requests per minute
	config.EnableAuditLog = true
	config.TaskStore = "redis" // Requires Redis URL to be set

	return config
}

// ProductionClientConfig returns a production-ready client configuration
func ProductionClientConfig(agentURL string) *ClientConfig {
	config := DefaultClientConfig(agentURL)
	config.ValidateJWS = true
	config.Streaming = true

	return config
}

// Helper function to create a complete production setup
func NewProductionA2A(serverName, serverDesc, serverURL, clientURL string) (*A2A, error) {
	config := &SDKConfig{
		ServerConfig:  ProductionServerConfig(serverName, serverDesc, serverURL),
		ClientConfig:  ProductionClientConfig(clientURL),
		LogLevel:      "info",
		EnableMetrics: true,
	}

	return NewA2A(config)
}
