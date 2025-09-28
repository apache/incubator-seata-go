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

package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// CryptoUtils provides cryptographic utilities
type CryptoUtils struct{}

// NewCryptoUtils creates a new crypto utilities instance
func NewCryptoUtils() *CryptoUtils {
	return &CryptoUtils{}
}

// GenerateRandomKey generates a random key of specified length in bytes
func (c *CryptoUtils) GenerateRandomKey(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("key length must be positive")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateSecretKey generates a 32-byte (256-bit) secret key suitable for JWT signing
func (c *CryptoUtils) GenerateSecretKey() (string, error) {
	return c.GenerateRandomKey(32)
}

// GenerateAPIKey generates a 16-byte (128-bit) API key
func (c *CryptoUtils) GenerateAPIKey() (string, error) {
	return c.GenerateRandomKey(16)
}

// GenerateSessionID generates an 8-byte (64-bit) session ID
func (c *CryptoUtils) GenerateSessionID() (string, error) {
	return c.GenerateRandomKey(8)
}

// ValidateKeyLength validates that a key meets minimum security requirements
func (c *CryptoUtils) ValidateKeyLength(key string, minBytes int) error {
	decoded, err := hex.DecodeString(key)
	if err != nil {
		return fmt.Errorf("invalid hex encoded key: %w", err)
	}

	if len(decoded) < minBytes {
		return fmt.Errorf("key too short: got %d bytes, need at least %d bytes", len(decoded), minBytes)
	}

	return nil
}

// Global instance for convenience
var globalCrypto = NewCryptoUtils()

// Package-level convenience functions
func GenerateRandomKey(length int) (string, error) {
	return globalCrypto.GenerateRandomKey(length)
}

func GenerateSecretKey() (string, error) {
	return globalCrypto.GenerateSecretKey()
}

func GenerateAPIKey() (string, error) {
	return globalCrypto.GenerateAPIKey()
}

func GenerateSessionID() (string, error) {
	return globalCrypto.GenerateSessionID()
}

func ValidateKeyLength(key string, minBytes int) error {
	return globalCrypto.ValidateKeyLength(key, minBytes)
}

// Legacy function for backward compatibility
func GenerateRandomKeyLegacy() string {
	key, err := GenerateSecretKey()
	if err != nil {
		panic(fmt.Sprintf("failed to generate random key: %v", err))
	}
	return key
}
