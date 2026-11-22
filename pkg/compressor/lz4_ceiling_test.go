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

// -*- coding: utf-8 -*-
// Regression tests for the seata-go #1215 ceiling discussion (XiaoFeiASK P1).
package compressor

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// A compressible block whose decoded size exceeds 64 MiB must still round
// trip: undo_log.rollback_info is a longblob and carries no 64 MiB constraint
// of its own (reported by XiaoFeiASK). One KiB of random bytes repeated ten
// times forms a 10 KiB pattern; repeating that pattern to 67 MiB compresses
// with self-overlapping matches and decodes past the 64 MiB mark that the
// previous fixed ceiling rejected.
func TestLz4DecompressLargeBlockRoundTrip(t *testing.T) {
	unit := make([]byte, 1024)
	if _, err := rand.Read(unit); err != nil {
		t.Fatalf("rand: %v", err)
	}
	pattern := bytes.Repeat(unit, 10)
	var data []byte
	for len(data) <= 67108865 {
		data = append(data, pattern...)
	}
	l := &Lz4{}
	compressed, err := l.Compress(data)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	if len(compressed) >= len(data) {
		t.Fatalf("expected the sample to compress: %d -> %d", len(data), len(compressed))
	}
	decoded, err := l.Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress of a %d-byte block (%d-byte input) failed: %v", len(data), len(compressed), err)
	}
	if !bytes.Equal(decoded, data) {
		t.Fatalf("round trip mismatch: got %d bytes", len(decoded))
	}
}
