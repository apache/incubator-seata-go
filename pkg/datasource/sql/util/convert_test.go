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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertDbVersion(t *testing.T) {
	version1 := "3.1.2"
	v1Int, err1 := ConvertDbVersion(version1)
	assert.NoError(t, err1)

	version2 := "3.1.3"
	v2Int, err2 := ConvertDbVersion(version2)
	assert.NoError(t, err2)

	assert.Less(t, v1Int, v2Int)

	version3 := "3.1.3"
	v3Int, err3 := ConvertDbVersion(version3)
	assert.NoError(t, err3)
	assert.Equal(t, v2Int, v3Int)
}
