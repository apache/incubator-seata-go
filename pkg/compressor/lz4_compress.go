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
	"fmt"

	"github.com/pierrec/lz4/v4"
)

type Lz4 struct {
}

const (
	// lz4DecompressMaxBufferSize caps one decoded block so a hostile payload cannot
	// trick us into multi-MiB (or worse) allocations before the integrity check.
	// 64 MiB comfortably exceeds any realistic undo row while keeping the ceiling
	// tight enough to bound worst-case memory in a corrupted-input path.
	lz4DecompressMaxBufferSize = 0x04000000
)

func (l *Lz4) Compress(data []byte) ([]byte, error) {

	buffer := make([]byte, lz4.CompressBlockBound(len(data)))

	var compressor lz4.Compressor

	n, err := compressor.CompressBlock(data, buffer)
	if err != nil {
		return nil, err
	}
	if n >= len(data) {
		return nil, fmt.Errorf("`%s` is not compressible", string(data))
	}

	return buffer[:n], nil
}

func (l *Lz4) Decompress(in []byte) ([]byte, error) {
	// The decoder reports "destination too small" and "corrupt input" with the
	// same error, so growing the destination until the block decodes cannot tell
	// them apart: a corrupt block would try every size up to the ceiling. Walk
	// the token stream first - that yields the exact output size without writing
	// anything - then decode once into a buffer of that size.
	size, err := lz4DecodedSize(in)
	if err != nil {
		return nil, err
	}

	out := make([]byte, size)
	n, err := lz4.UncompressBlock(in, out)
	if err != nil {
		return nil, err
	}

	return out[:n], nil
}

// lz4DecodedSize returns the number of bytes that an LZ4 block decodes to by
// walking its token stream, which is what UncompressBlock does with the data.
func lz4DecodedSize(in []byte) (int, error) {
	var size int

	for si := 0; si < len(in); {
		token := in[si]
		si++

		literals := int(token >> 4)
		if literals == 0xF {
			for {
				if si >= len(in) {
					return 0, lz4.ErrInvalidSourceShortBuffer
				}
				b := in[si]
				si++
				literals += int(b)
				// Guard against int overflow on 32-bit platforms: a long run of
				// 0xFF extension bytes can push literals past MaxInt, after which
				// the negative value would bypass the length check below and walk
				// si backwards. Mirror the upstream decoder's overflow guard.
				if literals < 0 || literals > len(in) {
					return 0, lz4.ErrInvalidSourceShortBuffer
				}
				if b != 0xFF {
					break
				}
			}
		}
		if literals > len(in)-si {
			return 0, lz4.ErrInvalidSourceShortBuffer
		}
		si += literals
		size += literals

		if si == len(in) {
			// A block may end on literals, but only when the token carries no
			// match, which is what the last sequence of a valid block looks like.
			if token&0xF != 0 {
				return 0, lz4.ErrInvalidSourceShortBuffer
			}
			return size, nil
		}

		if len(in)-si < 2 {
			return 0, lz4.ErrInvalidSourceShortBuffer
		}
		// The 16-bit match offset must point into already-produced output.
		offset := int(in[si]) | int(in[si+1])<<8
		if offset == 0 || offset > size {
			// Offset zero cannot refer to earlier output; offset past the
			// currently produced bytes is impossible in a valid block.
			return 0, lz4.ErrInvalidSourceShortBuffer
		}
		si += 2

		match := int(token & 0xF)
		if match == 0xF {
			for {
				if si >= len(in) {
					return 0, lz4.ErrInvalidSourceShortBuffer
				}
				b := in[si]
				si++
				match += int(b)
				// Same 32-bit overflow guard as the literals accumulation above.
				if match < 0 || match > lz4DecompressMaxBufferSize {
					return 0, lz4.ErrInvalidSourceShortBuffer
				}
				if b != 0xFF {
					break
				}
			}
		}
		// The four bytes at the match offset are the minimum match length.
		size += match + 4
		if size > lz4DecompressMaxBufferSize {
			return 0, fmt.Errorf("lz4: decoded size exceeds %d bytes", lz4DecompressMaxBufferSize)
		}
	}

	// Only empty input decodes to nothing without a token.
	return size, nil
}

func (l *Lz4) GetCompressorType() CompressorType {
	return CompressorLz4
}
