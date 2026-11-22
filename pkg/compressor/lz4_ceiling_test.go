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
