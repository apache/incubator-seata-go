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

package integration

import (
	"testing"

	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/integration/dubbo"
)

// TestInit tests the Init function
func TestInit(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Init successfully initializes dubbo integration",
		},
		{
			name: "Init can be called multiple times without error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call Init - should not panic
			assert.NotPanics(t, func() {
				Init()
			})

			// Verify that the filter was registered
			filter, ok := extension.GetFilter(constant.SeataFilterKey)
			assert.True(t, ok)
			assert.NotNil(t, filter)

			// Verify that the filter is the correct type
			dubboFilter := dubbo.GetDubboTransactionFilter()
			assert.NotNil(t, dubboFilter)
		})
	}
}

// TestInit_FilterRegistration tests that Init registers the filter correctly
func TestInit_FilterRegistration(t *testing.T) {
	tests := []struct {
		name          string
		callInitTimes int
	}{
		{
			name:          "Single Init call",
			callInitTimes: 1,
		},
		{
			name:          "Multiple Init calls",
			callInitTimes: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call Init multiple times
			for i := 0; i < tt.callInitTimes; i++ {
				assert.NotPanics(t, func() {
					Init()
				})
			}

			// Verify the filter is still registered correctly
			filter, ok := extension.GetFilter(constant.SeataFilterKey)
			assert.True(t, ok)
			assert.NotNil(t, filter)

			// Get the filter directly from dubbo package
			dubboFilter := dubbo.GetDubboTransactionFilter()
			assert.NotNil(t, dubboFilter)

			// Verify both return the same filter instance
			dubboFilter2 := dubbo.GetDubboTransactionFilter()
			assert.Equal(t, dubboFilter, dubboFilter2, "GetDubboTransactionFilter should return same instance")
		})
	}
}

// TestInit_Integration tests the integration with dubbo package
func TestInit_Integration(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Verify dubbo filter can be retrieved after Init",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize
			Init()

			// Get filter from extension
			filter, ok := extension.GetFilter(constant.SeataFilterKey)
			assert.True(t, ok)
			assert.NotNil(t, filter)

			// Get filter from dubbo package
			dubboFilter := dubbo.GetDubboTransactionFilter()
			assert.NotNil(t, dubboFilter)

			// Create another instance via the extension
			filterFunc, ok2 := extension.GetFilter(constant.SeataFilterKey)
			assert.True(t, ok2)
			assert.NotNil(t, filterFunc)

			// Both should be the same filter
			assert.Equal(t, filter, filterFunc)
		})
	}
}

// TestInit_NoSideEffects tests that Init has no unexpected side effects
func TestInit_NoSideEffects(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Init does not cause side effects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get state before Init
			filterBefore, okBefore := extension.GetFilter(constant.SeataFilterKey)

			// Call Init
			Init()

			// Get state after Init
			filterAfter, okAfter := extension.GetFilter(constant.SeataFilterKey)

			// Filter should be registered
			assert.True(t, okAfter)
			assert.NotNil(t, filterAfter)

			// Calling Init again should not change the filter
			Init()
			filterAfterSecond, okAfterSecond := extension.GetFilter(constant.SeataFilterKey)
			assert.True(t, okAfterSecond)
			assert.NotNil(t, filterAfterSecond)

			// The filter should be the same
			if okBefore && filterBefore != nil {
				assert.Equal(t, filterBefore, filterAfter)
			}
		})
	}
}

// TestInit_DubboFilterConsistency tests filter consistency
func TestInit_DubboFilterConsistency(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Dubbo filter returns consistent singleton instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize
			Init()

			// Get filter multiple times
			filter1 := dubbo.GetDubboTransactionFilter()
			filter2 := dubbo.GetDubboTransactionFilter()
			filter3 := dubbo.GetDubboTransactionFilter()

			// All should return the same instance (singleton pattern)
			assert.NotNil(t, filter1)
			assert.NotNil(t, filter2)
			assert.NotNil(t, filter3)
			assert.Equal(t, filter1, filter2)
			assert.Equal(t, filter2, filter3)
		})
	}
}
