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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"seata-go-ai-a2a/pkg/auth/jws"
	"seata-go-ai-a2a/pkg/transport/grpc"
	"seata-go-ai-a2a/pkg/transport/jsonrpc"
	"seata-go-ai-a2a/pkg/transport/rest"
	"seata-go-ai-a2a/pkg/transport/sse"
	"seata-go-ai-a2a/pkg/types"
)

// ClientConfig represents the configuration for the A2A client
type ClientConfig struct {
	// Target agent information
	AgentURL           string `json:"agentUrl"`           // Agent's base URL
	PreferredTransport string `json:"preferredTransport"` // "grpc", "rest", "jsonrpc", "sse"

	// Discovery and AgentCard
	AgentCard          *types.AgentCard `json:"-"`                  // Pre-fetched agent card
	SkipAgentCardFetch bool             `json:"skipAgentCardFetch"` // Skip fetching agent card

	// Authentication Configuration
	AuthConfig *ClientAuthConfig `json:"authConfig"`

	// Transport-specific configurations
	HTTPClient  *http.Client  `json:"-"` // Custom HTTP client
	GRPCOptions []interface{} `json:"-"` // gRPC dial options

	// Timeouts
	RequestTimeout time.Duration `json:"requestTimeout"` // Default: 30s
	ConnectTimeout time.Duration `json:"connectTimeout"` // Default: 10s

	// JWS Validation
	ValidateJWS     bool     `json:"validateJws"`     // Validate agent card JWS
	TrustedJWKSURLs []string `json:"trustedJwksUrls"` // Trusted JWKS URLs for validation

	// Client capabilities
	Streaming bool `json:"streaming"` // Support streaming

	// Retry configuration
	MaxRetries int           `json:"maxRetries"` // Default: 3
	RetryDelay time.Duration `json:"retryDelay"` // Default: 1s
}

// ClientAuthConfig represents authentication configuration for the client
type ClientAuthConfig struct {
	Type        string            `json:"type"`        // "bearer", "apikey", "basic", "mtls"
	Credentials map[string]string `json:"credentials"` // Type-specific credentials

	// OAuth2/JWT specific
	TokenURL     string   `json:"tokenUrl,omitempty"`
	ClientID     string   `json:"clientId,omitempty"`
	ClientSecret string   `json:"clientSecret,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// Client represents the unified A2A client
type Client struct {
	config       *ClientConfig
	agentCard    *types.AgentCard
	jwsValidator *jws.Validator

	// Transport clients
	grpcClient    *grpc.Client
	restClient    *rest.Client
	jsonrpcClient *jsonrpc.Client
	sseClient     *sse.Client
	httpClient    *http.Client

	// Active transport client
	activeTransport string

	// Authentication
	authHeaders map[string]string
	authMutex   sync.RWMutex

	// Lifecycle
	mu     sync.RWMutex
	closed bool
}

// NewClient creates a new A2A client with the given configuration
func NewClient(config *ClientConfig) (*Client, error) {
	if err := validateClientConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	setClientConfigDefaults(config)

	client := &Client{
		config:       config,
		jwsValidator: jws.NewValidator(),
		httpClient:   config.HTTPClient,
		authHeaders:  make(map[string]string),
	}

	if client.httpClient == nil {
		client.httpClient = &http.Client{
			Timeout: config.RequestTimeout,
		}
	}

	// Fetch and validate agent card if not provided
	if !config.SkipAgentCardFetch {
		if err := client.fetchAgentCard(); err != nil {
			return nil, fmt.Errorf("failed to fetch agent card: %w", err)
		}

		if config.ValidateJWS {
			if err := client.validateAgentCardJWS(); err != nil {
				return nil, fmt.Errorf("failed to validate agent card JWS: %w", err)
			}
		}
	} else if config.AgentCard != nil {
		client.agentCard = config.AgentCard
	}

	// Initialize authentication
	if err := client.initAuth(); err != nil {
		return nil, fmt.Errorf("failed to initialize authentication: %w", err)
	}

	// Initialize transport client
	if err := client.initTransport(); err != nil {
		return nil, fmt.Errorf("failed to initialize transport: %w", err)
	}

	return client, nil
}

// SendMessage sends a message to the agent and returns the created task
func (c *Client) SendMessage(ctx context.Context, message *types.Message, metadata map[string]any) (*types.Task, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	req := &types.SendMessageRequest{
		Request:  message,
		Metadata: metadata,
	}

	switch c.activeTransport {
	case "grpc":
		if c.grpcClient == nil {
			return nil, fmt.Errorf("gRPC client not initialized")
		}
		return c.grpcClient.SendMessage(ctx, req)

	case "rest":
		if c.restClient == nil {
			return nil, fmt.Errorf("REST client not initialized")
		}
		resp, err := c.restClient.SendMessage(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp.Task, nil

	case "jsonrpc":
		if c.jsonrpcClient == nil {
			return nil, fmt.Errorf("JSON-RPC client not initialized")
		}
		// For now, assume jsonrpc client returns Task directly or we need to implement it
		return nil, fmt.Errorf("JSON-RPC SendMessage not fully implemented")

	default:
		return nil, fmt.Errorf("unsupported transport: %s", c.activeTransport)
	}
}

// GetTask retrieves a task by ID
func (c *Client) GetTask(ctx context.Context, taskID string) (*types.Task, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	req := &types.GetTaskRequest{
		Name: fmt.Sprintf("tasks/%s", taskID),
	}

	switch c.activeTransport {
	case "grpc":
		return c.grpcClient.GetTask(ctx, req)
	case "rest":
		return c.restClient.GetTask(ctx, req)
	case "jsonrpc":
		return c.jsonrpcClient.GetTask(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported transport: %s", c.activeTransport)
	}
}

// CancelTask cancels a task by ID
func (c *Client) CancelTask(ctx context.Context, taskID string) (*types.Task, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	req := &types.CancelTaskRequest{
		Name: fmt.Sprintf("tasks/%s", taskID),
	}

	switch c.activeTransport {
	case "grpc":
		return c.grpcClient.CancelTask(ctx, req)
	case "rest":
		return c.restClient.CancelTask(ctx, req)
	case "jsonrpc":
		return c.jsonrpcClient.CancelTask(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported transport: %s", c.activeTransport)
	}
}

// ListTasks lists tasks with optional filtering
func (c *Client) ListTasks(ctx context.Context, req *types.ListTasksRequest) (*types.ListTasksResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	switch c.activeTransport {
	case "grpc":
		return c.grpcClient.ListTasks(ctx, req)
	case "rest":
		return c.restClient.ListTasks(ctx, req)
	case "jsonrpc":
		return c.jsonrpcClient.ListTasks(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported transport: %s", c.activeTransport)
	}
}

// GetAgentCard returns the fetched agent card
func (c *Client) GetAgentCard() *types.AgentCard {
	return c.agentCard
}

// StreamTaskUpdates creates a stream to receive task updates (if supported)
func (c *Client) StreamTaskUpdates(ctx context.Context, taskID string) (<-chan *types.Task, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	// For now, return not implemented for all transports
	// This would need to be implemented based on actual transport capabilities
	return nil, fmt.Errorf("streaming not yet implemented")
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	var errs []error

	// Close transport clients
	if c.grpcClient != nil {
		if err := c.grpcClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.restClient != nil {
		if err := c.restClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.jsonrpcClient != nil {
		if err := c.jsonrpcClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.sseClient != nil {
		if err := c.sseClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Close JWS validator
	if c.jwsValidator != nil {
		c.jwsValidator.Close()
	}

	c.closed = true

	if len(errs) > 0 {
		return fmt.Errorf("errors closing client: %v", errs)
	}

	return nil
}

// fetchAgentCard fetches the agent card from the agent URL
func (c *Client) fetchAgentCard() error {
	agentCardURL := c.config.AgentURL
	if !isValidURL(agentCardURL) {
		// Try to construct a valid URL
		if u, err := url.Parse(agentCardURL); err == nil {
			u.Path = "/agentCard"
			agentCardURL = u.String()
		} else {
			agentCardURL = agentCardURL + "/agentCard"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", agentCardURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers if configured
	c.authMutex.RLock()
	for key, value := range c.authHeaders {
		req.Header.Set(key, value)
	}
	c.authMutex.RUnlock()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch agent card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch agent card: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var agentCard types.AgentCard
	if err := json.Unmarshal(body, &agentCard); err != nil {
		return fmt.Errorf("failed to unmarshal agent card: %w", err)
	}

	c.agentCard = &agentCard
	return nil
}

// validateAgentCardJWS validates the JWS signature on the agent card
func (c *Client) validateAgentCardJWS() error {
	if c.agentCard == nil {
		return fmt.Errorf("no agent card to validate")
	}

	if len(c.agentCard.Signatures) == 0 {
		return fmt.Errorf("agent card has no signatures")
	}

	// Create a copy of the agent card without signatures for validation
	agentCardCopy := *c.agentCard
	agentCardCopy.Signatures = nil

	payload, err := json.Marshal(agentCardCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal agent card for validation: %w", err)
	}

	// Validate each signature
	for i, signature := range c.agentCard.Signatures {
		if err := c.jwsValidator.ValidateSignature(context.Background(), payload, signature); err != nil {
			return fmt.Errorf("failed to validate signature %d: %w", i, err)
		}
	}

	return nil
}

// initAuth initializes authentication based on configuration
func (c *Client) initAuth() error {
	if c.config.AuthConfig == nil {
		return nil
	}

	c.authMutex.Lock()
	defer c.authMutex.Unlock()

	switch c.config.AuthConfig.Type {
	case "bearer":
		if token, ok := c.config.AuthConfig.Credentials["token"]; ok {
			c.authHeaders["Authorization"] = "Bearer " + token
		}

	case "apikey":
		if key, ok := c.config.AuthConfig.Credentials["key"]; ok {
			if location, ok := c.config.AuthConfig.Credentials["location"]; ok {
				switch location {
				case "header":
					headerName := c.config.AuthConfig.Credentials["name"]
					if headerName == "" {
						headerName = "X-API-Key"
					}
					c.authHeaders[headerName] = key
				case "query":
					// Query parameter auth will be handled at request time
					// Store in credentials for now
				}
			}
		}

	case "basic":
		if username, ok := c.config.AuthConfig.Credentials["username"]; ok {
			if password, ok := c.config.AuthConfig.Credentials["password"]; ok {
				c.authHeaders["Authorization"] = "Basic " + encodeBasicAuth(username, password)
			}
		}
	}

	return nil
}

// initTransport initializes the appropriate transport client
func (c *Client) initTransport() error {
	preferredTransport := c.config.PreferredTransport

	// If no preferred transport specified, determine from agent card
	if preferredTransport == "" && c.agentCard != nil {
		preferredTransport = c.agentCard.PreferredTransport
	}

	// Default to REST if still not specified
	if preferredTransport == "" {
		preferredTransport = "rest"
	}

	switch preferredTransport {
	case "grpc":
		return c.initGRPCClient()
	case "rest":
		return c.initRESTClient()
	case "jsonrpc":
		return c.initJSONRPCClient()
	case "sse":
		return c.initSSEClient()
	default:
		return fmt.Errorf("unsupported transport: %s", preferredTransport)
	}
}

// initGRPCClient initializes the gRPC client
func (c *Client) initGRPCClient() error {
	grpcAddr := extractGRPCAddress(c.config.AgentURL, c.agentCard)
	if grpcAddr == "" {
		return fmt.Errorf("no gRPC address available")
	}

	config := &grpc.ClientConfig{
		Address: grpcAddr,
		Timeout: c.config.RequestTimeout,
	}

	var err error
	c.grpcClient, err = grpc.NewClient(config)
	if err != nil {
		return err
	}

	c.activeTransport = "grpc"
	return nil
}

// initRESTClient initializes the REST client
func (c *Client) initRESTClient() error {
	config := &rest.ClientConfig{
		BaseURL: c.config.AgentURL,
		Timeout: c.config.RequestTimeout,
	}

	c.restClient = rest.NewClient(config)

	c.activeTransport = "rest"
	return nil
}

// initJSONRPCClient initializes the JSON-RPC client
func (c *Client) initJSONRPCClient() error {
	jsonrpcURL := c.config.AgentURL + "/jsonrpc"

	config := &jsonrpc.ClientConfig{
		Endpoint: jsonrpcURL,
		Timeout:  c.config.RequestTimeout,
	}

	c.jsonrpcClient = jsonrpc.NewClient(config)

	c.activeTransport = "jsonrpc"
	return nil
}

// initSSEClient initializes the SSE client
func (c *Client) initSSEClient() error {
	sseURL := c.config.AgentURL + "/events"

	config := &sse.ClientConfig{
		BaseURL: sseURL,
		Timeout: 0,
	}

	c.sseClient = sse.NewClient(config)

	c.activeTransport = "sse"
	return nil
}

// Helper functions
func validateClientConfig(config *ClientConfig) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	if config.AgentURL == "" {
		return fmt.Errorf("agent URL is required")
	}

	return nil
}

func setClientConfigDefaults(config *ClientConfig) {
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 10 * time.Second
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}
}

func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func extractGRPCAddress(agentURL string, agentCard *types.AgentCard) string {
	// Try to extract from agent card first
	if agentCard != nil {
		for _, iface := range agentCard.AdditionalInterfaces {
			if iface.Transport == "grpc" {
				return iface.URL
			}
		}
	}

	// Default: assume gRPC is on port 9090
	if u, err := url.Parse(agentURL); err == nil {
		u.Scheme = "grpc"
		if u.Port() == "" {
			u.Host += ":9090"
		}
		return u.Host
	}

	return ""
}

func encodeBasicAuth(username, password string) string {
	// This is a placeholder - in real implementation, use base64 encoding
	return fmt.Sprintf("%s:%s", username, password) // Should be base64 encoded
}
