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
	// The LZ4 block format does not record the size of the uncompressed data,
	// so Decompress has to guess how large the destination buffer should be.
	// lz4DecompressBufferSizeRatio is the first guess, expressed as a multiple
	// of the compressed length; the buffer is then doubled until the block
	// decodes.
	lz4DecompressBufferSizeRatio = 32
	// lz4DecompressBufferSizeFloor keeps tiny blocks from needing several
	// growth steps.
	lz4DecompressBufferSizeFloor = 4096
	// lz4DecompressMaxExpansionRatio bounds the growth loop. In the block
	// format a match sequence costs at least a token plus a 2-byte offset and
	// every extra length byte adds 255 output bytes, so a well formed block
	// expands by at most ~255 times its own size; a block that needs more than
	// this cannot be valid and is reported as corrupted instead.
	lz4DecompressMaxExpansionRatio = 512
	// lz4DecompressMaxBufferSize is the absolute ceiling for one block so that
	// a corrupted or hostile payload cannot make the decoder allocate
	// without bound.
	lz4DecompressMaxBufferSize = 0x40000000
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
	maxSize := len(in) * lz4DecompressMaxExpansionRatio
	if maxSize > lz4DecompressMaxBufferSize {
		maxSize = lz4DecompressMaxBufferSize
	}

	size := len(in) * lz4DecompressBufferSizeRatio
	if size < lz4DecompressBufferSizeFloor {
		size = lz4DecompressBufferSizeFloor
	}
	if size > maxSize {
		size = maxSize
	}

	// Grow the destination buffer until the whole block fits: a payload that
	// compresses better than the initial guess is valid LZ4 data and must not
	// fail to decompress.
	for {
		out := make([]byte, size)
		n, err := lz4.UncompressBlock(in, out)
		if err == nil {
			return out[:n], nil
		}
		if size >= maxSize {
			return nil, err
		}
		size *= 2
		if size > maxSize {
			size = maxSize
		}
	}
}

func (l *Lz4) GetCompressorType() CompressorType {
	return CompressorLz4
}
