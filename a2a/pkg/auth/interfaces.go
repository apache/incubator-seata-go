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

package auth

import (
	"context"
	"crypto"
	"time"

	"seata-go-ai-a2a/pkg/types"

	"google.golang.org/grpc/metadata"
)

// User represents an authenticated user
type User interface {
	// GetUserID returns the unique user ID
	GetUserID() string

	// GetName returns the user's display name
	GetName() string

	// GetEmail returns the user's email address
	GetEmail() string

	// GetScopes returns the user's authorized scopes
	GetScopes() []string

	// GetClaims returns additional user claims
	GetClaims() map[string]interface{}

	// IsAuthenticated returns true if the user is authenticated
	IsAuthenticated() bool
}

// Authenticator defines the interface for authentication providers
type Authenticator interface {
	// Name returns the name of this authenticator
	Name() string

	// Authenticate attempts to authenticate a request
	Authenticate(ctx context.Context, md metadata.MD) (User, error)

	// SupportsScheme returns true if this authenticator supports the given security scheme
	SupportsScheme(scheme types.SecurityScheme) bool
}

// JWSValidator validates JSON Web Signatures for Agent Cards
type JWSValidator interface {
	// ValidateSignature validates a JWS signature for the given payload
	ValidateSignature(ctx context.Context, payload []byte, signature *types.AgentCardSignature) error
}

// JWSSigner signs Agent Cards using JWS
type JWSSigner interface {
	// SignAgentCard signs an Agent Card and returns the signature
	SignAgentCard(ctx context.Context, agentCard *types.AgentCard, privateKey crypto.PrivateKey, keyID, jwksURL string) (*types.AgentCardSignature, error)
}

// TokenValidator validates JWT tokens
type TokenValidator interface {
	// ValidateToken validates a JWT token and returns the claims
	ValidateToken(ctx context.Context, tokenString string) (map[string]interface{}, error)
}

// RateLimiter defines rate limiting interface
type RateLimiter interface {
	// Allow returns true if the request is allowed
	Allow(ctx context.Context, key string) bool

	// RecordFailure records a failed request
	RecordFailure(ctx context.Context, key string, duration time.Duration)

	// Reset resets the rate limiter for a specific key
	Reset(ctx context.Context, key string) error

	// Close cleans up resources
	Close() error
}

// AuditLogger defines audit logging interface
type AuditLogger interface {
	// LogAuthAttempt logs an authentication attempt
	LogAuthAttempt(ctx context.Context, event interface{})

	// LogAuthFailure logs an authentication failure
	LogAuthFailure(ctx context.Context, event interface{})

	// LogAuthSuccess logs a successful authentication
	LogAuthSuccess(ctx context.Context, event interface{})

	// Close cleans up resources
	Close() error
}

// ErrorCode represents authentication error codes
type ErrorCode string

// Authentication error codes
const (
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeMissingCredentials = "MISSING_CREDENTIALS"
	ErrCodeInvalidToken       = "INVALID_TOKEN"
	ErrCodeExpiredToken       = "EXPIRED_TOKEN"
	ErrCodeInvalidSignature   = "INVALID_SIGNATURE"
	ErrCodeJWKSFetchFailed    = "JWKS_FETCH_FAILED"
	ErrCodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
	ErrCodeInternalError      = "INTERNAL_ERROR"
)

// AuthenticationError represents an authentication error
type AuthenticationError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

func (e *AuthenticationError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}
