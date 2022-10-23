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
	out := make([]byte, 100*len(in))
	n, err := lz4.UncompressBlock(in, out)
	if err != nil {
		return nil, err
	}
	return out[:n], nil
}

func (l *Lz4) GetCompressorType() CompressorType {
	return CompressorLz4
}
