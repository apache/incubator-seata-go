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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/types"
)

// BasicAuthenticator implements HTTP Basic authentication with secure password handling
type BasicAuthenticator struct {
	name           string
	users          map[string]*BasicUserInfo // username -> user info mapping
	bcryptCost     int                       // bcrypt cost parameter
	maxFailures    int                       // maximum failed attempts before locking
	lockoutTime    time.Duration             // lockout duration after max failures
	failureTracker map[string]*FailureInfo   // track failed authentication attempts
}

// BasicUserInfo represents information about a basic auth user
type BasicUserInfo struct {
	UserID       string            `json:"userId"`
	Username     string            `json:"username"`
	PasswordHash string            `json:"passwordHash"`   // bcrypt hash
	Salt         string            `json:"salt,omitempty"` // Additional salt if needed
	Name         string            `json:"name"`
	Email        string            `json:"email"`
	Scopes       []string          `json:"scopes"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	LastLoginAt  *time.Time        `json:"lastLoginAt,omitempty"`
	IsActive     bool              `json:"isActive"`
}

// FailureInfo tracks authentication failures for rate limiting
type FailureInfo struct {
	Count       int        `json:"count"`
	LastFailAt  time.Time  `json:"lastFailAt"`
	LockedUntil *time.Time `json:"lockedUntil,omitempty"`
}

// BasicAuthenticatorConfig represents configuration for Basic authentication
type BasicAuthenticatorConfig struct {
	Name        string                    `json:"name"`
	Users       map[string]*BasicUserInfo `json:"users"`                 // username -> user info
	BcryptCost  int                       `json:"bcryptCost,omitempty"`  // Default: 12
	MaxFailures int                       `json:"maxFailures,omitempty"` // Default: 5
	LockoutTime time.Duration             `json:"lockoutTime,omitempty"` // Default: 15 minutes
}

// NewBasicAuthenticator creates a new Basic authenticator with security hardening
func NewBasicAuthenticator(config *BasicAuthenticatorConfig) *BasicAuthenticator {
	bcryptCost := config.BcryptCost
	if bcryptCost == 0 {
		bcryptCost = 12 // Secure default
	}

	maxFailures := config.MaxFailures
	if maxFailures == 0 {
		maxFailures = 5 // Default: 5 failures before lockout
	}

	lockoutTime := config.LockoutTime
	if lockoutTime == 0 {
		lockoutTime = 15 * time.Minute // Default: 15 minutes lockout
	}

	return &BasicAuthenticator{
		name:           config.Name,
		users:          config.Users,
		bcryptCost:     bcryptCost,
		maxFailures:    maxFailures,
		lockoutTime:    lockoutTime,
		failureTracker: make(map[string]*FailureInfo),
	}
}

// Name returns the name of the authenticator
func (b *BasicAuthenticator) Name() string {
	return b.name
}

// Authenticate attempts to authenticate a request using HTTP Basic authentication
func (b *BasicAuthenticator) Authenticate(ctx context.Context, md metadata.MD) (auth.User, error) {
	// Extract authorization header
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeMissingCredentials,
			Message: "Missing authorization header",
		}
	}

	authHeader := authHeaders[0]
	if !strings.HasPrefix(authHeader, "Basic ") {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Invalid authorization header format, expected Basic credentials",
		}
	}

	// Decode base64 credentials
	encodedCredentials := strings.TrimPrefix(authHeader, "Basic ")
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCredentials)
	if err != nil {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Invalid base64 encoding in Basic authentication",
			Details: err.Error(),
		}
	}

	credentials := string(decodedBytes)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Invalid Basic authentication format, expected username:password",
		}
	}

	username := parts[0]
	password := parts[1]

	// Validate input length to prevent DoS attacks
	if len(username) > 255 || len(password) > 512 {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Username or password too long",
		}
	}

	// Check if account is locked due to failed attempts
	if b.isAccountLocked(username) {
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Account temporarily locked due to too many failed attempts",
		}
	}

	// Validate credentials
	userInfo, exists := b.users[username]
	if !exists || !userInfo.IsActive {
		b.recordFailedAttempt(username)
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Invalid username or password",
		}
	}

	// Validate password using bcrypt
	if !b.validatePassword(password, userInfo.PasswordHash) {
		b.recordFailedAttempt(username)
		return &auth.UnauthenticatedUser{}, &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidCredentials,
			Message: "Invalid username or password",
		}
	}

	// Reset failure count on successful authentication
	b.resetFailureCount(username)

	// Update last login time
	now := time.Now()
	userInfo.LastLoginAt = &now
	userInfo.UpdatedAt = now

	// Convert metadata to map[string]interface{}
	claims := make(map[string]interface{})
	for k, v := range userInfo.Metadata {
		claims[k] = v
	}
	claims["auth_method"] = "basic"
	claims["username"] = username
	claims["last_login_at"] = now.Unix()
	claims["user_created_at"] = userInfo.CreatedAt.Unix()

	return auth.NewAuthenticatedUser(
		userInfo.UserID,
		userInfo.Name,
		userInfo.Email,
		userInfo.Scopes,
		claims,
	), nil
}

// SupportsScheme returns true if this authenticator supports HTTP Basic schemes
func (b *BasicAuthenticator) SupportsScheme(scheme types.SecurityScheme) bool {
	if scheme.SecuritySchemeType() == types.SecuritySchemeTypeHTTPAuth {
		if httpScheme, ok := scheme.(*types.HTTPAuthSecurityScheme); ok {
			return strings.ToLower(httpScheme.Scheme) == "basic"
		}
	}
	return false
}

// validatePassword validates a password against its bcrypt hash
func (b *BasicAuthenticator) validatePassword(password, passwordHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	return err == nil
}

// HashPassword creates a bcrypt hash of the given password
func (b *BasicAuthenticator) HashPassword(password string) (string, error) {
	// Validate password strength
	if err := b.validatePasswordStrength(password); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.bcryptCost)
	if err != nil {
		return "", &auth.AuthenticationError{
			Code:    ErrCodePasswordHashFailed,
			Message: "Failed to hash password",
			Details: err.Error(),
		}
	}

	return string(hash), nil
}

// validatePasswordStrength validates password meets security requirements
func (b *BasicAuthenticator) validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return &auth.AuthenticationError{
			Code:    ErrCodeWeakPassword,
			Message: "Password must be at least 8 characters long",
		}
	}

	if len(password) > 512 {
		return &auth.AuthenticationError{
			Code:    ErrCodeWeakPassword,
			Message: "Password too long (max 512 characters)",
		}
	}

	// Check for basic complexity requirements
	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return &auth.AuthenticationError{
			Code:    ErrCodeWeakPassword,
			Message: "Password must contain at least one uppercase letter, one lowercase letter, and one digit",
		}
	}

	return nil
}

// isAccountLocked checks if an account is currently locked
func (b *BasicAuthenticator) isAccountLocked(username string) bool {
	failInfo, exists := b.failureTracker[username]
	if !exists {
		return false
	}

	if failInfo.LockedUntil != nil && time.Now().Before(*failInfo.LockedUntil) {
		return true
	}

	// Clear expired lockout
	if failInfo.LockedUntil != nil && time.Now().After(*failInfo.LockedUntil) {
		failInfo.LockedUntil = nil
		failInfo.Count = 0
	}

	return false
}

// recordFailedAttempt records a failed authentication attempt
func (b *BasicAuthenticator) recordFailedAttempt(username string) {
	now := time.Now()

	failInfo, exists := b.failureTracker[username]
	if !exists {
		failInfo = &FailureInfo{}
		b.failureTracker[username] = failInfo
	}

	failInfo.Count++
	failInfo.LastFailAt = now

	// Lock account if max failures reached
	if failInfo.Count >= b.maxFailures {
		lockUntil := now.Add(b.lockoutTime)
		failInfo.LockedUntil = &lockUntil
	}
}

// resetFailureCount resets the failure count for a user
func (b *BasicAuthenticator) resetFailureCount(username string) {
	delete(b.failureTracker, username)
}

// AddUser adds a new user to the authenticator with secure password hashing
func (b *BasicAuthenticator) AddUser(username, password string, info *BasicUserInfo) error {
	if b.users == nil {
		b.users = make(map[string]*BasicUserInfo)
	}

	// Hash the password
	passwordHash, err := b.HashPassword(password)
	if err != nil {
		return err
	}

	// Generate user ID if not provided
	if info.UserID == "" {
		info.UserID = b.generateUserID()
	}

	// Set timestamps
	now := time.Now()
	info.Username = username
	info.PasswordHash = passwordHash
	info.CreatedAt = now
	info.UpdatedAt = now
	info.IsActive = true

	b.users[username] = info
	return nil
}

// UpdateUserPassword updates a user's password with proper hashing
func (b *BasicAuthenticator) UpdateUserPassword(username, newPassword string) error {
	userInfo, exists := b.users[username]
	if !exists {
		return &auth.AuthenticationError{
			Code:    ErrCodeUserNotFound,
			Message: "User not found",
		}
	}

	passwordHash, err := b.HashPassword(newPassword)
	if err != nil {
		return err
	}

	userInfo.PasswordHash = passwordHash
	userInfo.UpdatedAt = time.Now()

	// Reset failure count on password change
	b.resetFailureCount(username)

	return nil
}

// RemoveUser removes a user from the authenticator
func (b *BasicAuthenticator) RemoveUser(username string) {
	delete(b.users, username)
	delete(b.failureTracker, username)
}

// DeactivateUser deactivates a user account without removing it
func (b *BasicAuthenticator) DeactivateUser(username string) error {
	userInfo, exists := b.users[username]
	if !exists {
		return &auth.AuthenticationError{
			Code:    ErrCodeUserNotFound,
			Message: "User not found",
		}
	}

	userInfo.IsActive = false
	userInfo.UpdatedAt = time.Now()
	b.resetFailureCount(username)

	return nil
}

// ActivateUser activates a user account
func (b *BasicAuthenticator) ActivateUser(username string) error {
	userInfo, exists := b.users[username]
	if !exists {
		return &auth.AuthenticationError{
			Code:    ErrCodeUserNotFound,
			Message: "User not found",
		}
	}

	userInfo.IsActive = true
	userInfo.UpdatedAt = time.Now()
	b.resetFailureCount(username)

	return nil
}

// ListUsers returns all configured users (without sensitive information)
func (b *BasicAuthenticator) ListUsers() []*BasicUserInfo {
	var users []*BasicUserInfo
	for _, info := range b.users {
		// Create a copy without sensitive information
		userCopy := *info
		userCopy.PasswordHash = "" // Never expose password hash
		userCopy.Salt = ""         // Never expose salt
		users = append(users, &userCopy)
	}
	return users
}

// GetUser returns user information (without sensitive data)
func (b *BasicAuthenticator) GetUser(username string) (*BasicUserInfo, bool) {
	info, exists := b.users[username]
	if !exists {
		return nil, false
	}

	// Return copy without sensitive information
	userCopy := *info
	userCopy.PasswordHash = ""
	userCopy.Salt = ""
	return &userCopy, true
}

// GetFailureInfo returns failure tracking information for a user
func (b *BasicAuthenticator) GetFailureInfo(username string) (*FailureInfo, bool) {
	info, exists := b.failureTracker[username]
	return info, exists
}

// UnlockUser manually unlocks a user account
func (b *BasicAuthenticator) UnlockUser(username string) {
	b.resetFailureCount(username)
}

// generateUserID generates a secure random user ID
func (b *BasicAuthenticator) generateUserID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("user_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("user_%x", bytes)
}

// Additional error codes for password management
const (
	ErrCodePasswordHashFailed = "PASSWORD_HASH_FAILED"
	ErrCodeWeakPassword       = "WEAK_PASSWORD"
	ErrCodeUserNotFound       = "USER_NOT_FOUND"
)
