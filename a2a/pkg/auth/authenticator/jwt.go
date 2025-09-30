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
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"google.golang.org/grpc/metadata"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/types"
)

// JWTAuthenticator implements JWT-based authentication
type JWTAuthenticator struct {
	name      string
	jwksURL   string
	audience  string
	issuer    string
	clockSkew time.Duration
	keySet    jwk.Set
	validator auth.TokenValidator
}

// JWTAuthenticatorConfig represents configuration for JWT authentication
type JWTAuthenticatorConfig struct {
	Name           string        `json:"name"`
	JWKSURL        string        `json:"jwksUrl"`
	Audience       string        `json:"audience,omitempty"`
	Issuer         string        `json:"issuer,omitempty"`
	ClockSkew      time.Duration `json:"clockSkew,omitempty"`
	RequiredClaims []string      `json:"requiredClaims,omitempty"`
}

// NewJWTAuthenticator creates a new JWT authenticator
func NewJWTAuthenticator(config *JWTAuthenticatorConfig) *JWTAuthenticator {
	clockSkew := config.ClockSkew
	if clockSkew == 0 {
		clockSkew = 30 * time.Second // Default clock skew
	}

	return &JWTAuthenticator{
		name:      config.Name,
		jwksURL:   config.JWKSURL,
		audience:  config.Audience,
		issuer:    config.Issuer,
		clockSkew: clockSkew,
		validator: NewDefaultTokenValidator(config.JWKSURL, config.Audience, config.Issuer),
	}
}

// Name returns the name of the authenticator
func (j *JWTAuthenticator) Name() string {
	return j.name
}

// Authenticate attempts to authenticate a request using JWT tokens
func (j *JWTAuthenticator) Authenticate(ctx context.Context, md metadata.MD) (auth.User, error) {
	// Extract authorization header
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeMissingCredentials,
			Message: "Missing authorization header",
		}
	}

	authHeader := authHeaders[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Invalid authorization header format, expected Bearer token",
		}
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Empty bearer token",
		}
	}

	// Validate the token
	claims, err := j.validator.ValidateToken(ctx, token)
	if err != nil {
		return &auth.UnauthenticatedUser{}, err
	}

	// Extract user information from claims
	user, err := j.extractUserFromClaims(claims)
	if err != nil {
		return &auth.UnauthenticatedUser{}, err
	}

	return user, nil
}

// SupportsScheme returns true if this authenticator supports JWT-related schemes
func (j *JWTAuthenticator) SupportsScheme(scheme types.SecurityScheme) bool {
	switch scheme.SecuritySchemeType() {
	case types.SecuritySchemeTypeHTTPAuth:
		if httpScheme, ok := scheme.(*types.HTTPAuthSecurityScheme); ok {
			return strings.ToLower(httpScheme.Scheme) == "bearer"
		}
		return false
	case types.SecuritySchemeTypeOAuth2:
		return true
	case types.SecuritySchemeTypeOpenIDConnect:
		return true
	default:
		return false
	}
}

// extractUserFromClaims extracts user information from JWT claims
func (j *JWTAuthenticator) extractUserFromClaims(claims map[string]interface{}) (auth.User, error) {
	// Extract standard claims
	var userID, name, email string
	var scopes []string

	if sub, ok := claims["sub"].(string); ok {
		userID = sub
	}

	if nameValue, ok := claims["name"].(string); ok {
		name = nameValue
	} else if preferredUsername, ok := claims["preferred_username"].(string); ok {
		name = preferredUsername
	}

	if emailValue, ok := claims["email"].(string); ok {
		email = emailValue
	}

	// Extract scopes
	if scopeValue, ok := claims["scope"]; ok {
		if scopeStr, ok := scopeValue.(string); ok {
			scopes = strings.Fields(scopeStr)
		} else if scopeSlice, ok := scopeValue.([]interface{}); ok {
			for _, s := range scopeSlice {
				if scopeStr, ok := s.(string); ok {
					scopes = append(scopes, scopeStr)
				}
			}
		}
	}

	if userID == "" {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidToken,
			Message: "Missing subject (sub) claim in JWT",
		}
	}

	return auth.NewAuthenticatedUser(userID, name, email, scopes, claims), nil
}

// DefaultTokenValidator implements JWT token validation
type DefaultTokenValidator struct {
	jwksURL  string
	audience string
	issuer   string
	keySet   jwk.Set
	mutex    sync.RWMutex
}

// NewDefaultTokenValidator creates a new token validator
func NewDefaultTokenValidator(jwksURL, audience, issuer string) *DefaultTokenValidator {
	return &DefaultTokenValidator{
		jwksURL:  jwksURL,
		audience: audience,
		issuer:   issuer,
	}
}

// ValidateToken validates a JWT token and returns the claims
func (v *DefaultTokenValidator) ValidateToken(ctx context.Context, tokenString string) (map[string]interface{}, error) {
	// Parse the token without verification first to get the kid
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidToken,
			Message: "Failed to parse JWT token",
			Details: err.Error(),
		}
	}

	// Get the key ID from the header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidToken,
			Message: "Missing key ID (kid) in JWT header",
		}
	}

	// Fetch the key set if not already cached (thread-safe)
	v.mutex.RLock()
	currentKeySet := v.keySet
	v.mutex.RUnlock()

	if currentKeySet == nil {
		// Double-check pattern to avoid race condition
		v.mutex.Lock()
		if v.keySet == nil {
			keySet, err := jwk.Fetch(ctx, v.jwksURL)
			if err != nil {
				v.mutex.Unlock()
				return nil, &auth.AuthenticationError{
					Code:    auth.ErrCodeJWKSFetchFailed,
					Message: "Failed to fetch JWKS",
					Details: err.Error(),
				}
			}
			v.keySet = keySet
		}
		currentKeySet = v.keySet
		v.mutex.Unlock()
	}

	// Find the key
	key, ok := currentKeySet.LookupKeyID(kid)
	if !ok {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidToken,
			Message: "Key not found in JWKS",
			Details: "Key ID: " + kid,
		}
	}

	// Get the raw key
	var rawKey interface{}
	if err := key.Raw(&rawKey); err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidToken,
			Message: "Failed to extract raw key",
			Details: err.Error(),
		}
	}

	// Parse and validate the token
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		switch token.Method.(type) {
		case *jwt.SigningMethodRSA, *jwt.SigningMethodECDSA:
			return rawKey, nil
		default:
			return nil, &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidToken,
				Message: "Unsupported signing method",
				Details: token.Header["alg"].(string),
			}
		}
	})

	if err != nil {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidToken,
			Message: "Token validation failed",
			Details: err.Error(),
		}
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidToken,
			Message: "Invalid token claims",
		}
	}

	// Validate standard claims
	if err := v.validateStandardClaims(claims); err != nil {
		return nil, err
	}

	// Convert to map[string]interface{}
	result := make(map[string]interface{})
	for k, v := range claims {
		result[k] = v
	}

	return result, nil
}

// ValidateWithKeySet validates a token using the provided key set
func (v *DefaultTokenValidator) ValidateWithKeySet(ctx context.Context, tokenString string, keySet jwk.Set) (map[string]interface{}, error) {
	// Create a temporary validator to avoid modifying shared state
	tempValidator := &DefaultTokenValidator{
		jwksURL:  v.jwksURL,
		audience: v.audience,
		issuer:   v.issuer,
		keySet:   keySet,
	}

	return tempValidator.ValidateToken(ctx, tokenString)
}

// validateStandardClaims validates standard JWT claims
func (v *DefaultTokenValidator) validateStandardClaims(claims jwt.MapClaims) error {
	now := time.Now()

	// Validate audience
	if v.audience != "" {
		if aud, ok := claims["aud"]; ok {
			audValid := false
			switch audVal := aud.(type) {
			case string:
				audValid = audVal == v.audience
			case []interface{}:
				for _, a := range audVal {
					if audStr, ok := a.(string); ok && audStr == v.audience {
						audValid = true
						break
					}
				}
			}
			if !audValid {
				return &auth.AuthenticationError{
					Code:    auth.ErrCodeInvalidToken,
					Message: "Invalid audience",
					Details: "Expected: " + v.audience,
				}
			}
		} else {
			return &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidToken,
				Message: "Missing audience claim",
				Details: "Expected: " + v.audience,
			}
		}
	}

	// Validate issuer
	if v.issuer != "" {
		if iss, ok := claims["iss"].(string); ok {
			if iss != v.issuer {
				return &auth.AuthenticationError{
					Code:    auth.ErrCodeInvalidToken,
					Message: "Invalid issuer",
					Details: "Expected: " + v.issuer,
				}
			}
		} else {
			return &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidToken,
				Message: "Missing issuer claim",
				Details: "Expected: " + v.issuer,
			}
		}
	}

	// Validate expiration
	if exp, ok := claims["exp"].(float64); ok {
		if now.Unix() >= int64(exp) {
			return &auth.AuthenticationError{
				Code:    auth.ErrCodeExpiredToken,
				Message: "Token has expired",
			}
		}
	}

	// Validate not before
	if nbf, ok := claims["nbf"].(float64); ok {
		if now.Unix() < int64(nbf) {
			return &auth.AuthenticationError{
				Code:    auth.ErrCodeInvalidToken,
				Message: "Token not yet valid",
			}
		}
	}

	return nil
}
