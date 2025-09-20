package auth

import (
	"context"
	"net/http"
	"strings"

	"agenthub/pkg/common"
	"agenthub/pkg/utils"
)

// Middleware provides HTTP authentication middleware following K8s patterns
type Middleware struct {
	jwtManager *JWTManager
	optional   bool
	logger     *utils.Logger
	writer     *common.HTTPResponseWriter
}

// MiddlewareConfig holds middleware configuration
type MiddlewareConfig struct {
	JWTManager *JWTManager
	Optional   bool
	Logger     *utils.Logger
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(config MiddlewareConfig) *Middleware {
	logger := config.Logger
	if logger == nil {
		logger = utils.WithField("component", "auth-middleware")
	}
	
	return &Middleware{
		jwtManager: config.JWTManager,
		optional:   config.Optional,
		logger:     logger,
		writer:     common.NewHTTPResponseWriter(),
	}
}

// Handler returns an HTTP middleware handler
func (m *Middleware) Handler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := m.extractToken(r)
		
		if token == "" {
			if m.optional {
				// Optional authentication, continue processing request
				m.logger.Debug("No token provided, continuing with optional auth")
				next(w, r)
				return
			}
			m.logger.Warn("Missing authorization token")
			m.writer.WriteUnauthorized(w, "Missing authorization token")
			return
		}

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			m.logger.Error("Token validation failed: %v", err)
			if m.optional {
				// Optional authentication, continue even with invalid token
				m.logger.Debug("Invalid token, continuing with optional auth")
				next(w, r)
				return
			}
			m.writer.WriteUnauthorized(w, "Invalid token")
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), common.ClaimsContextKey, claims)
		m.logger.Debug("Token validated successfully for user: %s", claims.Username)
		next(w, r.WithContext(ctx))
	}
}

// RequireRole returns a middleware that requires specific roles
func (m *Middleware) RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.Handler(func(w http.ResponseWriter, r *http.Request) {
			claims := common.GetClaimsFromContext(r.Context())
			if claims == nil {
				m.logger.Warn("Authentication required but no claims found")
				m.writer.WriteForbidden(w, "Authentication required")
				return
			}

			if !claims.HasAnyRole(roles...) {
				m.logger.Warn("Insufficient permissions for user %s, required roles: %v, has roles: %v", 
					claims.Username, roles, claims.Roles)
				m.writer.WriteForbidden(w, "Insufficient permissions")
				return
			}

			m.logger.Debug("Role check passed for user %s", claims.Username)
			next(w, r)
		})
	}
}

// RequireAllRoles returns a middleware that requires all specified roles
func (m *Middleware) RequireAllRoles(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.Handler(func(w http.ResponseWriter, r *http.Request) {
			claims := common.GetClaimsFromContext(r.Context())
			if claims == nil {
				m.writer.WriteForbidden(w, "Authentication required")
				return
			}

			for _, role := range roles {
				if !claims.HasRole(role) {
					m.logger.Warn("Missing required role %s for user %s", role, claims.Username)
					m.writer.WriteForbidden(w, "Insufficient permissions")
					return
				}
			}

			m.logger.Debug("All roles check passed for user %s", claims.Username)
			next(w, r)
		})
	}
}

// extractToken extracts JWT token from request
func (m *Middleware) extractToken(r *http.Request) string {
	// Extract from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Try to extract from query parameter
		return r.URL.Query().Get("token")
	}

	// Support "Bearer <token>" format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}

	// Return header value directly
	return authHeader
}

// GetOptional returns whether the middleware is optional
func (m *Middleware) GetOptional() bool {
	return m.optional
}

// SetOptional sets whether the middleware is optional
func (m *Middleware) SetOptional(optional bool) {
	m.optional = optional
}