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
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"seata-go-ai-a2a/pkg/auth/internal"
	"seata-go-ai-a2a/pkg/types"
)

// AuthenticationInterceptor provides comprehensive gRPC authentication
type AuthenticationInterceptor struct {
	authenticators       []Authenticator
	exemptMethods        []string            // Methods that don't require authentication
	methodAuthenticators map[string][]string // Method -> authenticator names mapping
	rateLimiter          RateLimiter         // Optional rate limiting
	auditLogger          AuditLogger         // Optional audit logging
	config               *InterceptorConfig
}

// InterceptorConfig configures the authentication interceptor
type InterceptorConfig struct {
	RequireAuthentication bool                `json:"requireAuthentication"` // Default: true
	ExemptMethods         []string            `json:"exemptMethods"`         // Methods that don't require auth
	MethodAuthenticators  map[string][]string `json:"methodAuthenticators"`  // Method-specific authenticators
	EnableRateLimiting    bool                `json:"enableRateLimiting"`    // Default: false
	EnableAuditLogging    bool                `json:"enableAuditLogging"`    // Default: false
	MaxRequestsPerSecond  int                 `json:"maxRequestsPerSecond"`  // Default: 100
	MaxRequestsPerMinute  int                 `json:"maxRequestsPerMinute"`  // Default: 1000
	RateLimitWindow       time.Duration       `json:"rateLimitWindow"`       // Default: 1 minute
	FailureThreshold      int                 `json:"failureThreshold"`      // Default: 10
	FailureWindow         time.Duration       `json:"failureWindow"`         // Default: 5 minutes
	BanDuration           time.Duration       `json:"banDuration"`           // Default: 15 minutes
}

// AuthenticationResult represents the result of authentication
type AuthenticationResult struct {
	User          User                   `json:"user"`
	Authenticator string                 `json:"authenticator"`
	AuthTime      time.Time              `json:"authTime"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// NewAuthenticationInterceptor creates a new authentication interceptor
func NewAuthenticationInterceptor(config *InterceptorConfig, authenticators ...Authenticator) *AuthenticationInterceptor {
	if config == nil {
		config = &InterceptorConfig{
			RequireAuthentication: true,
			MaxRequestsPerSecond:  100,
			MaxRequestsPerMinute:  1000,
			RateLimitWindow:       time.Minute,
			FailureThreshold:      10,
			FailureWindow:         5 * time.Minute,
			BanDuration:           15 * time.Minute,
		}
	}

	interceptor := &AuthenticationInterceptor{
		authenticators:       authenticators,
		exemptMethods:        config.ExemptMethods,
		methodAuthenticators: config.MethodAuthenticators,
		config:               config,
	}

	// Initialize rate limiter if enabled
	if config.EnableRateLimiting {
		rateLimiterConfig := &internal.RateLimiterConfig{
			MaxRequestsPerSecond: config.MaxRequestsPerSecond,
			MaxRequestsPerMinute: config.MaxRequestsPerMinute,
			Window:               config.RateLimitWindow,
			FailureThreshold:     config.FailureThreshold,
			FailureWindow:        config.FailureWindow,
			BanDuration:          config.BanDuration,
		}
		interceptor.rateLimiter = internal.NewDefaultRateLimiter(rateLimiterConfig)
	}

	// Initialize audit logger if enabled
	if config.EnableAuditLogging {
		auditLoggerConfig := &internal.AuditLoggerConfig{
			Enabled:       true,
			BufferSize:    1000,
			FlushInterval: 10 * time.Second,
		}
		interceptor.auditLogger = internal.NewDefaultAuditLogger(&types.DefaultLogger{}, auditLoggerConfig)
	}

	return interceptor
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for authentication
func (a *AuthenticationInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if method is exempt from authentication
		if a.isMethodExempt(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract client identifier for rate limiting
		clientID := a.extractClientIdentifier(ctx)

		// Check rate limiting
		if a.rateLimiter != nil {
			allowed := a.rateLimiter.Allow(ctx, clientID)
			if !allowed {
				if a.auditLogger != nil {
					event := &internal.AuthFailureEvent{
						AuthAttemptEvent: internal.AuthAttemptEvent{
							Timestamp:  time.Now(),
							Method:     "rate_limit",
							RemoteAddr: clientID,
						},
						Reason: "rate limit exceeded",
					}
					a.auditLogger.LogAuthFailure(ctx, event)
				}
				return nil, status.Error(codes.ResourceExhausted, "Rate limit exceeded")
			}
		}

		// Perform authentication
		authResult, err := a.authenticate(ctx, info.FullMethod)
		if err != nil {
			// Record authentication failure for rate limiting
			if a.rateLimiter != nil {
				a.rateLimiter.RecordFailure(ctx, clientID, time.Since(time.Now()))
			}

			// Log authentication failure
			if a.auditLogger != nil {
				var authErr *AuthenticationError
				if authError, ok := err.(*AuthenticationError); ok {
					authErr = authError
				} else {
					authErr = &AuthenticationError{
						Code:    ErrCodeInvalidCredentials,
						Message: err.Error(),
					}
				}

				event := &internal.AuthFailureEvent{
					AuthAttemptEvent: internal.AuthAttemptEvent{
						Timestamp:  time.Now(),
						Method:     info.FullMethod,
						RemoteAddr: clientID,
					},
					Reason: authErr.Code.String(),
					Error:  authErr.Message,
				}
				a.auditLogger.LogAuthFailure(ctx, event)
			}

			return nil, a.convertToGRPCError(err)
		}

		// Add authenticated user to context
		ctx = context.WithValue(ctx, UserContextKey, authResult.User)
		ctx = context.WithValue(ctx, AuthResultContextKey, authResult)

		// Log successful authentication
		if a.auditLogger != nil {
			event := &internal.AuthSuccessEvent{
				AuthAttemptEvent: internal.AuthAttemptEvent{
					Timestamp:  time.Now(),
					Method:     info.FullMethod,
					UserID:     authResult.User.GetUserID(),
					RemoteAddr: clientID,
				},
				Scopes: authResult.User.GetScopes(),
			}
			a.auditLogger.LogAuthSuccess(ctx, event)
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for authentication
func (a *AuthenticationInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if method is exempt from authentication
		if a.isMethodExempt(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		clientID := a.extractClientIdentifier(ctx)

		// Check rate limiting
		if a.rateLimiter != nil {
			allowed := a.rateLimiter.Allow(ctx, clientID)
			if !allowed {
				if a.auditLogger != nil {
					event := &internal.AuthFailureEvent{
						AuthAttemptEvent: internal.AuthAttemptEvent{
							Timestamp:  time.Now(),
							Method:     "rate_limit",
							RemoteAddr: clientID,
						},
						Reason: "rate limit exceeded",
					}
					a.auditLogger.LogAuthFailure(ctx, event)
				}
				return status.Error(codes.ResourceExhausted, "Rate limit exceeded")
			}
		}

		// Perform authentication
		authResult, err := a.authenticate(ctx, info.FullMethod)
		if err != nil {
			// Record authentication failure for rate limiting
			if a.rateLimiter != nil {
				a.rateLimiter.RecordFailure(ctx, clientID, time.Since(time.Now()))
			}

			// Log authentication failure
			if a.auditLogger != nil {
				var authErr *AuthenticationError
				if authError, ok := err.(*AuthenticationError); ok {
					authErr = authError
				} else {
					authErr = &AuthenticationError{
						Code:    ErrCodeInvalidCredentials,
						Message: err.Error(),
					}
				}

				event := &internal.AuthFailureEvent{
					AuthAttemptEvent: internal.AuthAttemptEvent{
						Timestamp:  time.Now(),
						Method:     info.FullMethod,
						RemoteAddr: clientID,
					},
					Reason: authErr.Code.String(),
					Error:  authErr.Message,
				}
				a.auditLogger.LogAuthFailure(ctx, event)
			}

			return a.convertToGRPCError(err)
		}

		// Create wrapped stream with authenticated context
		ctx = context.WithValue(ctx, UserContextKey, authResult.User)
		ctx = context.WithValue(ctx, AuthResultContextKey, authResult)

		wrappedStream := &authenticatedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Log successful authentication
		if a.auditLogger != nil {
			event := &internal.AuthSuccessEvent{
				AuthAttemptEvent: internal.AuthAttemptEvent{
					Timestamp:  time.Now(),
					Method:     info.FullMethod,
					UserID:     authResult.User.GetUserID(),
					RemoteAddr: clientID,
				},
				Scopes: authResult.User.GetScopes(),
			}
			a.auditLogger.LogAuthSuccess(ctx, event)
		}

		return handler(srv, wrappedStream)
	}
}

// authenticate performs authentication using configured authenticators
func (a *AuthenticationInterceptor) authenticate(ctx context.Context, method string) (*AuthenticationResult, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, &AuthenticationError{
			Code:    ErrCodeMissingCredentials,
			Message: "Missing gRPC metadata",
		}
	}

	// Determine which authenticators to try for this method
	authenticatorsToTry := a.getAuthenticatorsForMethod(method)
	if len(authenticatorsToTry) == 0 {
		authenticatorsToTry = a.authenticators
	}

	var lastError error

	// Try each authenticator
	for _, authenticator := range authenticatorsToTry {
		user, err := authenticator.Authenticate(ctx, md)
		if err != nil {
			lastError = err
			continue
		}

		if user.IsAuthenticated() {
			return &AuthenticationResult{
				User:          user,
				Authenticator: authenticator.Name(),
				AuthTime:      time.Now(),
				Metadata: map[string]interface{}{
					"authenticator_name": authenticator.Name(),
					"method":             method,
				},
			}, nil
		}
	}

	if lastError != nil {
		return nil, lastError
	}

	return nil, &AuthenticationError{
		Code:    ErrCodeInvalidCredentials,
		Message: "Authentication failed with all configured authenticators",
	}
}

// isMethodExempt checks if a method is exempt from authentication
func (a *AuthenticationInterceptor) isMethodExempt(method string) bool {
	if !a.config.RequireAuthentication {
		return true
	}

	for _, exemptMethod := range a.exemptMethods {
		if exemptMethod == method || strings.HasPrefix(method, exemptMethod) {
			return true
		}
	}
	return false
}

// getAuthenticatorsForMethod returns authenticators configured for a specific method
func (a *AuthenticationInterceptor) getAuthenticatorsForMethod(method string) []Authenticator {
	authenticatorNames, exists := a.methodAuthenticators[method]
	if !exists {
		return nil
	}

	var result []Authenticator
	for _, name := range authenticatorNames {
		for _, auth := range a.authenticators {
			if auth.Name() == name {
				result = append(result, auth)
				break
			}
		}
	}
	return result
}

// extractClientIdentifier extracts a client identifier for rate limiting
func (a *AuthenticationInterceptor) extractClientIdentifier(ctx context.Context) string {
	// Try to extract from peer info first
	if peer, ok := ctx.Value("peer").(interface {
		Addr() interface{ String() string }
	}); ok {
		return peer.Addr().String()
	}

	// Fallback to extracting from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "unknown"
	}

	// Try common identification headers
	if values := md.Get("x-forwarded-for"); len(values) > 0 {
		return values[0]
	}

	if values := md.Get("x-real-ip"); len(values) > 0 {
		return values[0]
	}

	if values := md.Get("user-agent"); len(values) > 0 {
		return fmt.Sprintf("ua:%s", values[0])
	}

	return "anonymous"
}

// convertToGRPCError converts authentication errors to gRPC status codes
func (a *AuthenticationInterceptor) convertToGRPCError(err error) error {
	var authErr *AuthenticationError
	if errors.As(err, &authErr) {
		switch authErr.Code {
		case ErrCodeMissingCredentials:
			return status.Error(codes.Unauthenticated, authErr.Message)
		case ErrCodeInvalidCredentials, ErrCodeInvalidToken:
			return status.Error(codes.Unauthenticated, authErr.Message)
		case ErrCodeExpiredToken:
			return status.Error(codes.Unauthenticated, "Token expired")
		case ErrCodeJWKSFetchFailed:
			return status.Error(codes.Unavailable, "Authentication service unavailable")
		default:
			return status.Error(codes.Internal, "Authentication error")
		}
	}
	return status.Error(codes.Internal, "Internal authentication error")
}

// authenticatedServerStream wraps grpc.ServerStream with authenticated context
type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}

// Context keys for storing authentication information
type contextKey string

const (
	UserContextKey       contextKey = "authenticated_user"
	AuthResultContextKey contextKey = "auth_result"
)

// GetUserFromContext extracts the authenticated user from context
func GetUserFromContext(ctx context.Context) (User, bool) {
	if user, ok := ctx.Value(UserContextKey).(User); ok {
		return user, true
	}
	return &UnauthenticatedUser{}, false
}

// GetAuthResultFromContext extracts the authentication result from context
func GetAuthResultFromContext(ctx context.Context) (*AuthenticationResult, bool) {
	if result, ok := ctx.Value(AuthResultContextKey).(*AuthenticationResult); ok {
		return result, true
	}
	return nil, false
}

// RequireUser is a helper function to get authenticated user or return error
func RequireUser(ctx context.Context) (User, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok || !user.IsAuthenticated() {
		return nil, status.Error(codes.Unauthenticated, "Authentication required")
	}
	return user, nil
}

// RequireScope checks if the authenticated user has the required scope
func RequireScope(ctx context.Context, requiredScope string) error {
	user, err := RequireUser(ctx)
	if err != nil {
		return err
	}

	for _, scope := range user.GetScopes() {
		if scope == requiredScope {
			return nil
		}
	}

	return status.Errorf(codes.PermissionDenied, "Required scope '%s' not found", requiredScope)
}

// RequireAnyScope checks if the authenticated user has at least one of the required scopes
func RequireAnyScope(ctx context.Context, requiredScopes ...string) error {
	user, err := RequireUser(ctx)
	if err != nil {
		return err
	}

	userScopes := user.GetScopes()
	for _, requiredScope := range requiredScopes {
		for _, userScope := range userScopes {
			if userScope == requiredScope {
				return nil
			}
		}
	}

	return status.Errorf(codes.PermissionDenied, "None of the required scopes found: %v", requiredScopes)
}

// RequireAllScopes checks if the authenticated user has all the required scopes
func RequireAllScopes(ctx context.Context, requiredScopes ...string) error {
	user, err := RequireUser(ctx)
	if err != nil {
		return err
	}

	userScopes := user.GetScopes()
	for _, requiredScope := range requiredScopes {
		found := false
		for _, userScope := range userScopes {
			if userScope == requiredScope {
				found = true
				break
			}
		}
		if !found {
			return status.Errorf(codes.PermissionDenied, "Required scope '%s' not found", requiredScope)
		}
	}

	return nil
}

// Add String method to ErrorCode for audit logging
func (e ErrorCode) String() string {
	return string(e)
}
