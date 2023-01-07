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
	"io/ioutil"

	"github.com/dsnet/compress/bzip2"
)

type Bzip2 struct {
}

// Compress Bzip2 compress
func (g *Bzip2) Compress(b []byte) ([]byte, error) {
	var buffer bytes.Buffer
	gz, err := bzip2.NewWriter(&buffer, &bzip2.WriterConfig{Level: bzip2.DefaultCompression})
	if err != nil {
		return nil, err
	}

	if _, err := gz.Write(b); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// Decompress Bzip2 decompress
func (g *Bzip2) Decompress(in []byte) ([]byte, error) {
	reader, err := bzip2.NewReader(bytes.NewReader(in), nil)
	if err != nil {
		return nil, err
	}
	if err = reader.Close(); err != nil {
		return nil, err
	}
	output, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (g *Bzip2) GetCompressorType() CompressorType {
	return CompressorBzip2
}
