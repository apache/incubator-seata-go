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

package compressor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLz4Compress(t *testing.T) {
	sample := "test"

	lz4 := new(Lz4)

	compressResult, err := lz4.Compress([]byte(sample))
	assert.NoError(t, err)
	t.Logf("Compressed result: %v", string(compressResult))

	decompressResult, err := lz4.Decompress(compressResult)
	assert.NoError(t, err)
	t.Logf("Decompressed result: %v", string(decompressResult))
}
