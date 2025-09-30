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

package authenticator

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/types"
)

// APIKeyAuthenticator implements API Key-based authentication
type APIKeyAuthenticator struct {
	name      string
	validKeys map[string]*APIKeyInfo // key -> user info mapping
	schemes   []*types.APIKeySecurityScheme
}

// APIKeyInfo represents information about an API key
type APIKeyInfo struct {
	UserID   string            `json:"userId"`
	Name     string            `json:"name"`
	Email    string            `json:"email"`
	Scopes   []string          `json:"scopes"`
	Metadata map[string]string `json:"metadata"`
}

// APIKeyAuthenticatorConfig represents configuration for API Key authentication
type APIKeyAuthenticatorConfig struct {
	Name    string                        `json:"name"`
	Keys    map[string]*APIKeyInfo        `json:"keys"` // key -> user info
	Schemes []*types.APIKeySecurityScheme `json:"schemes"`
}

// NewAPIKeyAuthenticator creates a new API Key authenticator
func NewAPIKeyAuthenticator(config *APIKeyAuthenticatorConfig) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		name:      config.Name,
		validKeys: config.Keys,
		schemes:   config.Schemes,
	}
}

// Name returns the name of the authenticator
func (a *APIKeyAuthenticator) Name() string {
	return a.name
}

// Authenticate attempts to authenticate a request using API keys
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, md metadata.MD) (auth.User, error) {
	// Try each configured scheme
	for _, scheme := range a.schemes {
		if !a.SupportsScheme(scheme) {
			continue
		}

		apiKey, found := a.extractAPIKey(md, scheme)
		if !found {
			continue
		}

		// Validate the API key
		keyInfo, valid := a.validKeys[apiKey]
		if !valid {
			return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidCredentials,
				Message: "Invalid API key",
			}
		}

		// Convert metadata to map[string]interface{}
		claims := make(map[string]interface{})
		for k, v := range keyInfo.Metadata {
			claims[k] = v
		}
		claims["auth_method"] = "api_key"
		claims["scheme_name"] = scheme.Name

		return auth.NewAuthenticatedUser(
			keyInfo.UserID,
			keyInfo.Name,
			keyInfo.Email,
			keyInfo.Scopes,
			claims,
		), nil
	}

	return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
		Code:    auth.ErrCodeMissingCredentials,
		Message: "No valid API key found",
	}
}

// SupportsScheme returns true if this authenticator supports API Key schemes
func (a *APIKeyAuthenticator) SupportsScheme(scheme types.SecurityScheme) bool {
	return scheme.SecuritySchemeType() == types.SecuritySchemeTypeAPIKey
}

// extractAPIKey extracts the API key from metadata based on the scheme configuration
func (a *APIKeyAuthenticator) extractAPIKey(md metadata.MD, scheme *types.APIKeySecurityScheme) (string, bool) {
	switch scheme.Location {
	case "header":
		headerName := strings.ToLower(scheme.Name)
		values := md.Get(headerName)
		if len(values) > 0 && values[0] != "" {
			return values[0], true
		}

		// Also try the authorization header for common patterns
		if headerName == "authorization" || headerName == "x-api-key" {
			authHeaders := md.Get("authorization")
			for _, authHeader := range authHeaders {
				if strings.HasPrefix(authHeader, "ApiKey ") {
					return strings.TrimPrefix(authHeader, "ApiKey "), true
				}
				if strings.HasPrefix(authHeader, "Bearer ") && headerName == "authorization" {
					// Some APIs use Bearer tokens as API keys
					return strings.TrimPrefix(authHeader, "Bearer "), true
				}
			}
		}

	case "query":
		// gRPC doesn't directly support query parameters, but they might be passed
		// in a special metadata field
		queryMD := md.Get("grpc-query-" + scheme.Name)
		if len(queryMD) > 0 && queryMD[0] != "" {
			return queryMD[0], true
		}

	case "cookie":
		// Extract from cookie header
		cookieHeaders := md.Get("cookie")
		for _, cookieHeader := range cookieHeaders {
			cookies := parseCookies(cookieHeader)
			if value, exists := cookies[scheme.Name]; exists {
				return value, true
			}
		}
	}

	return "", false
}

// parseCookies parses a cookie header string and returns a map of cookie names to values
func parseCookies(cookieHeader string) map[string]string {
	cookies := make(map[string]string)
	pairs := strings.Split(cookieHeader, ";")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			cookies[name] = value
		}
	}

	return cookies
}

// AddAPIKey adds a new API key to the authenticator
func (a *APIKeyAuthenticator) AddAPIKey(key string, info *APIKeyInfo) {
	if a.validKeys == nil {
		a.validKeys = make(map[string]*APIKeyInfo)
	}
	a.validKeys[key] = info
}

// RemoveAPIKey removes an API key from the authenticator
func (a *APIKeyAuthenticator) RemoveAPIKey(key string) {
	delete(a.validKeys, key)
}

// ListAPIKeys returns all configured API keys (without the actual key values)
func (a *APIKeyAuthenticator) ListAPIKeys() []*APIKeyInfo {
	var keys []*APIKeyInfo
	for _, info := range a.validKeys {
		keys = append(keys, info)
	}
	return keys
}

// ValidateAPIKey checks if an API key is valid
func (a *APIKeyAuthenticator) ValidateAPIKey(key string) (*APIKeyInfo, bool) {
	info, exists := a.validKeys[key]
	return info, exists
}
