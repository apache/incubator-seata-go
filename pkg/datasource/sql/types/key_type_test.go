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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyTypeConstants(t *testing.T) {
	assert.Equal(t, KeyType("NULL"), Null)
	assert.Equal(t, KeyType("PRIMARY_KEY"), PrimaryKey)
}

func TestKeyType_Number(t *testing.T) {
	tests := []struct {
		name     string
		keyType  KeyType
		expected IndexType
	}{
		{"Null key type", Null, IndexType(0)},
		{"Primary key type", PrimaryKey, IndexType(1)},
		{"Unknown key type", KeyType("UNKNOWN"), IndexType(0)},
		{"Empty key type", KeyType(""), IndexType(0)},
		{"Custom key type", KeyType("CUSTOM"), IndexType(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.keyType.Number()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKeyType_String(t *testing.T) {
	// Test that KeyType behaves as a string
	assert.Equal(t, "NULL", string(Null))
	assert.Equal(t, "PRIMARY_KEY", string(PrimaryKey))
}

func TestKeyType_Comparison(t *testing.T) {
	// Test KeyType comparison
	assert.True(t, Null == KeyType("NULL"))
	assert.True(t, PrimaryKey == KeyType("PRIMARY_KEY"))
	assert.False(t, Null == PrimaryKey)
	assert.False(t, Null == KeyType("UNKNOWN"))
}

func TestKeyType_CaseSensitive(t *testing.T) {
	// Test that KeyType is case sensitive
	assert.False(t, Null == KeyType("null"))
	assert.False(t, PrimaryKey == KeyType("primary_key"))
	assert.False(t, PrimaryKey == KeyType("Primary_Key"))
}

func TestKeyType_NumberMapping(t *testing.T) {
	// Test that the Number() method maps to correct IndexType values
	assert.Equal(t, IndexTypeNull, Null.Number())
	assert.Equal(t, IndexTypePrimaryKey, PrimaryKey.Number())
}
