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

func TestGetCompressor(t *testing.T) {
	tests := []struct {
		name           string
		compressorType CompressorType
		wantType       CompressorType
	}{
		{
			name:           "None compressor",
			compressorType: CompressorNone,
			wantType:       CompressorNone,
		},
		{
			name:           "Gzip compressor",
			compressorType: CompressorGzip,
			wantType:       CompressorGzip,
		},
		{
			name:           "Zip compressor",
			compressorType: CompressorZip,
			wantType:       CompressorZip,
		},
		{
			name:           "Sevenz compressor",
			compressorType: CompressorSevenz,
			wantType:       CompressorSevenz,
		},
		{
			name:           "Bzip2 compressor",
			compressorType: CompressorBzip2,
			wantType:       CompressorBzip2,
		},
		{
			name:           "Lz4 compressor",
			compressorType: CompressorLz4,
			wantType:       CompressorLz4,
		},
		{
			name:           "Deflate compressor",
			compressorType: CompressorDeflate,
			wantType:       CompressorDeflate,
		},
		{
			name:           "Zstd compressor",
			compressorType: CompressorZstd,
			wantType:       CompressorZstd,
		},
		{
			name:           "Snappy compressor",
			compressorType: CompressorSnappy,
			wantType:       CompressorSnappy,
		},
		{
			name:           "Unknown compressor falls back to None",
			compressorType: CompressorType("Unknown"),
			wantType:       CompressorNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor := tt.compressorType.GetCompressor()
			assert.NotNil(t, compressor)
			assert.EqualValues(t, tt.wantType, compressor.GetCompressorType())
		})
	}
}
