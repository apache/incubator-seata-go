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
	"net/http"
	"strings"
	"time"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/auth/jws"
	"seata-go-ai-a2a/pkg/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthMiddleware provides authentication middleware for HTTP handlers
type AuthMiddleware struct {
	authManager  *AuthManager
	jwsValidator *jws.Validator
	optional     bool
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authManager *AuthManager, jwsValidator *jws.Validator, optional bool) *AuthMiddleware {
	return &AuthMiddleware{
		authManager:  authManager,
		jwsValidator: jwsValidator,
		optional:     optional,
	}
}

// HTTPMiddleware returns an HTTP middleware function
func (am *AuthMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Convert HTTP headers to gRPC metadata
			md := metadata.New(nil)
			for name, values := range r.Header {
				for _, value := range values {
					md.Append(strings.ToLower(name), value)
				}
			}

			// Attempt authentication
			user, err := am.authManager.Authenticate(r.Context(), md)
			if err != nil && !am.optional {
				am.writeAuthError(w, err)
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), "user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GRPCUnaryInterceptor returns a gRPC unary server interceptor for authentication
func (am *AuthMiddleware) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Attempt authentication
		user, err := am.authManager.Authenticate(ctx, md)
		if err != nil && !am.optional {
			return nil, am.convertAuthError(err)
		}

		// Add user to context
		ctx = context.WithValue(ctx, "user", user)

		return handler(ctx, req)
	}
}

// GRPCStreamInterceptor returns a gRPC stream server interceptor for authentication
func (am *AuthMiddleware) GRPCStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			md = metadata.New(nil)
		}

		// Attempt authentication
		user, err := am.authManager.Authenticate(ss.Context(), md)
		if err != nil && !am.optional {
			return am.convertAuthError(err)
		}

		// Add user to context
		ctx := context.WithValue(ss.Context(), "user", user)
		wrappedStream := &wrappedServerStream{ss, ctx}

		return handler(srv, wrappedStream)
	}
}

// writeAuthError writes an authentication error as HTTP response
func (am *AuthMiddleware) writeAuthError(w http.ResponseWriter, err error) {
	if authErr, ok := err.(*auth.AuthenticationError); ok {
		statusCode := http.StatusUnauthorized

		switch authErr.Code {
		case auth.ErrCodeMissingCredentials:
			statusCode = http.StatusUnauthorized
		case auth.ErrCodeInvalidCredentials:
			statusCode = http.StatusUnauthorized
		case auth.ErrCodeInvalidToken:
			statusCode = http.StatusUnauthorized
		case auth.ErrCodeExpiredToken:
			statusCode = http.StatusUnauthorized
		case auth.ErrCodeRateLimitExceeded:
			statusCode = http.StatusTooManyRequests
		default:
			statusCode = http.StatusInternalServerError
		}

		types.WriteErrorResponse(w, statusCode, string(authErr.Code), authErr.Message, authErr.Details)
	} else {
		types.WriteErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Authentication failed", err.Error())
	}
}

// convertAuthError converts authentication errors to gRPC status errors
func (am *AuthMiddleware) convertAuthError(err error) error {
	if authErr, ok := err.(*auth.AuthenticationError); ok {
		code := codes.Unauthenticated

		switch authErr.Code {
		case auth.ErrCodeMissingCredentials:
			code = codes.Unauthenticated
		case auth.ErrCodeInvalidCredentials:
			code = codes.Unauthenticated
		case auth.ErrCodeInvalidToken:
			code = codes.Unauthenticated
		case auth.ErrCodeExpiredToken:
			code = codes.Unauthenticated
		case auth.ErrCodeRateLimitExceeded:
			code = codes.ResourceExhausted
		default:
			code = codes.Internal
		}

		return status.Error(code, authErr.Message)
	}

	return status.Error(codes.Internal, "Authentication failed")
}

// wrappedServerStream wraps a ServerStream to override context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// JWSMiddleware provides JWS validation middleware for Agent Cards
type JWSMiddleware struct {
	validator       *jws.Validator
	trustedJWKSURLs []string
	required        bool
}

// NewJWSMiddleware creates a new JWS validation middleware
func NewJWSMiddleware(validator *jws.Validator, trustedJWKSURLs []string, required bool) *JWSMiddleware {
	return &JWSMiddleware{
		validator:       validator,
		trustedJWKSURLs: trustedJWKSURLs,
		required:        required,
	}
}

// ValidateAgentCard validates the JWS signature of an agent card
func (jm *JWSMiddleware) ValidateAgentCard(agentCard *types.AgentCard) error {
	if agentCard == nil {
		return fmt.Errorf("agent card is nil")
	}

	if len(agentCard.Signatures) == 0 {
		if jm.required {
			return fmt.Errorf("agent card signature is required")
		}
		return nil // Optional validation, no signature present
	}

	// Create a copy without signatures for validation
	cardCopy := *agentCard
	cardCopy.Signatures = nil

	payload, err := json.Marshal(cardCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal agent card for validation: %w", err)
	}

	// Validate each signature
	for i, signature := range agentCard.Signatures {
		// If we have trusted JWKS URLs, validate against them
		if len(jm.trustedJWKSURLs) > 0 {
			if err := jm.validateSignatureWithTrustedKeys(payload, signature); err != nil {
				return fmt.Errorf("signature %d validation failed: %w", i, err)
			}
		} else {
			// Standard JWS validation (fetches keys from JKU header)
			if err := jm.validator.ValidateSignature(context.Background(), payload, signature); err != nil {
				return fmt.Errorf("signature %d validation failed: %w", i, err)
			}
		}
	}

	return nil
}

// validateSignatureWithTrustedKeys validates a signature against trusted JWKS URLs only
func (jm *JWSMiddleware) validateSignatureWithTrustedKeys(payload []byte, signature *types.AgentCardSignature) error {
	// Decode protected header to get the JKU
	protectedBytes, err := json.RawMessage(signature.Protected).MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to decode protected header: %w", err)
	}

	var header struct {
		JKU string `json:"jku"`
	}
	if err := json.Unmarshal(protectedBytes, &header); err != nil {
		return fmt.Errorf("failed to parse protected header: %w", err)
	}

	// Check if the JKU is in our trusted list
	trusted := false
	for _, trustedURL := range jm.trustedJWKSURLs {
		if header.JKU == trustedURL {
			trusted = true
			break
		}
	}

	if !trusted {
		return fmt.Errorf("JKU %s is not in the trusted JWKS URLs list", header.JKU)
	}

	// Proceed with standard validation
	return jm.validator.ValidateSignature(context.Background(), payload, signature)
}

// RateLimitMiddleware provides rate limiting middleware
type RateLimitMiddleware struct {
	limiter auth.RateLimiter
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(limiter auth.RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// HTTPMiddleware returns an HTTP middleware function for rate limiting
func (rl *RateLimitMiddleware) HTTPMiddleware(keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	if keyFunc == nil {
		keyFunc = func(r *http.Request) string {
			// Default: use IP address as key
			return r.RemoteAddr
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			if !rl.limiter.Allow(r.Context(), key) {
				types.WriteErrorResponse(w, http.StatusTooManyRequests,
					string(auth.ErrCodeRateLimitExceeded),
					"Rate limit exceeded",
					"Too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GRPCUnaryInterceptor returns a gRPC unary server interceptor for rate limiting
func (rl *RateLimitMiddleware) GRPCUnaryInterceptor(keyFunc func(context.Context) string) grpc.UnaryServerInterceptor {
	if keyFunc == nil {
		keyFunc = func(ctx context.Context) string {
			// Default: try to get peer address
			if peer, ok := metadata.FromIncomingContext(ctx); ok {
				if values := peer.Get("x-forwarded-for"); len(values) > 0 {
					return values[0]
				}
			}
			return "unknown"
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		key := keyFunc(ctx)

		if !rl.limiter.Allow(ctx, key) {
			return nil, status.Error(codes.ResourceExhausted, "Rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// AuditMiddleware provides audit logging middleware
type AuditMiddleware struct {
	logger auth.AuditLogger
}

// NewAuditMiddleware creates a new audit logging middleware
func NewAuditMiddleware(logger auth.AuditLogger) *AuditMiddleware {
	return &AuditMiddleware{
		logger: logger,
	}
}

// HTTPMiddleware returns an HTTP middleware function for audit logging
func (al *AuditMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status
			wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapper, r)

			// Log the request
			event := map[string]interface{}{
				"timestamp":   start,
				"method":      r.Method,
				"path":        r.URL.Path,
				"status_code": wrapper.statusCode,
				"duration_ms": time.Since(start).Milliseconds(),
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.UserAgent(),
			}

			// Add user info if available
			if user := r.Context().Value("user"); user != nil {
				if authUser, ok := user.(auth.User); ok {
					event["user_id"] = authUser.GetUserID()
					event["user_name"] = authUser.GetName()
				}
			}

			if wrapper.statusCode >= 400 {
				al.logger.LogAuthFailure(r.Context(), event)
			} else {
				al.logger.LogAuthSuccess(r.Context(), event)
			}
		})
	}
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MiddlewareChain provides a convenient way to chain multiple middlewares
type MiddlewareChain struct {
	middlewares []func(http.Handler) http.Handler
}

// NewMiddlewareChain creates a new middleware chain
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}

// Add adds a middleware to the chain
func (mc *MiddlewareChain) Add(middleware func(http.Handler) http.Handler) *MiddlewareChain {
	mc.middlewares = append(mc.middlewares, middleware)
	return mc
}

// Build builds the middleware chain and returns the final handler
func (mc *MiddlewareChain) Build(handler http.Handler) http.Handler {
	// Apply middlewares in reverse order so they execute in the correct order
	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		handler = mc.middlewares[i](handler)
	}
	return handler
}

// Helper functions for common middleware setups

// DefaultSecurityMiddleware creates a default security middleware chain
func DefaultSecurityMiddleware(authManager *AuthManager, jwsValidator *jws.Validator, rateLimiter auth.RateLimiter, auditLogger auth.AuditLogger) *MiddlewareChain {
	chain := NewMiddlewareChain()

	// CORS middleware (allow all origins by default)
	chain.Add(CORSMiddleware(nil))

	// Rate limiting
	if rateLimiter != nil {
		chain.Add(NewRateLimitMiddleware(rateLimiter).HTTPMiddleware(nil))
	}

	// Authentication
	if authManager != nil {
		chain.Add(NewAuthMiddleware(authManager, jwsValidator, false).HTTPMiddleware())
	}

	// Audit logging
	if auditLogger != nil {
		chain.Add(NewAuditMiddleware(auditLogger).HTTPMiddleware())
	}

	return chain
}

// OptionalSecurityMiddleware creates an optional security middleware chain (for public endpoints)
func OptionalSecurityMiddleware(authManager *AuthManager, jwsValidator *jws.Validator, auditLogger auth.AuditLogger) *MiddlewareChain {
	chain := NewMiddlewareChain()

	// CORS middleware
	chain.Add(CORSMiddleware(nil))

	// Optional authentication (doesn't fail if no auth provided)
	if authManager != nil {
		chain.Add(NewAuthMiddleware(authManager, jwsValidator, true).HTTPMiddleware())
	}

	// Audit logging
	if auditLogger != nil {
		chain.Add(NewAuditMiddleware(auditLogger).HTTPMiddleware())
	}

	return chain
}

// GetUserFromContext extracts the authenticated user from the request context
func GetUserFromContext(ctx context.Context) (auth.User, bool) {
	user := ctx.Value("user")
	if user == nil {
		return &auth.UnauthenticatedUser{}, false
	}

	if authUser, ok := user.(auth.User); ok {
		return authUser, authUser.IsAuthenticated()
	}

	return &auth.UnauthenticatedUser{}, false
}
