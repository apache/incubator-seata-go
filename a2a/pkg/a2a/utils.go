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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"seata-go-ai-a2a/pkg/types"
)

// AgentCardBuilder provides a fluent API for building agent cards
type AgentCardBuilder struct {
	card *types.AgentCard
}

// NewAgentCard creates a new agent card builder
func NewAgentCard(name, description string) *AgentCardBuilder {
	card := &types.AgentCard{
		ProtocolVersion: "1.0",
		Name:            name,
		Description:     description,
		Version:         "1.0.0",
		Capabilities: &types.AgentCapabilities{
			Streaming:              false,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
		SecuritySchemes: make(map[string]types.SecurityScheme),
		Security:        []*types.Security{},
		Skills:          []*types.AgentSkill{},
	}

	return &AgentCardBuilder{card: card}
}

// WithURL sets the agent URL
func (b *AgentCardBuilder) WithURL(url string) *AgentCardBuilder {
	b.card.URL = url
	return b
}

// WithVersion sets the agent version
func (b *AgentCardBuilder) WithVersion(version string) *AgentCardBuilder {
	b.card.Version = version
	return b
}

// WithProvider sets the provider information
func (b *AgentCardBuilder) WithProvider(organization, url string) *AgentCardBuilder {
	b.card.Provider = &types.AgentProvider{
		Organization: organization,
		URL:          url,
	}
	return b
}

// WithDocumentationURL sets the documentation URL
func (b *AgentCardBuilder) WithDocumentationURL(url string) *AgentCardBuilder {
	b.card.DocumentationURL = url
	return b
}

// WithIconURL sets the icon URL
func (b *AgentCardBuilder) WithIconURL(url string) *AgentCardBuilder {
	b.card.IconURL = url
	return b
}

// WithPreferredTransport sets the preferred transport
func (b *AgentCardBuilder) WithPreferredTransport(transport string) *AgentCardBuilder {
	b.card.PreferredTransport = transport
	return b
}

// AddInterface adds an additional interface
func (b *AgentCardBuilder) AddInterface(url, transport string) *AgentCardBuilder {
	b.card.AdditionalInterfaces = append(b.card.AdditionalInterfaces, &types.AgentInterface{
		URL:       url,
		Transport: transport,
	})
	return b
}

// WithStreaming enables streaming capabilities
func (b *AgentCardBuilder) WithStreaming() *AgentCardBuilder {
	b.card.Capabilities.Streaming = true
	return b
}

// WithPushNotifications enables push notification capabilities
func (b *AgentCardBuilder) WithPushNotifications() *AgentCardBuilder {
	b.card.Capabilities.PushNotifications = true
	return b
}

// AddAPIKeyAuth adds API key authentication scheme
func (b *AgentCardBuilder) AddAPIKeyAuth(name, description, location, keyName string) *AgentCardBuilder {
	scheme := &types.APIKeySecurityScheme{
		Description: description,
		Location:    location, // "query", "header", or "cookie"
		Name:        keyName,
	}

	b.card.SecuritySchemes[name] = scheme
	return b
}

// AddBearerAuth adds Bearer token authentication scheme
func (b *AgentCardBuilder) AddBearerAuth(name, description, bearerFormat string) *AgentCardBuilder {
	scheme := &types.HTTPAuthSecurityScheme{
		Description:  description,
		Scheme:       "bearer",
		BearerFormat: bearerFormat,
	}

	b.card.SecuritySchemes[name] = scheme
	return b
}

// AddBasicAuth adds Basic authentication scheme
func (b *AgentCardBuilder) AddBasicAuth(name, description string) *AgentCardBuilder {
	scheme := &types.HTTPAuthSecurityScheme{
		Description: description,
		Scheme:      "basic",
	}

	b.card.SecuritySchemes[name] = scheme
	return b
}

// AddOAuth2Auth adds OAuth2 authentication scheme
func (b *AgentCardBuilder) AddOAuth2Auth(name, description, authURL, tokenURL string, scopes map[string]string) *AgentCardBuilder {
	flow := &types.AuthorizationCodeOAuthFlow{
		AuthorizationURL: authURL,
		TokenURL:         tokenURL,
		Scopes:           scopes,
	}

	scheme := &types.OAuth2SecurityScheme{
		Description: description,
		Flows:       flow,
	}

	b.card.SecuritySchemes[name] = scheme
	return b
}

// AddMutualTLSAuth adds mutual TLS authentication scheme
func (b *AgentCardBuilder) AddMutualTLSAuth(name, description string) *AgentCardBuilder {
	scheme := &types.MutualTLSSecurityScheme{
		Description: description,
	}

	b.card.SecuritySchemes[name] = scheme
	return b
}

// RequireAuth adds a security requirement
func (b *AgentCardBuilder) RequireAuth(schemes map[string][]string) *AgentCardBuilder {
	security := &types.Security{
		Schemes: schemes,
	}
	b.card.Security = append(b.card.Security, security)
	return b
}

// AddSkill adds a skill to the agent card
func (b *AgentCardBuilder) AddSkill(id, name, description string, tags, examples []string) *AgentCardBuilder {
	skill := &types.AgentSkill{
		ID:          id,
		Name:        name,
		Description: description,
		Tags:        tags,
		Examples:    examples,
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}

	b.card.Skills = append(b.card.Skills, skill)
	return b
}

// AddExtension adds an extension to the capabilities
func (b *AgentCardBuilder) AddExtension(uri, description string, required bool, params map[string]any) *AgentCardBuilder {
	extension := &types.AgentExtension{
		URI:         uri,
		Description: description,
		Required:    required,
		Params:      params,
	}

	b.card.Capabilities.Extensions = append(b.card.Capabilities.Extensions, extension)
	return b
}

// Build returns the constructed agent card
func (b *AgentCardBuilder) Build() *types.AgentCard {
	return b.card
}

// TaskBuilder provides a fluent API for building tasks (useful for testing)
type TaskBuilder struct {
	task *types.Task
}

// NewTask creates a new task builder
func NewTask(id, contextID string) *TaskBuilder {
	task := &types.Task{
		ID:        id,
		ContextID: contextID,
		Kind:      "task",
		Status: &types.TaskStatus{
			State:     types.TaskStateSubmitted,
			Timestamp: time.Now(),
		},
		History:   []*types.Message{},
		Artifacts: []*types.Artifact{},
		Metadata:  make(map[string]any),
	}

	return &TaskBuilder{task: task}
}

// WithState sets the task state
func (b *TaskBuilder) WithState(state types.TaskState) *TaskBuilder {
	b.task.Status.State = state
	b.task.Status.Timestamp = time.Now()
	return b
}

// AddMessage adds a message to the task history
func (b *TaskBuilder) AddMessage(message *types.Message) *TaskBuilder {
	b.task.History = append(b.task.History, message)
	return b
}

// AddArtifact adds an artifact to the task
func (b *TaskBuilder) AddArtifact(artifact *types.Artifact) *TaskBuilder {
	b.task.Artifacts = append(b.task.Artifacts, artifact)
	return b
}

// WithMetadata sets task metadata
func (b *TaskBuilder) WithMetadata(metadata map[string]any) *TaskBuilder {
	b.task.Metadata = metadata
	return b
}

// Build returns the constructed task
func (b *TaskBuilder) Build() *types.Task {
	return b.task
}

// HTTP Response Utilities

// WriteJSONResponse writes a JSON response with appropriate headers
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// WriteErrorResponse writes a standardized error response
func WriteErrorResponse(w http.ResponseWriter, statusCode int, code, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	if details != "" {
		errorResponse["error"].(map[string]interface{})["details"] = details
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// Auth Utilities

// ExtractBearerToken extracts the bearer token from an Authorization header
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}

	return token, nil
}

// ExtractBasicAuth extracts username and password from a Basic auth header
func ExtractBasicAuth(authHeader string) (string, string, error) {
	if authHeader == "" {
		return "", "", fmt.Errorf("missing authorization header")
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", "", fmt.Errorf("invalid authorization header format")
	}

	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", fmt.Errorf("invalid base64 encoding")
	}

	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid credentials format")
	}

	return parts[0], parts[1], nil
}

// EncodeBasicAuth encodes username and password for Basic authentication
func EncodeBasicAuth(username, password string) string {
	credentials := username + ":" + password
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	return "Basic " + encoded
}

// URL Utilities

// ParseTaskIDFromPath extracts task ID from a path like "/api/v1/tasks/{taskId}"
func ParseTaskIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "tasks" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// BuildTaskURL builds a task URL from base URL and task ID
func BuildTaskURL(baseURL, taskID string) string {
	return fmt.Sprintf("%s/api/v1/tasks/%s", strings.TrimSuffix(baseURL, "/"), taskID)
}

// Validation Utilities

// ValidateAgentCard performs basic validation of an agent card
func ValidateAgentCard(card *types.AgentCard) error {
	if card == nil {
		return fmt.Errorf("agent card is nil")
	}

	if card.Name == "" {
		return fmt.Errorf("agent card name is required")
	}

	if card.Description == "" {
		return fmt.Errorf("agent card description is required")
	}

	if card.ProtocolVersion == "" {
		return fmt.Errorf("agent card protocol version is required")
	}

	if card.URL == "" {
		return fmt.Errorf("agent card URL is required")
	}

	// Validate security schemes if present
	for name, scheme := range card.SecuritySchemes {
		if scheme == nil {
			return fmt.Errorf("security scheme '%s' is nil", name)
		}
	}

	return nil
}

// ValidateMessage performs basic validation of a message
func ValidateMessage(msg *types.Message) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	if msg.MessageID == "" {
		return fmt.Errorf("message ID is required")
	}

	if len(msg.Parts) == 0 {
		return fmt.Errorf("message must have at least one part")
	}

	// Validate parts
	for i, part := range msg.Parts {
		if part == nil {
			return fmt.Errorf("message part %d is nil", i)
		}
	}

	return nil
}

// Context Utilities

// GenerateContextID generates a unique context ID
func GenerateContextID() string {
	return fmt.Sprintf("ctx_%d", time.Now().UnixNano())
}

// GenerateMessageID generates a unique message ID
func GenerateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// GenerateTaskID generates a unique task ID
func GenerateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

// GenerateArtifactID generates a unique artifact ID
func GenerateArtifactID() string {
	return fmt.Sprintf("artifact_%d", time.Now().UnixNano())
}

// Content Type Utilities

// DetectContentType attempts to detect content type from filename or content
func DetectContentType(filename string, content []byte) string {
	// First try to detect from content
	if len(content) > 0 {
		return http.DetectContentType(content)
	}

	// Fall back to filename extension
	ext := strings.ToLower(filename)
	if strings.HasSuffix(ext, ".json") {
		return "application/json"
	}
	if strings.HasSuffix(ext, ".xml") {
		return "application/xml"
	}
	if strings.HasSuffix(ext, ".txt") {
		return "text/plain"
	}
	if strings.HasSuffix(ext, ".html") {
		return "text/html"
	}
	if strings.HasSuffix(ext, ".pdf") {
		return "application/pdf"
	}
	if strings.HasSuffix(ext, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") {
		return "image/jpeg"
	}

	return "application/octet-stream"
}

// Time Utilities

// FormatTimestamp formats a timestamp for A2A protocol
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ParseTimestamp parses a timestamp from A2A protocol format
func ParseTimestamp(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// IsTaskStateTerminal checks if a task state is terminal
func IsTaskStateTerminal(state types.TaskState) bool {
	return state.IsTerminal()
}

// IsTaskStateInterrupted checks if a task state is interrupted
func IsTaskStateInterrupted(state types.TaskState) bool {
	return state.IsInterrupted()
}

// Middleware helper for common patterns

// CORSMiddleware creates a CORS middleware function
func CORSMiddleware(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			if len(origins) == 0 {
				// Allow all origins if none specified
				allowed = true
			} else {
				for _, allowedOrigin := range origins {
					if origin == allowedOrigin {
						allowed = true
						break
					}
				}
			}

			if allowed {
				if len(origins) == 0 {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
