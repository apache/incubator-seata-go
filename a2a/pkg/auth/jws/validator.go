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

package jws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/types"
)

// JWSHeader represents JWS protected header
type JWSHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	JWKSURL   string `json:"jku"`
	Type      string `json:"typ,omitempty"`
}

// JWKSCacheEntry represents a cached JWKS entry with expiration
type JWKSCacheEntry struct {
	KeySet    jwk.Set
	ExpiresAt time.Time
}

// Validator implements JWS validation for Agent Cards with in-memory caching
type Validator struct {
	keySetCache  map[string]*JWKSCacheEntry
	cacheMutex   sync.RWMutex
	cacheExpiry  time.Duration
	stopChan     chan struct{}
	cleanupOnce  sync.Once
	singleFlight *types.SingleFlight
}

// NewValidator creates a new JWS validator with in-memory caching
func NewValidator() *Validator {
	validator := &Validator{
		keySetCache:  make(map[string]*JWKSCacheEntry),
		cacheExpiry:  5 * time.Minute, // Cache JWKS for 5 minutes
		stopChan:     make(chan struct{}),
		singleFlight: types.NewSingleFlight(),
	}

	// Start background cleanup routine
	go validator.cleanupExpiredEntries()

	return validator
}

// NewValidatorWithExpiry creates a new JWS validator with custom cache expiry
func NewValidatorWithExpiry(expiry time.Duration) *Validator {
	validator := &Validator{
		keySetCache:  make(map[string]*JWKSCacheEntry),
		cacheExpiry:  expiry,
		stopChan:     make(chan struct{}),
		singleFlight: types.NewSingleFlight(),
	}

	go validator.cleanupExpiredEntries()
	return validator
}

// ValidateSignature validates a JWS signature for the given payload
func (v *Validator) ValidateSignature(ctx context.Context, payload []byte, signature *types.AgentCardSignature) error {
	// Decode the protected header
	protectedBytes, err := base64.RawURLEncoding.DecodeString(signature.Protected)
	if err != nil {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to decode protected header",
			Details: err.Error(),
		}
	}

	var header JWSHeader
	if err := json.Unmarshal(protectedBytes, &header); err != nil {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Failed to parse protected header",
			Details: err.Error(),
		}
	}

	// Validate required fields
	if header.Algorithm == "" {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Missing algorithm in JWS header",
		}
	}

	if header.KeyID == "" {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Missing key ID in JWS header",
		}
	}

	if header.JWKSURL == "" {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Missing JWKS URL in JWS header",
		}
	}

	// Get the key set
	keySet, err := v.GetKeySet(ctx, header.JWKSURL)
	if err != nil {
		return err
	}

	// Find the key
	key, ok := keySet.LookupKeyID(header.KeyID)
	if !ok {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "Key not found in JWKS",
			Details: fmt.Sprintf("Key ID: %s", header.KeyID),
		}
	}

	// Construct the JWS token for verification
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)
	jwsToken := fmt.Sprintf("%s.%s.%s", signature.Protected, payloadEncoded, signature.Signature)

	// Convert algorithm string to jwa.SignatureAlgorithm
	alg := jwa.SignatureAlgorithm(header.Algorithm)

	// Verify the signature
	_, err = jws.Verify([]byte(jwsToken), jws.WithKey(alg, key))
	if err != nil {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "JWS signature verification failed",
			Details: err.Error(),
		}
	}

	return nil
}

// GetKeySet retrieves the key set from the given JWKS URL with caching and single-flight pattern
func (v *Validator) GetKeySet(ctx context.Context, jwksURL string) (jwk.Set, error) {
	now := time.Now()

	// Check cache first
	v.cacheMutex.RLock()
	if entry, exists := v.keySetCache[jwksURL]; exists && now.Before(entry.ExpiresAt) {
		keySet := entry.KeySet
		v.cacheMutex.RUnlock()
		return keySet, nil
	}
	v.cacheMutex.RUnlock()

	// Use single-flight to ensure only one fetch per URL
	result, err := v.singleFlight.Do(jwksURL, func() (interface{}, error) {
		// Double-check cache inside single-flight
		v.cacheMutex.RLock()
		if entry, exists := v.keySetCache[jwksURL]; exists && time.Now().Before(entry.ExpiresAt) {
			keySet := entry.KeySet
			v.cacheMutex.RUnlock()
			return keySet, nil
		}
		v.cacheMutex.RUnlock()

		// Fetch from URL
		keySet, err := jwk.Fetch(ctx, jwksURL)
		if err != nil {
			return nil, &auth.AuthenticationError{
				Code:    auth.ErrCodeJWKSFetchFailed,
				Message: "Failed to fetch JWKS",
				Details: err.Error(),
			}
		}

		// Cache the key set with expiration
		v.cacheMutex.Lock()
		v.keySetCache[jwksURL] = &JWKSCacheEntry{
			KeySet:    keySet,
			ExpiresAt: time.Now().Add(v.cacheExpiry),
		}
		v.cacheMutex.Unlock()

		return keySet, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(jwk.Set), nil
}

// Close gracefully shuts down the validator and stops background routines
func (v *Validator) Close() {
	v.cleanupOnce.Do(func() {
		close(v.stopChan)

		// Clear cache
		v.cacheMutex.Lock()
		v.keySetCache = make(map[string]*JWKSCacheEntry)
		v.cacheMutex.Unlock()
	})
}

// cleanupExpiredEntries runs a background cleanup routine for expired cache entries
func (v *Validator) cleanupExpiredEntries() {
	ticker := time.NewTicker(v.cacheExpiry / 2) // Clean up twice as often as expiry
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			v.performCleanup()
		case <-v.stopChan:
			return
		}
	}
}

// performCleanup removes expired entries from the cache
func (v *Validator) performCleanup() {
	now := time.Now()

	v.cacheMutex.Lock()
	defer v.cacheMutex.Unlock()

	for url, entry := range v.keySetCache {
		if now.After(entry.ExpiresAt) {
			delete(v.keySetCache, url)
		}
	}
}
