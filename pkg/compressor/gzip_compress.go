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
	"compress/gzip"
	"io/ioutil"
)

type Gzip struct {
}

// Compress gzip compress
func (g *Gzip) Compress(b []byte) ([]byte, error) {
	var buffer bytes.Buffer

	gz := gzip.NewWriter(&buffer)

	if _, err := gz.Write(b); err != nil {
		return nil, err
	}

	if err := gz.Flush(); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// Decompress gzip decompress
func (g *Gzip) Decompress(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}

	if err = reader.Close(); err != nil {
		return nil, err
	}

	return ioutil.ReadAll(reader)
}

func (g *Gzip) GetCompressorType() CompressorType {
	return CompressorGzip
}
