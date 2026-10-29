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
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLz4Compress(t *testing.T) {
	sample := strings.Repeat("hello world", 100)

	lz4 := Lz4{}

	compressResult, err := lz4.Compress([]byte(sample))
	assert.NoError(t, err)
	t.Logf("Compressed result: %v", string(compressResult))

	decompressResult, err := lz4.Decompress(compressResult)
	assert.NoError(t, err)
	assert.Equal(t, sample, string(decompressResult))
	t.Logf("Decompressed result: %v", string(decompressResult))
}

func TestLz4DecompressHighlyCompressibleData(t *testing.T) {
	// The LZ4 block format does not record the size of the original data, so
	// Decompress has to provide a destination buffer that is large enough for
	// the whole block. Payloads that compress better than 100:1 -- zero
	// filled columns, long repeated runs, sparse log or SQL text -- used to
	// fail with "lz4: invalid source or destination buffer too short", which
	// makes the AT undo log and every compressed RPC body unreadable.
	lz4 := Lz4{}

	for _, size := range []int{1 << 12, 1 << 16, 1 << 20} {
		sample := make([]byte, size)

		compressed, err := lz4.Compress(sample)
		assert.NoError(t, err, "compress size=%d", size)

		decompressed, err := lz4.Decompress(compressed)
		assert.NoError(t, err, "decompress size=%d compressed to %d bytes", size, len(compressed))
		assert.Equal(t, size, len(decompressed), "decompressed size for input size=%d", size)
		assert.True(t, bytes.Equal(sample, decompressed), "content mismatch for input size=%d", size)
	}
}

func TestLz4DecompressRepetitiveText(t *testing.T) {
	sample := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 1<<14)
	lz4 := Lz4{}

	compressed, err := lz4.Compress([]byte(sample))
	assert.NoError(t, err)

	decompressed, err := lz4.Decompress(compressed)
	assert.NoError(t, err, "text of %d bytes compressed to %d bytes", len(sample), len(compressed))
	assert.Equal(t, len(sample), len(decompressed))
	assert.True(t, bytes.Equal([]byte(sample), decompressed), "repetitive text round trip mismatch")
}

func TestLz4DecompressInvalidData(t *testing.T) {
	// Malformed blocks must still be reported as errors instead of being
	// retried forever or truncated to a wrong result.
	lz4 := Lz4{}
	decompressed, err := lz4.Decompress([]byte{0x80, 0x01, 0xff, 0xfe, 0x7f, 0x00, 0x00, 0x00, 0x42, 0x00})
	assert.Error(t, err)
	assert.Nil(t, decompressed)
}

func TestLz4DecompressEmptyInput(t *testing.T) {
	lz4 := Lz4{}
	decompressed, err := lz4.Decompress(nil)
	assert.NoError(t, err)
	assert.Empty(t, decompressed)
}
