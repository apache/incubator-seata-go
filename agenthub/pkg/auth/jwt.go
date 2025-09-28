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
	"fmt"
	"time"

	"agenthub/pkg/common"
	"agenthub/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
)

// JWTManager manages JWT token operations following K8s patterns
type JWTManager struct {
	secretKey []byte
	issuer    string
	expiry    time.Duration
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey string
	Issuer    string
	Expiry    time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(config JWTConfig) *JWTManager {
	secretKey := config.SecretKey
	if secretKey == "" {
		// Generate random secret key if not provided
		var err error
		secretKey, err = utils.GenerateSecretKey()
		if err != nil {
			panic(fmt.Sprintf("failed to generate JWT secret key: %v", err))
		}
	}

	return &JWTManager{
		secretKey: []byte(secretKey),
		issuer:    config.Issuer,
		expiry:    config.Expiry,
	}
}

// GenerateToken generates a new JWT token
func (j *JWTManager) GenerateToken(userID, username string, roles []string) (string, error) {
	claims := &common.Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			Audience:  []string{"agent-hub"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.expiry)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateToken validates a JWT token and returns claims
func (j *JWTManager) ValidateToken(tokenString string) (*common.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &common.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*common.Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshToken refreshes a JWT token if it's within refresh window
func (j *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("invalid token for refresh: %w", err)
	}

	// Check if token is in refresh window (e.g., 30 minutes before expiry)
	refreshWindow := 30 * time.Minute
	if time.Until(claims.ExpiresAt.Time) > refreshWindow {
		return "", fmt.Errorf("token is not eligible for refresh yet")
	}

	// Generate new token
	return j.GenerateToken(claims.UserID, claims.Username, claims.Roles)
}

// GetSecretKey returns the secret key (for testing purposes)
func (j *JWTManager) GetSecretKey() string {
	return string(j.secretKey)
}

// SetExpiry sets the token expiry duration
func (j *JWTManager) SetExpiry(expiry time.Duration) {
	j.expiry = expiry
}
