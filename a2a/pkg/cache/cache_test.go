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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	t.Run("Basic Set and Get", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")
		expiration := 1 * time.Hour

		err := cache.Set(ctx, key, value, expiration)
		require.NoError(t, err)

		retrieved, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})

	t.Run("Cache Miss", func(t *testing.T) {
		_, err := cache.Get(ctx, "nonexistent-key")
		assert.Equal(t, ErrCacheMiss, err)
	})

	t.Run("Expiration", func(t *testing.T) {
		key := "expire-key"
		value := []byte("expire-value")
		expiration := 10 * time.Millisecond

		err := cache.Set(ctx, key, value, expiration)
		require.NoError(t, err)

		// Should still be available immediately
		retrieved, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		// Wait for expiration
		time.Sleep(20 * time.Millisecond)

		_, err = cache.Get(ctx, key)
		assert.Equal(t, ErrCacheMiss, err)
	})

	t.Run("Exists", func(t *testing.T) {
		key := "exists-key"
		value := []byte("exists-value")
		expiration := 1 * time.Hour

		exists, err := cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)

		err = cache.Set(ctx, key, value, expiration)
		require.NoError(t, err)

		exists, err = cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Delete", func(t *testing.T) {
		key := "delete-key"
		value := []byte("delete-value")
		expiration := 1 * time.Hour

		err := cache.Set(ctx, key, value, expiration)
		require.NoError(t, err)

		err = cache.Delete(ctx, key)
		require.NoError(t, err)

		_, err = cache.Get(ctx, key)
		assert.Equal(t, ErrCacheMiss, err)
	})

	t.Run("GetWithTTL", func(t *testing.T) {
		key := "ttl-key"
		value := []byte("ttl-value")
		expiration := 1 * time.Hour

		err := cache.Set(ctx, key, value, expiration)
		require.NoError(t, err)

		retrieved, ttl, err := cache.GetWithTTL(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
		assert.Greater(t, ttl, 50*time.Minute) // Should be close to 1 hour
		assert.LessOrEqual(t, ttl, expiration)
	})

	t.Run("SetExpiration", func(t *testing.T) {
		key := "set-exp-key"
		value := []byte("set-exp-value")
		initialExpiration := 1 * time.Hour
		newExpiration := 20 * time.Millisecond

		err := cache.Set(ctx, key, value, initialExpiration)
		require.NoError(t, err)

		err = cache.SetExpiration(ctx, key, newExpiration)
		require.NoError(t, err)

		// Wait for new expiration
		time.Sleep(30 * time.Millisecond)

		_, err = cache.Get(ctx, key)
		assert.Equal(t, ErrCacheMiss, err)
	})

	t.Run("Closed Cache", func(t *testing.T) {
		closedCache := NewMemoryCache()
		err := closedCache.Close()
		require.NoError(t, err)

		err = closedCache.Set(ctx, "key", []byte("value"), time.Hour)
		assert.Equal(t, ErrCacheClosed, err)

		_, err = closedCache.Get(ctx, "key")
		assert.Equal(t, ErrCacheClosed, err)
	})

	t.Run("Ping", func(t *testing.T) {
		err := cache.Ping(ctx)
		assert.NoError(t, err)

		closedCache := NewMemoryCache()
		err = closedCache.Close()
		require.NoError(t, err)

		err = closedCache.Ping(ctx)
		assert.Equal(t, ErrCacheClosed, err)
	})
}

func TestJSONCache(t *testing.T) {
	memCache := NewMemoryCache()
	defer memCache.Close()

	jsonCache := NewJSONCache(memCache)
	ctx := context.Background()

	t.Run("JSON Serialization", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		original := TestStruct{Name: "test", Value: 42}
		key := "json-key"
		expiration := 1 * time.Hour

		err := jsonCache.SetJSON(ctx, key, original, expiration)
		require.NoError(t, err)

		var retrieved TestStruct
		err = jsonCache.GetJSON(ctx, key, &retrieved)
		require.NoError(t, err)
		assert.Equal(t, original, retrieved)
	})

	t.Run("JSON Invalid Data", func(t *testing.T) {
		key := "invalid-json-key"
		expiration := 1 * time.Hour

		// Set invalid JSON directly through underlying cache
		err := memCache.Set(ctx, key, []byte("invalid json"), expiration)
		require.NoError(t, err)

		var result map[string]interface{}
		err = jsonCache.GetJSON(ctx, key, &result)
		assert.Error(t, err)
	})
}

func TestCacheErrors(t *testing.T) {
	t.Run("Error Types", func(t *testing.T) {
		assert.Error(t, ErrCacheMiss)
		assert.Error(t, ErrCacheClosed)
		assert.Equal(t, "cache miss", ErrCacheMiss.Error())
		assert.Equal(t, "cache is closed", ErrCacheClosed.Error())
	})
}

// Benchmark tests
func BenchmarkMemoryCacheSet(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	value := []byte("benchmark-value")
	expiration := 1 * time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench-key-" + string(rune(i))
		cache.Set(ctx, key, value, expiration)
	}
}

func BenchmarkMemoryCacheGet(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	value := []byte("benchmark-value")
	expiration := 1 * time.Hour

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := "bench-key-" + string(rune(i))
		cache.Set(ctx, key, value, expiration)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench-key-" + string(rune(i%1000))
		cache.Get(ctx, key)
	}
}

func BenchmarkJSONCacheSetJSON(b *testing.B) {
	memCache := NewMemoryCache()
	defer memCache.Close()

	jsonCache := NewJSONCache(memCache)
	ctx := context.Background()

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	data := TestStruct{Name: "benchmark", Value: 42}
	expiration := 1 * time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "json-bench-key-" + string(rune(i))
		jsonCache.SetJSON(ctx, key, data, expiration)
	}
}
