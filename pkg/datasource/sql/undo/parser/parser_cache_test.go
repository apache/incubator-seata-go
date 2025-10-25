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

package parser

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitCache(t *testing.T) {
	// Reset the cache for testing
	cache = nil
	once = sync.Once{}

	assert.Nil(t, cache)
	initCache()
	assert.NotNil(t, cache)

	// Test that cache is properly initialized with parsers
	assert.NotNil(t, cache.serializerNameToParser["json"])
	assert.NotNil(t, cache.serializerNameToParser["protobuf"])
}

func TestGetCache(t *testing.T) {
	// Reset the cache for testing
	cache = nil
	once = sync.Once{}

	assert.NotNil(t, GetCache())

	// Test singleton behavior
	cache1 := GetCache()
	cache2 := GetCache()
	assert.Equal(t, cache1, cache2)
}

func TestGetDefault(t *testing.T) {
	// Reset the cache for testing
	cache = nil
	once = sync.Once{}

	logParser, err := GetCache().GetDefault()
	assert.Nil(t, err)
	assert.NotNil(t, logParser)
	assert.Equal(t, DefaultSerializer, logParser.GetName())
}

func TestLoad(t *testing.T) {
	// Reset the cache for testing
	cache = nil
	once = sync.Once{}

	jsonParser, err := GetCache().Load("json")
	assert.Nil(t, err)
	assert.NotNil(t, jsonParser)
	assert.Equal(t, "json", jsonParser.GetName())

	// Test loading protobuf parser
	protobufParser, err := GetCache().Load("protobuf")
	assert.Nil(t, err)
	assert.NotNil(t, protobufParser)
	assert.Equal(t, "protobuf", protobufParser.GetName())

	// Test loading non-existent parser
	nonExistentParser, err := GetCache().Load("non-existent")
	assert.NotNil(t, err)
	assert.Nil(t, nonExistentParser)
	assert.Equal(t, "undo log parser type non-existent not found", err.Error())
}

func TestStore(t *testing.T) {
	// Reset the cache for testing
	cache = nil
	once = sync.Once{}
	GetCache()

	// Create a mock parser
	mockParser := &JsonParser{}

	// Store the parser
	cache.store(mockParser)

	// Verify it's stored
	storedParser, err := cache.Load("json")
	assert.Nil(t, err)
	assert.Equal(t, mockParser, storedParser)
}
