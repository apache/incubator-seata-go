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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitCache(t *testing.T) {
	// Reset cache to test initialization
	cache = nil
	assert.Nil(t, cache)
	initCache()
	assert.NotNil(t, cache)
}

func TestGetCache(t *testing.T) {
	assert.NotNil(t, GetCache())
}

func TestGetDefault(t *testing.T) {
	logParser, err := GetCache().GetDefault()
	assert.Nil(t, err)
	assert.NotNil(t, logParser)
	assert.Equal(t, DefaultSerializer, logParser.GetName())
}

func TestLoad(t *testing.T) {
	jsonParser, err := GetCache().Load("json")
	assert.Nil(t, err)
	assert.NotNil(t, jsonParser)
}

func TestLoadJsonParser(t *testing.T) {
	cache := GetCache()
	parser, err := cache.Load("json")
	assert.NoError(t, err)
	assert.NotNil(t, parser)
	assert.Equal(t, "json", parser.GetName())
	assert.IsType(t, &JsonParser{}, parser)
}

func TestLoadProtobufParser(t *testing.T) {
	cache := GetCache()
	parser, err := cache.Load("protobuf")
	assert.NoError(t, err)
	assert.NotNil(t, parser)
	assert.Equal(t, "protobuf", parser.GetName())
	assert.IsType(t, &ProtobufParser{}, parser)
}

func TestLoadNonExistentParser(t *testing.T) {
	cache := GetCache()
	parser, err := cache.Load("non-existent")
	assert.Error(t, err)
	assert.Nil(t, parser)
	assert.Contains(t, err.Error(), "undo log parser type non-existent not found")
}

func TestLoadEmptyName(t *testing.T) {
	cache := GetCache()
	parser, err := cache.Load("")
	assert.Error(t, err)
	assert.Nil(t, parser)
	assert.Contains(t, err.Error(), "undo log parser type  not found")
}

func TestCacheInitialization(t *testing.T) {
	cache := GetCache()
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.serializerNameToParser)
	assert.Len(t, cache.serializerNameToParser, 2)

	_, existsJson := cache.serializerNameToParser["json"]
	assert.True(t, existsJson)

	_, existsProtobuf := cache.serializerNameToParser["protobuf"]
	assert.True(t, existsProtobuf)
}

func TestCacheSingleton(t *testing.T) {
	cache1 := GetCache()
	cache2 := GetCache()
	assert.Same(t, cache1, cache2)
}

func TestStoreMethod(t *testing.T) {
	cache := GetCache()
	
	mockParser := &mockUndoLogParser{
		name:           "test-mock",
		defaultContent: []byte("test"),
		shouldError:    false,
	}
	
	cache.store(mockParser)
	
	loadedParser, err := cache.Load("test-mock")
	assert.NoError(t, err)
	assert.NotNil(t, loadedParser)
	assert.Equal(t, "test-mock", loadedParser.GetName())
	assert.Same(t, mockParser, loadedParser)
}
