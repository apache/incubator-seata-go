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
	"fmt"
	"sync"
)

var (
	once  sync.Once
	cache *UndoLogParserCache
)

// The type Undo log parser factory.
type UndoLogParserCache struct {
	// serializerName --> UndoLogParser
	serializerNameToParser map[string]UndoLogParser
}

func initCache() {
	cache = &UndoLogParserCache{
		serializerNameToParser: make(map[string]UndoLogParser, 0),
	}

	cache.store(&JsonParser{})
}

func GetCache() *UndoLogParserCache {
	once.Do(initCache)

	return cache
}

// Gets default UndoLogParser instance.
// return the instance
func (ulpc *UndoLogParserCache) GetDefault() (UndoLogParser, error) {
	return ulpc.Load(DefaultSerializer)
}

// Gets UndoLogParser by name
// param name parser name
// return the UndoLogParser
func (ulpc *UndoLogParserCache) Load(name string) (UndoLogParser, error) {
	if parser, ok := ulpc.serializerNameToParser[name]; ok && parser != nil {
		return parser, nil
	}

	return nil, fmt.Errorf("undo log parser type %v not found", name)
}

func (ulpc *UndoLogParserCache) store(parser UndoLogParser) {
	ulpc.serializerNameToParser[parser.GetName()] = parser
}
