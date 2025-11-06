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

func TestLoad_NotFound(t *testing.T) {
	parser, err := GetCache().Load("nonexistent")
	assert.NotNil(t, err)
	assert.Nil(t, parser)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoad_Protobuf(t *testing.T) {
	protobufParser, err := GetCache().Load("protobuf")
	assert.Nil(t, err)
	assert.NotNil(t, protobufParser)
	assert.Equal(t, "protobuf", protobufParser.GetName())
}

func TestUndoLogParserCache_Store(t *testing.T) {
	cache := GetCache()
	assert.NotNil(t, cache)

	// Verify both parsers are stored
	jsonParser, err := cache.Load("json")
	assert.NoError(t, err)
	assert.NotNil(t, jsonParser)

	protobufParser, err := cache.Load("protobuf")
	assert.NoError(t, err)
	assert.NotNil(t, protobufParser)
}

func TestDefaultSerializer(t *testing.T) {
	assert.Equal(t, "json", DefaultSerializer)
}
