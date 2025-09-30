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
	"crypto"
	"fmt"
	"log"
	"time"

	"seata-go-ai-a2a/pkg/types"
)

// SimpleServerBuilder provides a fluent API for building A2A servers with minimal configuration
type SimpleServerBuilder struct {
	config *ServerConfig
}

// NewSimpleServer creates a new simple server builder
func NewSimpleServer(name, description string) *SimpleServerBuilder {
	agentCard := &types.AgentCard{
		ProtocolVersion:    "1.0",
		Name:               name,
		Description:        description,
		PreferredTransport: "rest",
		Version:            "1.0.0",
		Capabilities: &types.AgentCapabilities{
			Streaming:              false,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
	}

	config := &ServerConfig{
		AgentCard:       agentCard,
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

	return &SimpleServerBuilder{config: config}
}

// WithURL sets the agent URL
func (b *SimpleServerBuilder) WithURL(url string) *SimpleServerBuilder {
	b.config.AgentCard.URL = url
	return b
}

// WithPort sets the HTTP listening port
func (b *SimpleServerBuilder) WithPort(port int) *SimpleServerBuilder {
	b.config.ListenAddr = fmt.Sprintf(":%d", port)
	return b
}

// WithGRPCPort sets the gRPC listening port
func (b *SimpleServerBuilder) WithGRPCPort(port int) *SimpleServerBuilder {
	b.config.GRPCAddr = fmt.Sprintf(":%d", port)
	return b
}

// WithTLS enables TLS with the provided certificate and key files
func (b *SimpleServerBuilder) WithTLS(certPath, keyPath string) *SimpleServerBuilder {
	b.config.TLSCertPath = certPath
	b.config.TLSKeyPath = keyPath
	return b
}

// WithAPIKeyAuth adds API key authentication
func (b *SimpleServerBuilder) WithAPIKeyAuth(apiKeys map[string]string) *SimpleServerBuilder {
	authConfig := &AuthenticatorConfig{
		Type: "apikey",
		Name: "apikey-auth",
		Config: map[string]interface{}{
			"location": "header",
			"keyName":  "X-API-Key",
			"apiKeys":  apiKeys,
		},
	}
	b.config.Authenticators = append(b.config.Authenticators, authConfig)
	return b
}

// WithJWTAuth adds JWT authentication
func (b *SimpleServerBuilder) WithJWTAuth(jwksURL, audience, issuer string) *SimpleServerBuilder {
	authConfig := &AuthenticatorConfig{
		Type: "jwt",
		Name: "jwt-auth",
		Config: map[string]interface{}{
			"jwksUrl":  jwksURL,
			"audience": audience,
			"issuer":   issuer,
		},
	}
	b.config.Authenticators = append(b.config.Authenticators, authConfig)
	return b
}

// WithBasicAuth adds basic authentication
func (b *SimpleServerBuilder) WithBasicAuth(users map[string]string) *SimpleServerBuilder {
	authConfig := &AuthenticatorConfig{
		Type: "basic",
		Name: "basic-auth",
		Config: map[string]interface{}{
			"users": users,
		},
	}
	b.config.Authenticators = append(b.config.Authenticators, authConfig)
	return b
}

// WithJWSSigning enables JWS signing for the agent card
func (b *SimpleServerBuilder) WithJWSSigning(privateKey crypto.PrivateKey, keyID, jwksURL string) *SimpleServerBuilder {
	b.config.JWSPrivateKey = privateKey
	b.config.JWSKeyID = keyID
	b.config.JWKSURL = jwksURL
	return b
}

// WithRedisStore configures Redis as the task store
func (b *SimpleServerBuilder) WithRedisStore(redisURL string) *SimpleServerBuilder {
	b.config.TaskStore = "redis"
	b.config.RedisURL = redisURL
	return b
}

// WithStreamingSupport enables streaming capabilities
func (b *SimpleServerBuilder) WithStreamingSupport() *SimpleServerBuilder {
	b.config.AgentCard.Capabilities.Streaming = true
	b.config.EnableSSE = true
	return b
}

// WithPushNotifications enables push notification support
func (b *SimpleServerBuilder) WithPushNotifications() *SimpleServerBuilder {
	b.config.AgentCard.Capabilities.PushNotifications = true
	return b
}

// WithCORS sets allowed CORS origins
func (b *SimpleServerBuilder) WithCORS(origins []string) *SimpleServerBuilder {
	b.config.CORSOrigins = origins
	return b
}

// WithRateLimit enables rate limiting
func (b *SimpleServerBuilder) WithRateLimit(requestsPerMinute int) *SimpleServerBuilder {
	b.config.EnableRateLimit = true
	b.config.RateLimit = requestsPerMinute
	return b
}

// OnTaskCreated sets a callback for task creation events
func (b *SimpleServerBuilder) OnTaskCreated(callback func(task *types.Task)) *SimpleServerBuilder {
	b.config.OnTaskCreated = callback
	return b
}

// OnTaskCompleted sets a callback for task completion events
func (b *SimpleServerBuilder) OnTaskCompleted(callback func(task *types.Task)) *SimpleServerBuilder {
	b.config.OnTaskCompleted = callback
	return b
}

// OnTaskFailed sets a callback for task failure events
func (b *SimpleServerBuilder) OnTaskFailed(callback func(task *types.Task)) *SimpleServerBuilder {
	b.config.OnTaskFailed = callback
	return b
}

// Build creates and returns the configured A2A server
func (b *SimpleServerBuilder) Build() (*Server, error) {
	return NewServer(b.config)
}

// BuildAndStart creates, starts, and returns the A2A server
func (b *SimpleServerBuilder) BuildAndStart() (*Server, error) {
	server, err := b.Build()
	if err != nil {
		return nil, err
	}

	if err := server.Start(); err != nil {
		return nil, err
	}

	return server, nil
}

// SimpleClientBuilder provides a fluent API for building A2A clients with minimal configuration
type SimpleClientBuilder struct {
	config *ClientConfig
}

// NewSimpleClient creates a new simple client builder
func NewSimpleClient(agentURL string) *SimpleClientBuilder {
	config := &ClientConfig{
		AgentURL:           agentURL,
		RequestTimeout:     30 * time.Second,
		ConnectTimeout:     10 * time.Second,
		MaxRetries:         3,
		RetryDelay:         time.Second,
		ValidateJWS:        false,
		SkipAgentCardFetch: false,
	}

	return &SimpleClientBuilder{config: config}
}

// WithTransport sets the preferred transport
func (b *SimpleClientBuilder) WithTransport(transport string) *SimpleClientBuilder {
	b.config.PreferredTransport = transport
	return b
}

// WithTimeout sets request timeout
func (b *SimpleClientBuilder) WithTimeout(timeout time.Duration) *SimpleClientBuilder {
	b.config.RequestTimeout = timeout
	return b
}

// WithAPIKeyAuth adds API key authentication
func (b *SimpleClientBuilder) WithAPIKeyAuth(apiKey string) *SimpleClientBuilder {
	b.config.AuthConfig = &ClientAuthConfig{
		Type: "apikey",
		Credentials: map[string]string{
			"key":      apiKey,
			"location": "header",
			"name":     "X-API-Key",
		},
	}
	return b
}

// WithBearerToken adds bearer token authentication
func (b *SimpleClientBuilder) WithBearerToken(token string) *SimpleClientBuilder {
	b.config.AuthConfig = &ClientAuthConfig{
		Type: "bearer",
		Credentials: map[string]string{
			"token": token,
		},
	}
	return b
}

// WithBasicAuth adds basic authentication
func (b *SimpleClientBuilder) WithBasicAuth(username, password string) *SimpleClientBuilder {
	b.config.AuthConfig = &ClientAuthConfig{
		Type: "basic",
		Credentials: map[string]string{
			"username": username,
			"password": password,
		},
	}
	return b
}

// WithJWSValidation enables JWS validation for agent cards
func (b *SimpleClientBuilder) WithJWSValidation(trustedJWKSURLs []string) *SimpleClientBuilder {
	b.config.ValidateJWS = true
	b.config.TrustedJWKSURLs = trustedJWKSURLs
	return b
}

// WithStreaming enables streaming support
func (b *SimpleClientBuilder) WithStreaming() *SimpleClientBuilder {
	b.config.Streaming = true
	return b
}

// WithRetries configures retry behavior
func (b *SimpleClientBuilder) WithRetries(maxRetries int, delay time.Duration) *SimpleClientBuilder {
	b.config.MaxRetries = maxRetries
	b.config.RetryDelay = delay
	return b
}

// Build creates and returns the configured A2A client
func (b *SimpleClientBuilder) Build() (*Client, error) {
	return NewClient(b.config)
}

// Convenience functions for quick setup

// QuickServer creates and starts a simple A2A server with minimal configuration
func QuickServer(name, description string, port int) (*Server, error) {
	return NewSimpleServer(name, description).
		WithPort(port).
		BuildAndStart()
}

// QuickServerWithAuth creates and starts a simple A2A server with API key authentication
func QuickServerWithAuth(name, description string, port int, apiKeys map[string]string) (*Server, error) {
	return NewSimpleServer(name, description).
		WithPort(port).
		WithAPIKeyAuth(apiKeys).
		BuildAndStart()
}

// QuickClient creates a simple A2A client
func QuickClient(agentURL string) (*Client, error) {
	return NewSimpleClient(agentURL).Build()
}

// QuickClientWithAuth creates a simple A2A client with API key authentication
func QuickClientWithAuth(agentURL, apiKey string) (*Client, error) {
	return NewSimpleClient(agentURL).
		WithAPIKeyAuth(apiKey).
		Build()
}

// Message helpers for easy message creation

// TextMessage creates a simple text message
func TextMessage(text string) *types.Message {
	return &types.Message{
		MessageID: fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      types.RoleUser,
		Parts: []types.Part{
			&types.TextPart{Text: text},
		},
		Kind: "request",
	}
}

// TextMessageWithContext creates a text message with context ID
func TextMessageWithContext(text, contextID string) *types.Message {
	msg := TextMessage(text)
	msg.ContextID = contextID
	return msg
}

// FileMessage creates a message with file content
func FileMessage(filename, mimeType string, content []byte) *types.Message {
	return &types.Message{
		MessageID: fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      types.RoleUser,
		Parts: []types.Part{
			&types.FilePart{
				Name:     filename,
				MimeType: mimeType,
				Content:  &types.FileWithBytes{Bytes: content},
			},
		},
		Kind: "request",
	}
}

// DataMessage creates a message with structured data
func DataMessage(data map[string]any) *types.Message {
	return &types.Message{
		MessageID: fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      types.RoleUser,
		Parts: []types.Part{
			&types.DataPart{Data: data},
		},
		Kind: "request",
	}
}

// Usage examples and documentation

// Example usage:
//
// // Simple server setup
// server, err := QuickServer("My Agent", "A helpful AI agent", 8080)
// if err != nil {
//     log.Fatal(err)
// }
// defer server.Stop()
//
// // Server with authentication
// apiKeys := map[string]string{"user1": "secret123", "user2": "secret456"}
// server, err := QuickServerWithAuth("My Agent", "A helpful AI agent", 8080, apiKeys)
//
// // Advanced server setup
// server, err := NewSimpleServer("My Agent", "A helpful AI agent").
//     WithPort(8080).
//     WithJWTAuth("https://auth.example.com/.well-known/jwks.json", "my-app", "auth.example.com").
//     WithStreamingSupport().
//     WithPushNotifications().
//     WithRedisStore("redis://localhost:6379").
//     OnTaskCreated(func(task *types.Task) {
//         log.Printf("Task created: %s", task.ID)
//     }).
//     BuildAndStart()
//
// // Simple client setup
// client, err := QuickClient("http://localhost:8080")
// if err != nil {
//     log.Fatal(err)
// }
// defer client.Close()
//
// // Client with authentication
// client, err := QuickClientWithAuth("http://localhost:8080", "secret123")
//
// // Advanced client setup
// client, err := NewSimpleClient("http://localhost:8080").
//     WithTransport("grpc").
//     WithBearerToken("jwt-token-here").
//     WithJWSValidation([]string{"https://trusted.example.com/.well-known/jwks.json"}).
//     WithStreaming().
//     Build()
//
// // Send a message
// message := TextMessage("Hello, how can you help me?")
// task, err := client.SendMessage(context.Background(), message, nil)
// if err != nil {
//     log.Fatal(err)
// }
//
// // Wait for completion and get result
// for {
//     updatedTask, err := client.GetTask(context.Background(), task.ID)
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     if updatedTask.Status.State.IsTerminal() {
//         if updatedTask.Status.State == types.TaskStateCompleted {
//             log.Printf("Task completed: %+v", updatedTask.Artifacts)
//         } else {
//             log.Printf("Task failed: %s", updatedTask.Status.Update)
//         }
//         break
//     }
//
//     time.Sleep(time.Second)
// }

// LoggingMiddleware provides simple logging for development
func LoggingMiddleware() func(*types.Task) {
	return func(task *types.Task) {
		log.Printf("[A2A] Task %s: %s", task.ID, task.Status.State.String())
	}
}

// DebugMiddleware provides detailed logging for debugging
func DebugMiddleware() func(*types.Task) {
	return func(task *types.Task) {
		log.Printf("[A2A DEBUG] Task %s (%s): State=%s, Messages=%d, Artifacts=%d",
			task.ID, task.ContextID, task.Status.State.String(),
			len(task.History), len(task.Artifacts))
	}
}
