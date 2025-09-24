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

package util

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamedValueToValue(t *testing.T) {
	// Test normal conversion
	nvs := []driver.NamedValue{
		{Value: "test1", Ordinal: 1},
		{Value: 123, Ordinal: 2},
		{Value: true, Ordinal: 3},
	}

	vs := NamedValueToValue(nvs)
	assert.Len(t, vs, 3)
	assert.Equal(t, "test1", vs[0])
	assert.Equal(t, 123, vs[1])
	assert.Equal(t, true, vs[2])

	// Test empty slice
	emptyNvs := []driver.NamedValue{}
	emptyVs := NamedValueToValue(emptyNvs)
	assert.Len(t, emptyVs, 0)
}

func TestValueToNamedValue(t *testing.T) {
	// Test normal conversion
	vs := []driver.Value{
		"test1",
		123,
		true,
	}

	nvs := ValueToNamedValue(vs)
	assert.Len(t, nvs, 3)
	assert.Equal(t, "test1", nvs[0].Value)
	assert.Equal(t, 123, nvs[1].Value)
	assert.Equal(t, true, nvs[2].Value)
	assert.Equal(t, 0, nvs[0].Ordinal)
	assert.Equal(t, 1, nvs[1].Ordinal)
	assert.Equal(t, 2, nvs[2].Ordinal)

	// Test empty slice
	emptyVs := []driver.Value{}
	emptyNvs := ValueToNamedValue(emptyVs)
	assert.Len(t, emptyNvs, 0)
}
