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

package datasource

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testInitOnce sync.Once
)

// ensureInitialized ensures Init is called exactly once for all tests
func ensureInitialized() {
	testInitOnce.Do(func() {
		Init()
	})
}

// TestInitFunction tests that the Init function can be called without panic
func TestInitFunction(t *testing.T) {
	// Verify Init() can be called without panic
	assert.NotPanics(t, func() {
		ensureInitialized()
	}, "Init() should not panic")
}

// TestInitRegistersDrivers tests that Init properly registers the SQL drivers
func TestInitRegistersDrivers(t *testing.T) {
	ensureInitialized()

	// Verify that the seata-at-mysql driver is registered
	drivers := sql.Drivers()
	assert.Contains(t, drivers, "seata-at-mysql", "seata-at-mysql driver should be registered")

	// Verify that the seata-xa-mysql driver is registered
	assert.Contains(t, drivers, "seata-xa-mysql", "seata-xa-mysql driver should be registered")
}

// TestInitBothDriversExist verifies both AT and XA drivers are registered
func TestInitBothDriversExist(t *testing.T) {
	ensureInitialized()

	drivers := sql.Drivers()

	hasATDriver := false
	hasXADriver := false

	for _, driver := range drivers {
		if driver == "seata-at-mysql" {
			hasATDriver = true
		}
		if driver == "seata-xa-mysql" {
			hasXADriver = true
		}
	}

	assert.True(t, hasATDriver, "AT driver should be registered")
	assert.True(t, hasXADriver, "XA driver should be registered")
}

// TestInitDriverNames verifies the exact driver names
func TestInitDriverNames(t *testing.T) {
	ensureInitialized()

	drivers := sql.Drivers()

	// Check for exact driver names
	seataDriverCount := 0
	for _, driver := range drivers {
		if driver == "seata-at-mysql" {
			seataDriverCount++
		}
		if driver == "seata-xa-mysql" {
			seataDriverCount++
		}
	}

	assert.Equal(t, 2, seataDriverCount, "Exactly 2 Seata drivers should be registered")
}

// TestInitDriverOrder verifies drivers can be found in the driver list
func TestInitDriverOrder(t *testing.T) {
	ensureInitialized()

	drivers := sql.Drivers()

	atPos := -1
	xaPos := -1

	for i, driver := range drivers {
		if driver == "seata-at-mysql" {
			atPos = i
		}
		if driver == "seata-xa-mysql" {
			xaPos = i
		}
	}

	require.NotEqual(t, -1, atPos, "AT driver should be registered")
	require.NotEqual(t, -1, xaPos, "XA driver should be registered")

	t.Logf("AT driver at position %d, XA driver at position %d", atPos, xaPos)
}

// TestInitDriversPersist verifies drivers remain registered
func TestInitDriversPersist(t *testing.T) {
	ensureInitialized()

	// Check multiple times that drivers persist
	for i := 0; i < 3; i++ {
		drivers := sql.Drivers()
		assert.Contains(t, drivers, "seata-at-mysql", "Iteration %d: AT driver should persist", i+1)
		assert.Contains(t, drivers, "seata-xa-mysql", "Iteration %d: XA driver should persist", i+1)
	}
}

// TestInitDriverConstants verifies driver name constants
func TestInitDriverConstants(t *testing.T) {
	ensureInitialized()

	expectedDrivers := []string{"seata-at-mysql", "seata-xa-mysql"}
	drivers := sql.Drivers()

	for _, expected := range expectedDrivers {
		assert.Contains(t, drivers, expected, "Driver %s should be registered", expected)
	}
}

// TestInitIdempotentCheck documents that Init is not idempotent
// This test verifies the current behavior: calling Init() multiple times will panic
func TestInitIdempotentCheck(t *testing.T) {
	ensureInitialized()

	// Attempting to call Init() again should panic due to duplicate driver registration
	// This documents the current non-idempotent behavior
	assert.Panics(t, func() {
		Init()
	}, "Init() should panic when called multiple times (not idempotent)")
}

// TestInitOnlyOnce verifies that our test helper ensures single initialization
func TestInitOnlyOnce(t *testing.T) {
	// Call ensureInitialized multiple times
	// It should be safe because sync.Once guarantees single execution
	for i := 0; i < 5; i++ {
		assert.NotPanics(t, func() {
			ensureInitialized()
		}, "ensureInitialized() should be safe to call multiple times")
	}

	// Verify drivers are still properly registered
	drivers := sql.Drivers()
	assert.Contains(t, drivers, "seata-at-mysql")
	assert.Contains(t, drivers, "seata-xa-mysql")
}

// TestInitConcurrentAccess tests concurrent access to ensureInitialized
func TestInitConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	concurrency := 10

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()

			// Should not panic even when called concurrently
			assert.NotPanics(t, func() {
				ensureInitialized()
			}, "Goroutine %d: ensureInitialized() should not panic", id)

			// Verify drivers are available
			drivers := sql.Drivers()
			assert.Contains(t, drivers, "seata-at-mysql", "Goroutine %d: AT driver should exist", id)
			assert.Contains(t, drivers, "seata-xa-mysql", "Goroutine %d: XA driver should exist", id)
		}(i)
	}

	wg.Wait()
}

// TestInitDriverListStability verifies driver list stability
func TestInitDriverListStability(t *testing.T) {
	ensureInitialized()

	// Get driver list multiple times
	drivers1 := sql.Drivers()
	drivers2 := sql.Drivers()
	drivers3 := sql.Drivers()

	// Count Seata drivers in each
	countSeataDrivers := func(driverList []string) int {
		count := 0
		for _, d := range driverList {
			if d == "seata-at-mysql" || d == "seata-xa-mysql" {
				count++
			}
		}
		return count
	}

	count1 := countSeataDrivers(drivers1)
	count2 := countSeataDrivers(drivers2)
	count3 := countSeataDrivers(drivers3)

	assert.Equal(t, 2, count1, "First check should find 2 Seata drivers")
	assert.Equal(t, 2, count2, "Second check should find 2 Seata drivers")
	assert.Equal(t, 2, count3, "Third check should find 2 Seata drivers")
}

// TestInitPackageState verifies package initialization state
func TestInitPackageState(t *testing.T) {
	ensureInitialized()

	drivers := sql.Drivers()

	// Verify we have exactly the drivers we expect
	atFound := false
	xaFound := false

	for _, driver := range drivers {
		if driver == "seata-at-mysql" {
			atFound = true
		}
		if driver == "seata-xa-mysql" {
			xaFound = true
		}
	}

	assert.True(t, atFound, "Package should have AT driver registered")
	assert.True(t, xaFound, "Package should have XA driver registered")
}

// TestInitDriverNamesUnique verifies no duplicate driver registrations
func TestInitDriverNamesUnique(t *testing.T) {
	ensureInitialized()

	drivers := sql.Drivers()

	// Count occurrences of each Seata driver
	atCount := 0
	xaCount := 0

	for _, driver := range drivers {
		if driver == "seata-at-mysql" {
			atCount++
		}
		if driver == "seata-xa-mysql" {
			xaCount++
		}
	}

	assert.Equal(t, 1, atCount, "AT driver should be registered exactly once")
	assert.Equal(t, 1, xaCount, "XA driver should be registered exactly once")
}

// TestInitBasicFunctionality tests that Init performs its basic function
func TestInitBasicFunctionality(t *testing.T) {
	ensureInitialized()

	// After Init(), we should be able to query the driver list
	assert.NotPanics(t, func() {
		drivers := sql.Drivers()
		assert.NotNil(t, drivers, "Drivers list should not be nil")
		assert.Greater(t, len(drivers), 0, "Drivers list should not be empty")
	}, "Querying drivers after Init should not panic")
}

// TestInitRegistrationOrder verifies registration happens in expected order
func TestInitRegistrationOrder(t *testing.T) {
	ensureInitialized()

	// The order of registration is: AT first, then XA
	// We verify both are present (order may vary in the list)
	drivers := sql.Drivers()

	foundAT := false
	foundXA := false

	for _, driver := range drivers {
		if driver == "seata-at-mysql" {
			foundAT = true
		}
		if driver == "seata-xa-mysql" {
			foundXA = true
		}
	}

	assert.True(t, foundAT, "AT driver should be found (registered first)")
	assert.True(t, foundXA, "XA driver should be found (registered second)")
}

// TestInitNoExtraDrivers verifies only expected Seata drivers are registered
func TestInitNoExtraDrivers(t *testing.T) {
	ensureInitialized()

	drivers := sql.Drivers()

	// Count how many seata-* drivers exist
	seataDrivers := []string{}
	for _, driver := range drivers {
		if len(driver) >= 6 && driver[:6] == "seata-" {
			seataDrivers = append(seataDrivers, driver)
		}
	}

	// Should only have our two drivers
	assert.Len(t, seataDrivers, 2, "Should have exactly 2 seata-* drivers")
	assert.Contains(t, seataDrivers, "seata-at-mysql")
	assert.Contains(t, seataDrivers, "seata-xa-mysql")
}

// TestInitDriverRegistrationComplete verifies complete driver registration
func TestInitDriverRegistrationComplete(t *testing.T) {
	ensureInitialized()

	drivers := sql.Drivers()
	driverMap := make(map[string]bool)

	for _, driver := range drivers {
		driverMap[driver] = true
	}

	// Both drivers should be in the map
	assert.True(t, driverMap["seata-at-mysql"], "AT driver should be registered")
	assert.True(t, driverMap["seata-xa-mysql"], "XA driver should be registered")
}

// TestInitMultipleEnsureCalls verifies multiple ensureInitialized calls are safe
func TestInitMultipleEnsureCalls(t *testing.T) {
	// Call ensureInitialized many times
	for i := 0; i < 100; i++ {
		ensureInitialized()
	}

	// Drivers should still be properly registered
	drivers := sql.Drivers()
	assert.Contains(t, drivers, "seata-at-mysql", "AT driver should still be registered")
	assert.Contains(t, drivers, "seata-xa-mysql", "XA driver should still be registered")
}

// BenchmarkInitEnsureInitialized benchmarks the ensureInitialized function
func BenchmarkInitEnsureInitialized(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ensureInitialized()
	}
}

// BenchmarkInitDriverList benchmarks getting the driver list
func BenchmarkInitDriverList(b *testing.B) {
	ensureInitialized()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = sql.Drivers()
	}
}

// BenchmarkInitDriverCheck benchmarks checking for specific drivers
func BenchmarkInitDriverCheck(b *testing.B) {
	ensureInitialized()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		drivers := sql.Drivers()
		for _, driver := range drivers {
			if driver == "seata-at-mysql" {
				break
			}
		}
	}
}

// BenchmarkInitConcurrentDriverAccess benchmarks concurrent driver access
func BenchmarkInitConcurrentDriverAccess(b *testing.B) {
	ensureInitialized()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			drivers := sql.Drivers()
			_ = drivers
		}
	})
}
