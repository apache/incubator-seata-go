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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"

	"seata-go-ai-a2a/pkg/auth"
	"seata-go-ai-a2a/pkg/cache"
	"seata-go-ai-a2a/pkg/types"
)

// CachedJWKSEntry represents a cached JWKS entry for serialization
type CachedJWKSEntry struct {
	KeySetJSON []byte    `json:"key_set_json"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// CachedValidator implements JWS validation with distributed caching support
type CachedValidator struct {
	cache        cache.Cache
	cacheExpiry  time.Duration
	singleFlight *types.SingleFlight
}

// NewCachedValidator creates a new JWS validator with cache support
func NewCachedValidator(cacheInstance cache.Cache) *CachedValidator {
	return &CachedValidator{
		cache:        cacheInstance,
		cacheExpiry:  5 * time.Minute, // Cache JWKS for 5 minutes
		singleFlight: types.NewSingleFlight(),
	}
}

// NewCachedValidatorWithExpiry creates a new JWS validator with custom cache expiry
func NewCachedValidatorWithExpiry(cacheInstance cache.Cache, expiry time.Duration) *CachedValidator {
	return &CachedValidator{
		cache:        cacheInstance,
		cacheExpiry:  expiry,
		singleFlight: types.NewSingleFlight(),
	}
}

// ValidateSignature validates a JWS signature for the given payload
func (v *CachedValidator) ValidateSignature(ctx context.Context, payload []byte, signature *types.AgentCardSignature) error {
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

	// Get the key set from cache or fetch
	keySet, err := v.getKeySet(ctx, header.JWKSURL)
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

	// Verify signature
	return v.verifySignature(payload, signature, &header, key)
}

// getKeySet retrieves the key set from cache or fetches it from URL
func (v *CachedValidator) getKeySet(ctx context.Context, jwksURL string) (jwk.Set, error) {
	cacheKey := v.buildCacheKey(jwksURL)

	// Use single-flight to ensure only one fetch per URL
	result, err := v.singleFlight.Do(jwksURL, func() (interface{}, error) {
		// Try to get from cache first
		if keySet, err := v.getFromCache(ctx, cacheKey); err == nil {
			return keySet, nil
		}

		// Fetch from URL
		keySet, err := jwk.Fetch(ctx, jwksURL)
		if err != nil {
			return nil, &auth.AuthenticationError{
				Code:    auth.ErrCodeJWKSFetchFailed,
				Message: "Failed to fetch JWKS",
				Details: err.Error(),
			}
		}

		// Cache the key set
		if cacheErr := v.setInCache(ctx, cacheKey, keySet); cacheErr != nil {
			// Log cache error but don't fail the request
			// In production, you'd want to use proper logging here
		}

		return keySet, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(jwk.Set), nil
}

// buildCacheKey creates a cache key for JWKS URL
func (v *CachedValidator) buildCacheKey(jwksURL string) string {
	// Hash the URL to create a consistent key
	hash := sha256.Sum256([]byte(jwksURL))
	return fmt.Sprintf("jwks:%x", hash[:8]) // Use first 8 bytes of hash for shorter key
}

// getFromCache retrieves a key set from cache
func (v *CachedValidator) getFromCache(ctx context.Context, cacheKey string) (jwk.Set, error) {
	data, err := v.cache.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	var entry CachedJWKSEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	// Check if entry is expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, cache.ErrCacheMiss
	}

	// Parse the key set from JSON
	keySet, err := jwk.Parse(entry.KeySetJSON)
	if err != nil {
		return nil, err
	}

	return keySet, nil
}

// setInCache stores a key set in cache
func (v *CachedValidator) setInCache(ctx context.Context, cacheKey string, keySet jwk.Set) error {
	// Serialize the key set to JSON
	keySetJSON, err := json.Marshal(keySet)
	if err != nil {
		return err
	}

	entry := CachedJWKSEntry{
		KeySetJSON: keySetJSON,
		ExpiresAt:  time.Now().Add(v.cacheExpiry),
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return v.cache.Set(ctx, cacheKey, entryJSON, v.cacheExpiry)
}

// verifySignature performs the actual JWS signature verification
func (v *CachedValidator) verifySignature(payload []byte, signature *types.AgentCardSignature, header *JWSHeader, key jwk.Key) error {
	// Convert algorithm string to jwa.SignatureAlgorithm
	alg := jwa.SignatureAlgorithm(header.Algorithm)

	// Construct the JWS token for verification
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)
	jwsToken := fmt.Sprintf("%s.%s.%s", signature.Protected, payloadEncoded, signature.Signature)

	// Verify the signature
	_, err := jws.Verify([]byte(jwsToken), jws.WithKey(alg, key))
	if err != nil {
		return &auth.AuthenticationError{
			Code:    auth.ErrCodeInvalidSignature,
			Message: "JWS signature verification failed",
			Details: err.Error(),
		}
	}

	return nil
}

// Close cleans up resources
func (v *CachedValidator) Close() error {
	if v.cache != nil {
		return v.cache.Close()
	}
	return nil
}

// ClearCache removes all JWKS entries from cache
func (v *CachedValidator) ClearCache(ctx context.Context, jwksURL string) error {
	cacheKey := v.buildCacheKey(jwksURL)
	return v.cache.Delete(ctx, cacheKey)
}

// GetCacheStats returns cache statistics (if supported by underlying cache)
func (v *CachedValidator) GetCacheStats(ctx context.Context, jwksURL string) (bool, time.Duration, error) {
	cacheKey := v.buildCacheKey(jwksURL)

	exists, err := v.cache.Exists(ctx, cacheKey)
	if err != nil {
		return false, 0, err
	}

	if !exists {
		return false, 0, nil
	}

	_, ttl, err := v.cache.GetWithTTL(ctx, cacheKey)
	return exists, ttl, err
}
