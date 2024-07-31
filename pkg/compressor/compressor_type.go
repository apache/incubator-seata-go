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

type CompressorType string

const (
	// "None" means no compressor is used
	CompressorNone    CompressorType = "None"
	CompressorGzip    CompressorType = "Gzip"
	CompressorZip     CompressorType = "Zip"
	CompressorSevenz  CompressorType = "Sevenz"
	CompressorBzip2   CompressorType = "Bzip2"
	CompressorLz4     CompressorType = "Lz4"
	CompressorDeflate CompressorType = "Deflate"
	CompressorZstd    CompressorType = "Zstd"
)

func (c CompressorType) GetCompressor() Compressor {
	switch c {
	case CompressorNone:
		return &NoneCompressor{}
	case CompressorGzip:
		return &Gzip{}
	case CompressorZip:
		return &Zip{}
	case CompressorBzip2:
		return &Bzip2{}
	case CompressorLz4:
		return &Lz4{}
	case CompressorZstd:
		return &Zstd{}
	case CompressorDeflate:
		return &DeflateCompress{}
	default:
		return &NoneCompressor{}
	}
}
