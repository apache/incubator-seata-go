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
	"math/rand"
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

func TestLz4DecompressCorruptBlockDoesNotAllocate(t *testing.T) {
	// A corrupt block used to walk the whole growth ladder because the decoder
	// reports "destination too small" and "corrupt input" with the same error.
	// The size is now derived from the token stream, so a corrupt block is
	// rejected before a destination buffer exists and the allocation does not
	// grow with the corrupt payload.
	lz4 := Lz4{}

	for _, size := range []int{64 << 10, 1 << 20} {
		data := make([]byte, size)
		_, err := rand.New(rand.NewSource(int64(size))).Read(data)
		assert.NoError(t, err, "generate corrupt payload of %d bytes", size)

		var failed bool
		allocs := testing.AllocsPerRun(10, func() {
			out, err := lz4.Decompress(data)
			if err == nil || out != nil {
				failed = true
			}
		})
		assert.False(t, failed, "corrupt payload of %d bytes must be rejected", size)
		assert.Equal(t, 0.0, allocs, "corrupt payload of %d bytes allocated memory", size)
	}
}

func TestLz4DecompressEmptyInput(t *testing.T) {
	lz4 := Lz4{}
	decompressed, err := lz4.Decompress(nil)
	assert.NoError(t, err)
	assert.Empty(t, decompressed)
}

func TestLz4DecompressOverflowLiteralLength(t *testing.T) {
	// A 0xF literal token followed by a long run of 0xFF extension bytes can
	// overflow the int accumulator on 32-bit platforms. The overflow guard in
	// lz4DecodedSize must reject such a block with an error instead of panicking
	// on a negative index or a negative make() length.
	lz4 := Lz4{}

	// 0xF0: literal nibble = 0xF (extended), match nibble = 0.
	data := append([]byte{0xF0}, bytes.Repeat([]byte{0xFF}, 1<<24)...)

	out, err := lz4.Decompress(data)
	assert.Error(t, err, "overflowing literal length must be rejected")
	assert.Nil(t, out)
}

func TestLz4DecompressOverflowMatchLength(t *testing.T) {
	// Same overflow scenario for the match-length extension bytes. The token
	// carries a literal of 1 byte so the block reaches the match section.
	lz4 := Lz4{}

	// 0x1F: literal nibble = 1, match nibble = 0xF (extended).
	// One literal byte, then a 2-byte offset (1), then the 0xFF run.
	data := []byte{0x1F, 0x00, 0x01, 0x00}
	data = append(data, bytes.Repeat([]byte{0xFF}, 1<<24)...)

	out, err := lz4.Decompress(data)
	assert.Error(t, err, "overflowing match length must be rejected")
	assert.Nil(t, out)
}

func TestLz4DecompressTrailingLiteralsRespectCeiling(t *testing.T) {
	// A block can end on a literal run, and that path used to return before the
	// decoded-size ceiling was checked, so the effective allocation ceiling was
	// lz4DecompressMaxBufferSize + len(in). Fill the match section up to just
	// below the ceiling and let the trailing literals push it over.
	lz4 := Lz4{}

	// 0x10: one literal byte 0x61, then a match with offset 1 and length 0.
	// Decoding it yields five bytes and leaves a valid offset for the next
	// sequence.
	data := []byte{0x10, 0x61, 0x01, 0x00}

	// 0x0F: extended match with offset 1. The extension run declares a match
	// that ends just below the ceiling.
	const trailing = 402
	match := lz4DecompressMaxBufferSize - trailing - 8
	fill := match - 15
	data = append(data, 0x0F, 0x01, 0x00)
	data = append(data, bytes.Repeat([]byte{0xFF}, fill/255)...)
	data = append(data, byte(fill%255))

	// 0xF0: extended literal run with no match, carrying exactly `trailing`
	// literal bytes after the extension bytes.
	data = append(data, 0xF0, 0xFF, byte(trailing-15-255))
	data = append(data, bytes.Repeat([]byte{0x62}, trailing)...)

	out, err := lz4.Decompress(data)
	assert.Error(t, err, "trailing literals must not bypass the size ceiling")
	assert.Nil(t, out)
}
