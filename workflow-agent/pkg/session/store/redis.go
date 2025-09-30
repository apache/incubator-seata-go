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

package store

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"seata-go-ai-workflow-agent/pkg/session/config"
	"seata-go-ai-workflow-agent/pkg/session/types"
	"strings"
	"time"
)

// RedisStore implements SessionStore using Redis as backend
type RedisStore struct {
	client    redis.UniversalClient
	config    config.RedisConfig
	keyPrefix string
	startTime time.Time
}

// NewRedisStore creates a new Redis-based session store
func NewRedisStore(cfg config.RedisConfig) (*RedisStore, error) {
	var client redis.UniversalClient

	opts := &redis.UniversalOptions{
		Addrs:        []string{cfg.Addr},
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// Configure TLS if enabled
	if cfg.TLS.Enabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}

		if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		opts.TLSConfig = tlsConfig
	}

	client = redis.NewUniversalClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	keyPrefix := cfg.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "session:"
	}

	return &RedisStore{
		client:    client,
		config:    cfg,
		keyPrefix: keyPrefix,
		startTime: time.Now(),
	}, nil
}

// Get retrieves a session by ID
func (r *RedisStore) Get(ctx context.Context, sessionID string) (*types.SessionData, error) {
	key := r.sessionKey(sessionID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, types.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	session := &types.SessionData{}
	if err := json.Unmarshal([]byte(data), session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	if session.IsExpired() {
		// Clean up expired session
		r.client.Del(ctx, key)
		r.removeFromUserIndex(ctx, session.UserID, sessionID)
		return nil, types.ErrSessionExpired
	}

	return session, nil
}

// Set stores or updates a session
func (r *RedisStore) Set(ctx context.Context, session *types.SessionData) error {
	if session == nil {
		return types.ErrInvalidSession
	}

	session.UpdatedAt = time.Now()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := r.sessionKey(session.ID)

	// Calculate TTL
	var ttl time.Duration
	if session.ExpiresAt != nil {
		ttl = time.Until(*session.ExpiresAt)
		if ttl <= 0 {
			return types.ErrSessionExpired
		}
	}

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	if ttl > 0 {
		pipe.Set(ctx, key, data, ttl)
	} else {
		pipe.Set(ctx, key, data, 0)
	}

	// Update user index
	userIndexKey := r.userIndexKey(session.UserID)
	pipe.SAdd(ctx, userIndexKey, session.ID)

	if ttl > 0 {
		pipe.Expire(ctx, userIndexKey, ttl+time.Hour) // Keep index slightly longer
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set session: %w", err)
	}

	return nil
}

// Delete removes a session
func (r *RedisStore) Delete(ctx context.Context, sessionID string) error {
	key := r.sessionKey(sessionID)

	// Get session first to find user ID
	session, err := r.Get(ctx, sessionID)
	if err != nil {
		if err == types.ErrSessionNotFound {
			return nil // Already deleted
		}
		return err
	}

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)

	// Remove from user index
	userIndexKey := r.userIndexKey(session.UserID)
	pipe.SRem(ctx, userIndexKey, sessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// Exists checks if a session exists
func (r *RedisStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	key := r.sessionKey(sessionID)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	if exists == 0 {
		return false, nil
	}

	// Check if expired
	session, err := r.Get(ctx, sessionID)
	if err != nil {
		if err == types.ErrSessionExpired || err == types.ErrSessionNotFound {
			return false, nil
		}
		return false, err
	}

	return session != nil, nil
}

// List returns session IDs for a user with pagination
func (r *RedisStore) List(ctx context.Context, userID string, offset, limit int) ([]string, error) {
	userIndexKey := r.userIndexKey(userID)

	// Get all session IDs for user
	sessionIDs, err := r.client.SMembers(ctx, userIndexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Filter out expired sessions and get timestamps
	var validSessions []sessionWithTime
	for _, sessionID := range sessionIDs {
		session, err := r.Get(ctx, sessionID)
		if err != nil {
			if err == types.ErrSessionExpired || err == types.ErrSessionNotFound {
				// Remove expired session from index
				r.client.SRem(ctx, userIndexKey, sessionID)
				continue
			}
			return nil, err
		}

		validSessions = append(validSessions, sessionWithTime{
			ID:        sessionID,
			UpdatedAt: session.UpdatedAt,
		})
	}

	// Sort by update time (newest first)
	for i := 0; i < len(validSessions)-1; i++ {
		for j := i + 1; j < len(validSessions); j++ {
			if validSessions[i].UpdatedAt.Before(validSessions[j].UpdatedAt) {
				validSessions[i], validSessions[j] = validSessions[j], validSessions[i]
			}
		}
	}

	// Apply pagination
	end := offset + limit
	if offset > len(validSessions) {
		return []string{}, nil
	}
	if end > len(validSessions) {
		end = len(validSessions)
	}

	result := make([]string, end-offset)
	for i := offset; i < end; i++ {
		result[i-offset] = validSessions[i].ID
	}

	return result, nil
}

// Count returns the total number of sessions for a user
func (r *RedisStore) Count(ctx context.Context, userID string) (int64, error) {
	userIndexKey := r.userIndexKey(userID)

	sessionIDs, err := r.client.SMembers(ctx, userIndexKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get user sessions: %w", err)
	}

	count := int64(0)
	for _, sessionID := range sessionIDs {
		exists, err := r.Exists(ctx, sessionID)
		if err != nil {
			continue
		}
		if exists {
			count++
		}
	}

	return count, nil
}

// Cleanup removes expired sessions
func (r *RedisStore) Cleanup(ctx context.Context) (int64, error) {
	pattern := r.keyPrefix + "*"

	var cursor uint64
	var removed int64

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return removed, fmt.Errorf("failed to scan keys: %w", err)
		}

		for _, key := range keys {
			// Skip user index keys
			if strings.Contains(key, ":user:") {
				continue
			}

			// Check if session exists (Redis handles TTL automatically)
			exists, err := r.client.Exists(ctx, key).Result()
			if err != nil {
				continue
			}

			if exists == 0 {
				removed++
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return removed, nil
}

// Close closes the store and releases resources
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// Health checks store connectivity
func (r *RedisStore) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetStats returns store statistics
func (r *RedisStore) GetStats(ctx context.Context) (types.StoreStats, error) {
	stats := types.StoreStats{}

	// Count total sessions
	pattern := r.keyPrefix + "*"
	var cursor uint64
	var totalSessions, activeSessions int64
	var totalMessages int64

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return stats, fmt.Errorf("failed to scan keys: %w", err)
		}

		for _, key := range keys {
			// Skip user index keys
			if strings.Contains(key, ":user:") {
				continue
			}

			totalSessions++

			// Get session data to check if active
			data, err := r.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			session := &types.SessionData{}
			if err := json.Unmarshal([]byte(data), session); err != nil {
				continue
			}

			if !session.IsExpired() {
				activeSessions++
				totalMessages += int64(session.MessageCount)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	stats.TotalSessions = totalSessions
	stats.ActiveSessions = activeSessions
	stats.ExpiredSessions = totalSessions - activeSessions

	if activeSessions > 0 {
		stats.AverageMessages = float64(totalMessages) / float64(activeSessions)
	}

	return stats, nil
}

// sessionKey generates the Redis key for a session
func (r *RedisStore) sessionKey(sessionID string) string {
	return r.keyPrefix + sessionID
}

// userIndexKey generates the Redis key for user session index
func (r *RedisStore) userIndexKey(userID string) string {
	return r.keyPrefix + "user:" + userID
}

// removeFromUserIndex removes a session from user index
func (r *RedisStore) removeFromUserIndex(ctx context.Context, userID, sessionID string) {
	userIndexKey := r.userIndexKey(userID)
	r.client.SRem(ctx, userIndexKey, sessionID)
}

// sessionWithTime helper struct for sorting
type sessionWithTime struct {
	ID        string
	UpdatedAt time.Time
}
