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
	"io"

	"github.com/ulikunitz/xz/lzma"
)

type Sevenz struct{}

// Compress Sevenz compress using LZMA algorithm
func (s *Sevenz) Compress(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer, err := lzma.NewWriter(&buffer)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// Decompress Sevenz decompress using LZMA algorithm
func (s *Sevenz) Decompress(data []byte) ([]byte, error) {
	reader, err := lzma.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}

func (s *Sevenz) GetCompressorType() CompressorType {
	return CompressorSevenz
}
