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

package common

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey represents type-safe context keys
type contextKey string

const (
	// ClaimsContextKey is used to store JWT claims in request context
	ClaimsContextKey contextKey = "claims"
)

// ErrorResponse represents standard error response format
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    int                    `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Response represents standard response wrapper
type Response struct {
	Success bool           `json:"success"`
	Message string         `json:"message,omitempty"`
	Data    interface{}    `json:"data,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

// Claims represents JWT claims structure following K8s patterns
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// HasRole checks if the claims contain a specific role
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the claims contain any of the specified roles
func (c *Claims) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if c.HasRole(role) {
			return true
		}
	}
	return false
}

// GetClaimsFromContext extracts JWT claims from request context
func GetClaimsFromContext(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(ClaimsContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

// Security Schemes interfaces following OpenAPI patterns
type SecurityScheme interface {
	GetType() string
	Validate() error
}

// APIKeySecurityScheme represents API key authentication
type APIKeySecurityScheme struct {
	Type string `json:"type"`
	In   string `json:"in"`
	Name string `json:"name"`
}

func (s *APIKeySecurityScheme) GetType() string { return s.Type }
func (s *APIKeySecurityScheme) Validate() error { return nil }

// HTTPAuthSecurityScheme represents HTTP authentication
type HTTPAuthSecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

func (s *HTTPAuthSecurityScheme) GetType() string { return s.Type }
func (s *HTTPAuthSecurityScheme) Validate() error { return nil }

// OAuth2SecurityScheme represents OAuth2 authentication
type OAuth2SecurityScheme struct {
	Type  string     `json:"type"`
	Flows OAuthFlows `json:"flows"`
}

func (s *OAuth2SecurityScheme) GetType() string { return s.Type }
func (s *OAuth2SecurityScheme) Validate() error { return nil }

// OpenIdConnectSecurityScheme represents OpenID Connect authentication
type OpenIdConnectSecurityScheme struct {
	Type             string `json:"type"`
	OpenIDConnectURL string `json:"openIdConnectUrl"`
}

func (s *OpenIdConnectSecurityScheme) GetType() string { return s.Type }
func (s *OpenIdConnectSecurityScheme) Validate() error { return nil }

// OAuthFlows represents OAuth2 flows
type OAuthFlows struct {
	AuthorizationCode *AuthorizationCodeOAuthFlow `json:"authorizationCode,omitempty"`
	ClientCredentials *ClientCredentialsOAuthFlow `json:"clientCredentials,omitempty"`
	Implicit          *ImplicitOAuthFlow          `json:"implicit,omitempty"`
	Password          *PasswordOAuthFlow          `json:"password,omitempty"`
}

// OAuth flow structures
type AuthorizationCodeOAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl"`
	TokenURL         string            `json:"tokenUrl"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

type ClientCredentialsOAuthFlow struct {
	TokenURL   string            `json:"tokenUrl"`
	RefreshURL string            `json:"refreshUrl,omitempty"`
	Scopes     map[string]string `json:"scopes"`
}

type ImplicitOAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

type PasswordOAuthFlow struct {
	TokenURL   string            `json:"tokenUrl"`
	RefreshURL string            `json:"refreshUrl,omitempty"`
	Scopes     map[string]string `json:"scopes"`
}

// Resource represents base resource structure following K8s patterns
type Resource struct {
	Kind       string      `json:"kind"`
	APIVersion string      `json:"apiVersion"`
	Metadata   ObjectMeta  `json:"metadata"`
	Spec       interface{} `json:"spec,omitempty"`
	Status     interface{} `json:"status,omitempty"`
}

// ObjectMeta represents resource metadata following K8s patterns
type ObjectMeta struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	UID               string            `json:"uid,omitempty"`
	ResourceVersion   string            `json:"resourceVersion,omitempty"`
	Generation        int64             `json:"generation,omitempty"`
	CreationTimestamp *time.Time        `json:"creationTimestamp,omitempty"`
	DeletionTimestamp *time.Time        `json:"deletionTimestamp,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
}

// BaseResource 为存储资源提供通用的基础功能
type BaseResource struct {
	ID        string                 `json:"id"`
	Kind      string                 `json:"kind"`
	Version   string                 `json:"version"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// GetID 返回资源ID
func (r *BaseResource) GetID() string {
	return r.ID
}

// GetKind 返回资源类型
func (r *BaseResource) GetKind() string {
	return r.Kind
}

// GetVersion 返回资源版本
func (r *BaseResource) GetVersion() string {
	return r.Version
}

// GetMetadata 返回资源元数据
func (r *BaseResource) GetMetadata() map[string]interface{} {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	return r.Metadata
}

// SetMetadata 设置元数据值
func (r *BaseResource) SetMetadata(key string, value interface{}) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata[key] = value
}

// GetCreatedAt 返回创建时间
func (r *BaseResource) GetCreatedAt() time.Time {
	return r.CreatedAt
}

// GetUpdatedAt 返回更新时间
func (r *BaseResource) GetUpdatedAt() time.Time {
	return r.UpdatedAt
}
