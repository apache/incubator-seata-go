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
	"compress/flate"
	"io"
	"math/rand"
	"testing"
)

func TestDeflateCompressFinalizesStream(t *testing.T) {
	random := make([]byte, 64*1024)
	_, err := rand.New(rand.NewSource(1)).Read(random)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "nil"},
		{name: "empty", data: []byte{}},
		{name: "short", data: []byte("deflate")},
		{name: "repeated", data: bytes.Repeat([]byte("seata undo log "), 10000)},
		{name: "binary", data: random},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compressor := &DeflateCompress{}
			compressed, err := compressor.Compress(tc.data)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("standard-library", func(t *testing.T) {
				reader := flate.NewReader(bytes.NewReader(compressed))
				defer reader.Close()
				decoded, err := io.ReadAll(reader)
				if err != nil {
					t.Fatalf("reading compressed stream: %v", err)
				}
				if !bytes.Equal(tc.data, decoded) {
					t.Fatal("decoded data differs from original")
				}
			})

			t.Run("seata-decompressor", func(t *testing.T) {
				decoded, err := compressor.Decompress(compressed)
				if err != nil {
					t.Fatalf("reading compressed stream: %v", err)
				}
				if !bytes.Equal(tc.data, decoded) {
					t.Fatal("decoded data differs from original")
				}
			})
		})
	}
}
