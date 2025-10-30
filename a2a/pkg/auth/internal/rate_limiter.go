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

package internal

import (
	"context"
	"sync"
	"time"
)

// DefaultRateLimiter provides a comprehensive in-memory rate limiter implementation
type DefaultRateLimiter struct {
	config          *RateLimiterConfig
	clients         map[string]*ClientRateLimitState
	globalRequests  []RequestInfo
	mu              sync.RWMutex
	cleanupInterval time.Duration
	stopChan        chan struct{}
	statistics      *RateLimiterStatistics
}

// RateLimiterConfig configures the comprehensive rate limiter
type RateLimiterConfig struct {
	// Request rate limits
	MaxRequestsPerSecond int           `json:"max_requests_per_second"`
	MaxRequestsPerMinute int           `json:"max_requests_per_minute"`
	MaxRequestsPerHour   int           `json:"max_requests_per_hour"`
	MaxRequestsPerDay    int           `json:"max_requests_per_day"`
	BurstSize            int           `json:"burst_size"` // Token bucket burst size
	Window               time.Duration `json:"window"`     // Sliding window duration

	// Memory management
	MaxRequestHistory int `json:"max_request_history"` // Maximum requests to keep in memory per client
	MaxGlobalHistory  int `json:"max_global_history"`  // Maximum global requests to keep in memory

	// Failure tracking and banning
	FailureThreshold      int           `json:"failure_threshold"`       // Max failures before ban
	FailureWindow         time.Duration `json:"failure_window"`          // Window for counting failures
	BanDuration           time.Duration `json:"ban_duration"`            // Duration of ban
	ProgressiveBanEnabled bool          `json:"progressive_ban_enabled"` // Enable progressive ban durations
	MaxBanDuration        time.Duration `json:"max_ban_duration"`        // Maximum ban duration

	// Advanced features
	WhitelistedClients []string      `json:"whitelisted_clients"`  // Never rate limit these clients
	BlacklistedClients []string      `json:"blacklisted_clients"`  // Always block these clients
	GlobalRateLimit    int           `json:"global_rate_limit"`    // Global requests per second
	ClientQuotaEnabled bool          `json:"client_quota_enabled"` // Enable per-client quotas
	QuotaResetInterval time.Duration `json:"quota_reset_interval"` // How often to reset quotas

	// Adaptive rate limiting
	AdaptiveEnabled    bool    `json:"adaptive_enabled"`    // Enable adaptive rate limiting
	AdaptiveThreshold  float64 `json:"adaptive_threshold"`  // Threshold for adaptive adjustment
	AdaptiveAdjustment float64 `json:"adaptive_adjustment"` // How much to adjust rates
}

// ClientRateLimitState tracks the rate limit state for a specific client
type ClientRateLimitState struct {
	Requests        []RequestInfo `json:"requests"`
	Failures        []time.Time   `json:"failures"`
	BannedUntil     *time.Time    `json:"banned_until,omitempty"`
	BanCount        int           `json:"ban_count"`
	QuotaUsed       int           `json:"quota_used"`
	QuotaResetAt    time.Time     `json:"quota_reset_at"`
	LastRequestTime time.Time     `json:"last_request_time"`
	TokenBucket     *TokenBucket  `json:"token_bucket"`
	AdaptiveRate    float64       `json:"adaptive_rate"` // Current adaptive rate multiplier
}

// RequestInfo contains information about a request for rate limiting
type RequestInfo struct {
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Duration  int64     `json:"duration"` // nanoseconds
}

// TokenBucket implements token bucket algorithm for burst control
type TokenBucket struct {
	Tokens     float64   `json:"tokens"`
	LastRefill time.Time `json:"last_refill"`
	RefillRate float64   `json:"refill_rate"` // tokens per second
	BucketSize float64   `json:"bucket_size"`
}

// RateLimiterStatistics provides comprehensive statistics
type RateLimiterStatistics struct {
	TotalRequests        int64     `json:"total_requests"`
	AllowedRequests      int64     `json:"allowed_requests"`
	DeniedRequests       int64     `json:"denied_requests"`
	ActiveClients        int       `json:"active_clients"`
	BannedClients        int       `json:"banned_clients"`
	GlobalRequestsPerSec float64   `json:"global_requests_per_sec"`
	AverageResponseTime  float64   `json:"average_response_time"`
	PeakRequestsPerSec   float64   `json:"peak_requests_per_sec"`
	LastResetTime        time.Time `json:"last_reset_time"`
}

// DefaultRateLimiterConfig returns a default configuration
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		MaxRequestsPerSecond:  100,
		MaxRequestsPerMinute:  6000,
		MaxRequestsPerHour:    360000,
		MaxRequestsPerDay:     8640000,
		BurstSize:             10,
		Window:                time.Minute,
		MaxRequestHistory:     1000,
		MaxGlobalHistory:      10000,
		FailureThreshold:      5,
		FailureWindow:         time.Minute * 5,
		BanDuration:           time.Minute * 15,
		ProgressiveBanEnabled: true,
		MaxBanDuration:        time.Hour * 24,
		GlobalRateLimit:       1000,
		ClientQuotaEnabled:    false,
		QuotaResetInterval:    time.Hour,
		AdaptiveEnabled:       false,
		AdaptiveThreshold:     0.8,
		AdaptiveAdjustment:    0.1,
	}
}

// NewDefaultRateLimiter creates a new comprehensive rate limiter
func NewDefaultRateLimiter(config *RateLimiterConfig) *DefaultRateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	rl := &DefaultRateLimiter{
		config:          config,
		clients:         make(map[string]*ClientRateLimitState),
		globalRequests:  make([]RequestInfo, 0),
		cleanupInterval: time.Minute * 5,
		stopChan:        make(chan struct{}),
		statistics:      &RateLimiterStatistics{LastResetTime: time.Now()},
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow returns true if the request should be allowed
func (rl *DefaultRateLimiter) Allow(ctx context.Context, clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Check blacklist
	for _, blacklisted := range rl.config.BlacklistedClients {
		if clientID == blacklisted {
			rl.statistics.DeniedRequests++
			return false
		}
	}

	// Check whitelist
	for _, whitelisted := range rl.config.WhitelistedClients {
		if clientID == whitelisted {
			rl.recordRequest(clientID, now, true, 0)
			return true
		}
	}

	// Get or create client state
	state, exists := rl.clients[clientID]
	if !exists {
		state = &ClientRateLimitState{
			Requests:     make([]RequestInfo, 0),
			Failures:     make([]time.Time, 0),
			TokenBucket:  rl.createTokenBucket(),
			QuotaResetAt: now.Add(rl.config.QuotaResetInterval),
			AdaptiveRate: 1.0,
		}
		rl.clients[clientID] = state
	}

	// Check if banned
	if state.BannedUntil != nil && now.Before(*state.BannedUntil) {
		rl.statistics.DeniedRequests++
		return false
	}

	// Clear expired ban
	if state.BannedUntil != nil && now.After(*state.BannedUntil) {
		state.BannedUntil = nil
	}

	// Check global rate limit
	if !rl.checkGlobalRateLimit(now) {
		rl.statistics.DeniedRequests++
		return false
	}

	// Check per-client limits
	if !rl.checkClientRateLimit(state, now) {
		rl.statistics.DeniedRequests++
		return false
	}

	// Check token bucket
	if !rl.checkTokenBucket(state.TokenBucket, now) {
		rl.statistics.DeniedRequests++
		return false
	}

	// Check quota if enabled
	if rl.config.ClientQuotaEnabled && !rl.checkQuota(state, now) {
		rl.statistics.DeniedRequests++
		return false
	}

	// Request is allowed
	rl.recordRequest(clientID, now, true, 0)
	rl.statistics.AllowedRequests++
	return true
}

// Reset resets the rate limiter for a specific client
func (rl *DefaultRateLimiter) Reset(ctx context.Context, clientID string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.clients, clientID)
	return nil
}

// Close cleans up resources
func (rl *DefaultRateLimiter) Close() error {
	close(rl.stopChan)
	return nil
}

// Helper methods...

func (rl *DefaultRateLimiter) createTokenBucket() *TokenBucket {
	return &TokenBucket{
		Tokens:     float64(rl.config.BurstSize),
		LastRefill: time.Now(),
		RefillRate: float64(rl.config.MaxRequestsPerSecond),
		BucketSize: float64(rl.config.BurstSize),
	}
}

func (rl *DefaultRateLimiter) checkGlobalRateLimit(now time.Time) bool {
	// Clean old global requests
	rl.cleanOldRequests(&rl.globalRequests, now, time.Second)

	// Apply memory limits
	if len(rl.globalRequests) > rl.config.MaxGlobalHistory {
		// Keep only the most recent requests
		excess := len(rl.globalRequests) - rl.config.MaxGlobalHistory
		rl.globalRequests = rl.globalRequests[excess:]
	}

	return len(rl.globalRequests) < rl.config.GlobalRateLimit
}

func (rl *DefaultRateLimiter) checkClientRateLimit(state *ClientRateLimitState, now time.Time) bool {
	// Clean old requests
	rl.cleanOldRequests(&state.Requests, now, rl.config.Window)

	// Apply memory limits
	if len(state.Requests) > rl.config.MaxRequestHistory {
		// Keep only the most recent requests
		excess := len(state.Requests) - rl.config.MaxRequestHistory
		state.Requests = state.Requests[excess:]
	}

	// Count requests in different windows
	requestsInSecond := rl.countRequestsInWindow(state.Requests, now, time.Second)
	requestsInMinute := rl.countRequestsInWindow(state.Requests, now, time.Minute)
	requestsInHour := rl.countRequestsInWindow(state.Requests, now, time.Hour)
	requestsInDay := rl.countRequestsInWindow(state.Requests, now, time.Hour*24)

	// Apply adaptive rate limiting
	adaptiveLimit := func(base int) int {
		if rl.config.AdaptiveEnabled {
			return int(float64(base) * state.AdaptiveRate)
		}
		return base
	}

	return requestsInSecond < adaptiveLimit(rl.config.MaxRequestsPerSecond) &&
		requestsInMinute < adaptiveLimit(rl.config.MaxRequestsPerMinute) &&
		requestsInHour < adaptiveLimit(rl.config.MaxRequestsPerHour) &&
		requestsInDay < adaptiveLimit(rl.config.MaxRequestsPerDay)
}

func (rl *DefaultRateLimiter) checkTokenBucket(bucket *TokenBucket, now time.Time) bool {
	// Refill tokens
	elapsed := now.Sub(bucket.LastRefill).Seconds()
	tokensToAdd := elapsed * bucket.RefillRate
	bucket.Tokens = min(bucket.BucketSize, bucket.Tokens+tokensToAdd)
	bucket.LastRefill = now

	// Check if token available
	if bucket.Tokens >= 1.0 {
		bucket.Tokens -= 1.0
		return true
	}

	return false
}

func (rl *DefaultRateLimiter) checkQuota(state *ClientRateLimitState, now time.Time) bool {
	// Reset quota if needed
	if now.After(state.QuotaResetAt) {
		state.QuotaUsed = 0
		state.QuotaResetAt = now.Add(rl.config.QuotaResetInterval)
	}

	// Check quota (using daily limit as quota)
	if state.QuotaUsed >= rl.config.MaxRequestsPerDay {
		return false
	}

	return true
}

func (rl *DefaultRateLimiter) recordRequest(clientID string, timestamp time.Time, success bool, duration time.Duration) {
	// Record global request
	rl.globalRequests = append(rl.globalRequests, RequestInfo{
		Timestamp: timestamp,
		Success:   success,
		Duration:  duration.Nanoseconds(),
	})

	// Record client request
	if state, exists := rl.clients[clientID]; exists {
		state.Requests = append(state.Requests, RequestInfo{
			Timestamp: timestamp,
			Success:   success,
			Duration:  duration.Nanoseconds(),
		})
		state.LastRequestTime = timestamp

		if rl.config.ClientQuotaEnabled {
			state.QuotaUsed++
		}

		// Record failure if applicable
		if !success {
			state.Failures = append(state.Failures, timestamp)
			rl.checkForBan(state, timestamp)
		}
	}

	// Update statistics
	rl.statistics.TotalRequests++
}

func (rl *DefaultRateLimiter) checkForBan(state *ClientRateLimitState, now time.Time) {
	// Clean old failures
	cutoff := now.Add(-rl.config.FailureWindow)
	newFailures := make([]time.Time, 0, len(state.Failures))
	for _, failure := range state.Failures {
		if failure.After(cutoff) {
			newFailures = append(newFailures, failure)
		}
	}
	state.Failures = newFailures

	// Check if ban threshold reached
	if len(state.Failures) >= rl.config.FailureThreshold {
		banDuration := rl.config.BanDuration

		// Progressive ban duration
		if rl.config.ProgressiveBanEnabled && state.BanCount > 0 {
			multiplier := 1 << state.BanCount // Exponential backoff
			banDuration = time.Duration(int64(banDuration) * int64(multiplier))
			if banDuration > rl.config.MaxBanDuration {
				banDuration = rl.config.MaxBanDuration
			}
		}

		banUntil := now.Add(banDuration)
		state.BannedUntil = &banUntil
		state.BanCount++
		state.Failures = nil // Clear failures after ban
	}
}

func (rl *DefaultRateLimiter) cleanOldRequests(requests *[]RequestInfo, now time.Time, window time.Duration) {
	cutoff := now.Add(-window)
	newRequests := make([]RequestInfo, 0, len(*requests))
	for _, req := range *requests {
		if req.Timestamp.After(cutoff) {
			newRequests = append(newRequests, req)
		}
	}
	*requests = newRequests
}

func (rl *DefaultRateLimiter) countRequestsInWindow(requests []RequestInfo, now time.Time, window time.Duration) int {
	cutoff := now.Add(-window)
	count := 0
	for _, req := range requests {
		if req.Timestamp.After(cutoff) {
			count++
		}
	}
	return count
}

func (rl *DefaultRateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.performCleanup()
		case <-rl.stopChan:
			return
		}
	}
}

func (rl *DefaultRateLimiter) performCleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean up expired client states
	for clientID, state := range rl.clients {
		// Remove clients that haven't made requests in a long time
		if now.Sub(state.LastRequestTime) > time.Hour*24 {
			delete(rl.clients, clientID)
			continue
		}

		// Clean old requests and failures
		rl.cleanOldRequests(&state.Requests, now, rl.config.Window)
		cutoff := now.Add(-rl.config.FailureWindow)
		newFailures := make([]time.Time, 0, len(state.Failures))
		for _, failure := range state.Failures {
			if failure.After(cutoff) {
				newFailures = append(newFailures, failure)
			}
		}
		state.Failures = newFailures
	}

	// Clean global requests
	rl.cleanOldRequests(&rl.globalRequests, now, time.Hour)

	// Update statistics
	rl.updateStatistics(now)
}

func (rl *DefaultRateLimiter) updateStatistics(now time.Time) {
	rl.statistics.ActiveClients = len(rl.clients)

	bannedCount := 0
	for _, state := range rl.clients {
		if state.BannedUntil != nil && now.Before(*state.BannedUntil) {
			bannedCount++
		}
	}
	rl.statistics.BannedClients = bannedCount

	// Calculate requests per second
	recentRequests := rl.countRequestsInWindow(rl.globalRequests, now, time.Second)
	rl.statistics.GlobalRequestsPerSec = float64(recentRequests)

	if rl.statistics.GlobalRequestsPerSec > rl.statistics.PeakRequestsPerSec {
		rl.statistics.PeakRequestsPerSec = rl.statistics.GlobalRequestsPerSec
	}
}

// GetStatistics returns current rate limiter statistics
func (rl *DefaultRateLimiter) GetStatistics() *RateLimiterStatistics {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := *rl.statistics
	return &stats
}

// RecordFailure records a failed request for the client
func (rl *DefaultRateLimiter) RecordFailure(ctx context.Context, clientID string, duration time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.recordRequest(clientID, time.Now(), false, duration)
}

// min helper function
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
