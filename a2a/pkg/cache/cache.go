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

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache defines the interface for distributed caching
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in the cache with expiration
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error

	// Delete removes a key from the cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// SetExpiration updates the expiration time for a key
	SetExpiration(ctx context.Context, key string, expiration time.Duration) error

	// GetWithTTL retrieves a value and its remaining TTL
	GetWithTTL(ctx context.Context, key string) ([]byte, time.Duration, error)

	// Close closes the cache connection
	Close() error

	// Ping tests the cache connection
	Ping(ctx context.Context) error
}

// RedisCache implements Cache interface using Redis
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client, prefix string) *RedisCache {
	if prefix == "" {
		prefix = "a2a"
	}
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

// buildKey creates a prefixed cache key
func (r *RedisCache) buildKey(key string) string {
	return r.prefix + ":" + key
}

// Get retrieves a value from Redis
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := r.buildKey(key)
	result, err := r.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	return []byte(result), nil
}

// Set stores a value in Redis with expiration
func (r *RedisCache) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	fullKey := r.buildKey(key)
	return r.client.Set(ctx, fullKey, value, expiration).Err()
}

// Delete removes a key from Redis
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := r.buildKey(key)
	return r.client.Del(ctx, fullKey).Err()
}

// Exists checks if a key exists in Redis
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.buildKey(key)
	count, err := r.client.Exists(ctx, fullKey).Result()
	return count > 0, err
}

// SetExpiration updates the expiration time for a key
func (r *RedisCache) SetExpiration(ctx context.Context, key string, expiration time.Duration) error {
	fullKey := r.buildKey(key)
	return r.client.Expire(ctx, fullKey, expiration).Err()
}

// GetWithTTL retrieves a value and its remaining TTL
func (r *RedisCache) GetWithTTL(ctx context.Context, key string) ([]byte, time.Duration, error) {
	fullKey := r.buildKey(key)

	// Use pipeline for atomic operation
	pipe := r.client.Pipeline()
	getCmd := pipe.Get(ctx, fullKey)
	ttlCmd := pipe.TTL(ctx, fullKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		if err == redis.Nil {
			return nil, 0, ErrCacheMiss
		}
		return nil, 0, err
	}

	value, err := getCmd.Result()
	if err != nil {
		if err == redis.Nil {
			return nil, 0, ErrCacheMiss
		}
		return nil, 0, err
	}

	ttl, err := ttlCmd.Result()
	if err != nil {
		return nil, 0, err
	}

	return []byte(value), ttl, nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// Ping tests the Redis connection
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// MemoryCache implements Cache interface using in-memory storage
type MemoryCache struct {
	data   map[string]*memoryCacheEntry
	mutex  sync.RWMutex
	stopCh chan struct{}
	closed bool
}

type memoryCacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache instance
func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		data:   make(map[string]*memoryCacheEntry),
		stopCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go mc.cleanupExpired()

	return mc
}

// Get retrieves a value from memory cache
func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.closed {
		return nil, ErrCacheClosed
	}

	entry, exists := m.data[key]
	if !exists {
		return nil, ErrCacheMiss
	}

	if time.Now().After(entry.expiresAt) {
		// Entry expired, remove it
		delete(m.data, key)
		return nil, ErrCacheMiss
	}

	return entry.value, nil
}

// Set stores a value in memory cache with expiration
func (m *MemoryCache) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return ErrCacheClosed
	}

	expiresAt := time.Now().Add(expiration)
	m.data[key] = &memoryCacheEntry{
		value:     value,
		expiresAt: expiresAt,
	}

	return nil
}

// Delete removes a key from memory cache
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return ErrCacheClosed
	}

	delete(m.data, key)
	return nil
}

// Exists checks if a key exists in memory cache
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.closed {
		return false, ErrCacheClosed
	}

	entry, exists := m.data[key]
	if !exists {
		return false, nil
	}

	if time.Now().After(entry.expiresAt) {
		return false, nil
	}

	return true, nil
}

// SetExpiration updates the expiration time for a key
func (m *MemoryCache) SetExpiration(ctx context.Context, key string, expiration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return ErrCacheClosed
	}

	entry, exists := m.data[key]
	if !exists {
		return ErrCacheMiss
	}

	entry.expiresAt = time.Now().Add(expiration)
	return nil
}

// GetWithTTL retrieves a value and its remaining TTL
func (m *MemoryCache) GetWithTTL(ctx context.Context, key string) ([]byte, time.Duration, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.closed {
		return nil, 0, ErrCacheClosed
	}

	entry, exists := m.data[key]
	if !exists {
		return nil, 0, ErrCacheMiss
	}

	now := time.Now()
	if now.After(entry.expiresAt) {
		return nil, 0, ErrCacheMiss
	}

	ttl := entry.expiresAt.Sub(now)
	return entry.value, ttl, nil
}

// Close closes the memory cache
func (m *MemoryCache) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.closed {
		close(m.stopCh)
		m.closed = true
		m.data = nil
	}

	return nil
}

// Ping always returns nil for memory cache
func (m *MemoryCache) Ping(ctx context.Context) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.closed {
		return ErrCacheClosed
	}

	return nil
}

// cleanupExpired removes expired entries from memory cache
func (m *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performCleanup()
		case <-m.stopCh:
			return
		}
	}
}

// performCleanup removes expired entries
func (m *MemoryCache) performCleanup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return
	}

	now := time.Now()
	for key, entry := range m.data {
		if now.After(entry.expiresAt) {
			delete(m.data, key)
		}
	}
}

// Cache errors
var (
	ErrCacheMiss   = errors.New("cache miss")
	ErrCacheClosed = errors.New("cache is closed")
)

// JSONCache provides JSON serialization helpers
type JSONCache struct {
	cache Cache
}

// NewJSONCache wraps a cache with JSON serialization
func NewJSONCache(cache Cache) *JSONCache {
	return &JSONCache{cache: cache}
}

// GetJSON retrieves and unmarshals a JSON value
func (j *JSONCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := j.cache.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// SetJSON marshals and stores a JSON value
func (j *JSONCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return j.cache.Set(ctx, key, data, expiration)
}

// GetCache returns the underlying cache
func (j *JSONCache) GetCache() Cache {
	return j.cache
}
