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
	tests := []struct {
		name     string
		input    []driver.NamedValue
		expected []driver.Value
	}{
		{
			name:     "empty slice",
			input:    []driver.NamedValue{},
			expected: []driver.Value{},
		},
		{
			name: "single value",
			input: []driver.NamedValue{
				{Name: "param1", Ordinal: 1, Value: "test"},
			},
			expected: []driver.Value{"test"},
		},
		{
			name: "multiple values",
			input: []driver.NamedValue{
				{Name: "param1", Ordinal: 1, Value: "test1"},
				{Name: "param2", Ordinal: 2, Value: int64(123)},
				{Name: "param3", Ordinal: 3, Value: true},
			},
			expected: []driver.Value{"test1", int64(123), true},
		},
		{
			name: "nil values",
			input: []driver.NamedValue{
				{Name: "param1", Ordinal: 1, Value: nil},
				{Name: "param2", Ordinal: 2, Value: "test"},
			},
			expected: []driver.Value{nil, "test"},
		},
		{
			name: "various types",
			input: []driver.NamedValue{
				{Value: int64(42)},
				{Value: float64(3.14)},
				{Value: []byte("bytes")},
				{Value: true},
			},
			expected: []driver.Value{int64(42), float64(3.14), []byte("bytes"), true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NamedValueToValue(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, len(tt.expected), len(result))
		})
	}
}

func TestValueToNamedValue(t *testing.T) {
	tests := []struct {
		name     string
		input    []driver.Value
		expected []driver.NamedValue
	}{
		{
			name:     "empty slice",
			input:    []driver.Value{},
			expected: []driver.NamedValue{},
		},
		{
			name:  "single value",
			input: []driver.Value{"test"},
			expected: []driver.NamedValue{
				{Value: "test", Ordinal: 0},
			},
		},
		{
			name:  "multiple values",
			input: []driver.Value{"test1", int64(123), true},
			expected: []driver.NamedValue{
				{Value: "test1", Ordinal: 0},
				{Value: int64(123), Ordinal: 1},
				{Value: true, Ordinal: 2},
			},
		},
		{
			name:  "nil values",
			input: []driver.Value{nil, "test", nil},
			expected: []driver.NamedValue{
				{Value: nil, Ordinal: 0},
				{Value: "test", Ordinal: 1},
				{Value: nil, Ordinal: 2},
			},
		},
		{
			name:  "various types",
			input: []driver.Value{int64(42), float64(3.14), []byte("bytes"), true},
			expected: []driver.NamedValue{
				{Value: int64(42), Ordinal: 0},
				{Value: float64(3.14), Ordinal: 1},
				{Value: []byte("bytes"), Ordinal: 2},
				{Value: true, Ordinal: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValueToNamedValue(tt.input)
			assert.Equal(t, len(tt.expected), len(result))
			for i := range result {
				assert.Equal(t, tt.expected[i].Value, result[i].Value)
				assert.Equal(t, tt.expected[i].Ordinal, result[i].Ordinal)
			}
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	original := []driver.Value{"test", int64(123), true, nil}
	namedValues := ValueToNamedValue(original)
	converted := NamedValueToValue(namedValues)

	assert.Equal(t, original, converted)
}

func TestNamedValueToValuePreservesOrder(t *testing.T) {
	input := []driver.NamedValue{
		{Name: "z", Ordinal: 2, Value: "third"},
		{Name: "a", Ordinal: 0, Value: "first"},
		{Name: "m", Ordinal: 1, Value: "second"},
	}

	result := NamedValueToValue(input)
	expected := []driver.Value{"third", "first", "second"}

	assert.Equal(t, expected, result)
}

func TestValueToNamedValueOrdinalSequence(t *testing.T) {
	input := []driver.Value{"a", "b", "c", "d", "e"}
	result := ValueToNamedValue(input)

	for i, nv := range result {
		assert.Equal(t, i, nv.Ordinal)
		assert.Equal(t, input[i], nv.Value)
	}
}