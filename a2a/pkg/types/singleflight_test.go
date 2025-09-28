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

package types

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleFlight(t *testing.T) {
	t.Run("Basic Functionality", func(t *testing.T) {
		sf := NewSingleFlight()
		key := "test-key"
		expectedResult := "test-result"

		result, err := sf.Do(key, func() (interface{}, error) {
			return expectedResult, nil
		})

		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("Error Handling", func(t *testing.T) {
		sf := NewSingleFlight()
		key := "error-key"
		expectedError := errors.New("test error")

		result, err := sf.Do(key, func() (interface{}, error) {
			return nil, expectedError
		})

		assert.Equal(t, expectedError, err)
		assert.Nil(t, result)
	})

	t.Run("Single Execution with Multiple Callers", func(t *testing.T) {
		sf := NewSingleFlight()
		key := "single-exec-key"

		var callCount int32
		var wg sync.WaitGroup
		numGoroutines := 10
		results := make([]interface{}, numGoroutines)
		errors := make([]error, numGoroutines)

		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				defer wg.Done()

				result, err := sf.Do(key, func() (interface{}, error) {
					atomic.AddInt32(&callCount, 1)
					time.Sleep(10 * time.Millisecond) // Simulate work
					return "shared-result", nil
				})

				results[index] = result
				errors[index] = err
			}(i)
		}

		wg.Wait()

		// Function should only be called once
		assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))

		// All goroutines should get the same result
		for i := 0; i < numGoroutines; i++ {
			require.NoError(t, errors[i])
			assert.Equal(t, "shared-result", results[i])
		}
	})

	t.Run("Different Keys Execute Separately", func(t *testing.T) {
		sf := NewSingleFlight()

		var callCount int32
		var wg sync.WaitGroup
		numKeys := 5

		wg.Add(numKeys)

		for i := 0; i < numKeys; i++ {
			go func(keyIndex int) {
				defer wg.Done()

				key := "key-" + string(rune('A'+keyIndex))
				expectedResult := "result-" + string(rune('A'+keyIndex))

				result, err := sf.Do(key, func() (interface{}, error) {
					atomic.AddInt32(&callCount, 1)
					return expectedResult, nil
				})

				require.NoError(t, err)
				assert.Equal(t, expectedResult, result)
			}(i)
		}

		wg.Wait()

		// Each key should execute its function once
		assert.Equal(t, int32(numKeys), atomic.LoadInt32(&callCount))
	})

	t.Run("Forget Functionality", func(t *testing.T) {
		sf := NewSingleFlight()
		key := "forget-key"

		// Start a long-running operation
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			sf.Do(key, func() (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return "first-result", nil
			})
		}()

		// Give it a moment to start
		time.Sleep(10 * time.Millisecond)

		// Forget the key
		sf.Forget(key)

		// Now a new call should execute separately
		result, err := sf.Do(key, func() (interface{}, error) {
			return "second-result", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "second-result", result)

		wg.Wait()
	})

	t.Run("Concurrent Forget and Do", func(t *testing.T) {
		sf := NewSingleFlight()
		key := "concurrent-forget-key"

		var wg sync.WaitGroup
		var results []interface{}
		var mu sync.Mutex

		numGoroutines := 20
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				defer wg.Done()

				if index%5 == 0 {
					// Forget periodically
					sf.Forget(key)
				} else {
					result, err := sf.Do(key, func() (interface{}, error) {
						return "result", nil
					})

					if err == nil {
						mu.Lock()
						results = append(results, result)
						mu.Unlock()
					}
				}
			}(i)
		}

		wg.Wait()

		// Should have some results
		mu.Lock()
		assert.Greater(t, len(results), 0)
		mu.Unlock()
	})

	t.Run("Panic Recovery", func(t *testing.T) {
		sf := NewSingleFlight()
		key := "panic-key"

		// This should not panic the test
		assert.NotPanics(t, func() {
			result, err := sf.Do(key, func() (interface{}, error) {
				panic("test panic")
			})
			// The panic should be converted to an error
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	})

	t.Run("Nil Function", func(t *testing.T) {
		sf := NewSingleFlight()
		key := "nil-func-key"

		// This should handle nil function gracefully
		assert.NotPanics(t, func() {
			result, err := sf.Do(key, nil)
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	})
}

// Test internal state management
func TestSingleFlightInternalState(t *testing.T) {
	t.Run("Map Initialization", func(t *testing.T) {
		sf := &SingleFlight{}

		// Map should be initialized on first use
		result, err := sf.Do("test", func() (interface{}, error) {
			return "initialized", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "initialized", result)
		assert.NotNil(t, sf.m)
	})

	t.Run("Cleanup After Completion", func(t *testing.T) {
		sf := NewSingleFlight()
		key := "cleanup-key"

		_, err := sf.Do(key, func() (interface{}, error) {
			return "result", nil
		})
		require.NoError(t, err)

		// The call should be cleaned up from the map
		sf.mu.Lock()
		_, exists := sf.m[key]
		sf.mu.Unlock()

		assert.False(t, exists, "Call should be cleaned up after completion")
	})
}

// Benchmark tests
func BenchmarkSingleFlightDo(b *testing.B) {
	sf := NewSingleFlight()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := "benchmark-key"
			sf.Do(key, func() (interface{}, error) {
				return "result", nil
			})
		}
	})
}

func BenchmarkSingleFlightDoSeparateKeys(b *testing.B) {
	sf := NewSingleFlight()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key-" + string(rune(i%100))
			sf.Do(key, func() (interface{}, error) {
				return "result", nil
			})
			i++
		}
	})
}

func BenchmarkSingleFlightForget(b *testing.B) {
	sf := NewSingleFlight()
	key := "forget-benchmark-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sf.Forget(key)
	}
}
