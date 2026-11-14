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

package getty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/codec"
	"seata.apache.org/seata-go/v2/pkg/util/bytes"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
)

func TestRpcPackageHandler(t *testing.T) {
	msg := message.RpcMessage{
		ID:         1123,
		Type:       message.RequestTypeRequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: byte(1),
		HeadMap: map[string]string{
			"name":    " Jack",
			"age":     "12",
			"address": "Beijing",
		},
		Body: message.GlobalBeginRequest{
			Timeout:         2 * time.Second,
			TransactionName: "SeataGoTransaction",
		},
	}

	codec := RpcPackageHandler{}
	bytes, err := codec.Write(nil, msg)
	assert.Nil(t, err)
	msg2, _, _ := codec.Read(nil, bytes)

	assert.Equal(t, msg, msg2)
}

func TestRpcPackageHandler_EmptyHeadMapValue(t *testing.T) {
	msg := message.RpcMessage{
		ID:         1124,
		Type:       message.RequestTypeRequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: byte(1),
		HeadMap: map[string]string{
			"name": "Jack",
			"note": "",
		},
		Body: message.GlobalBeginRequest{
			Timeout:         2 * time.Second,
			TransactionName: "SeataGoTransaction",
		},
	}

	codec := RpcPackageHandler{}
	bytes, err := codec.Write(nil, msg)
	assert.Nil(t, err)
	msg2, _, err := codec.Read(nil, bytes)
	assert.Nil(t, err)

	assert.Equal(t, msg, msg2)
}

func TestRpcPackageHandler_EmptyHeadMapKey(t *testing.T) {
	msg := message.RpcMessage{
		ID:         1125,
		Type:       message.RequestTypeRequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: byte(1),
		HeadMap: map[string]string{
			"":     "value",
			"name": "Jack",
		},
		Body: message.GlobalBeginRequest{
			Timeout:         2 * time.Second,
			TransactionName: "SeataGoTransaction",
		},
	}

	codec := RpcPackageHandler{}
	bytes, err := codec.Write(nil, msg)
	assert.Nil(t, err)
	msg2, _, err := codec.Read(nil, bytes)
	assert.Nil(t, err)

	assert.Equal(t, msg, msg2)
}

// TestReadMalformedFrames verifies that malformed/hostile frames are rejected
// with errors (or awaited) instead of panicking on slice bounds or spinning
// the getty read loop.
func TestReadMalformedFrames(t *testing.T) {
	h := &RpcPackageHandler{}

	t.Run("short header waits for more data", func(t *testing.T) {
		pkg, consumed, err := h.Read(nil, []byte{0xda, 0xda, 0x01})
		assert.NoError(t, err)
		assert.Nil(t, pkg)
		assert.Equal(t, 0, consumed)
	})

	t.Run("head length below header size is rejected", func(t *testing.T) {
		frame := []byte{
			0xda, 0xda, 0x01,
			0x00, 0x00, 0x00, 0x10, // totalLength = 16
			0x00, 0x00, // headLength = 0 (< 16)
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}
		_, _, err := h.Read(nil, frame)
		assert.Error(t, err)
	})

	t.Run("head length above total length is rejected", func(t *testing.T) {
		frame := []byte{
			0xda, 0xda, 0x01,
			0x00, 0x00, 0x00, 0x10, // totalLength = 16
			0xff, 0xff, // headLength = 65535
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}
		_, _, err := h.Read(nil, frame)
		assert.Error(t, err)
	})

	t.Run("zero total length is rejected", func(t *testing.T) {
		frame := []byte{
			0xda, 0xda, 0x01,
			0x00, 0x00, 0x00, 0x00, // totalLength = 0
			0x00, 0x10, // headLength = 16
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}
		_, _, err := h.Read(nil, frame)
		assert.Error(t, err)
	})

	t.Run("truncated declared length waits for more data", func(t *testing.T) {
		// A valid-looking header whose TotalLength exceeds the buffer must
		// be awaited, not consumed (and must not spin with pkgLen 0).
		frame := []byte{
			0xda, 0xda, 0x01,
			0x00, 0x00, 0x00, 0x40, // totalLength = 64 > len(data)
			0x00, 0x10, // headLength = 16
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}
		pkg, consumed, err := h.Read(nil, frame)
		assert.NoError(t, err)
		assert.Nil(t, pkg)
		assert.Equal(t, 64, consumed)
	})
}

// TestDecodeHeapMapShortRead guards against the underlying buffer's
// short-read semantics: gost Buffer.Read returns (n<len(p), nil), so a
// declared key/value length larger than the remaining bytes must not be
// filled with zero padding and silently become garbage map data.
func TestDecodeHeapMapShortRead(t *testing.T) {
	// key "ABCD" (len 4) + value declared 0x2000 but only 2 bytes remain.
	in := bytes.NewByteBuffer([]byte{
		0x00, 0x04, 'A', 'B', 'C', 'D',
		0x20, 0x00, 'x', 'y',
	})
	m := decodeHeapMap(in, 10)
	if v, ok := m["ABCD"]; ok {
		assert.Equal(t, "xy", v, "value must be exactly the bytes present, not zero-padded")
	}
}

// TestDecodeHeapMapProgressWrap: 4+keyLength+valueLength can exceed uint16;
// accumulated progress must not wrap and stall/corrupt the loop.
func TestDecodeHeapMapProgressWrap(t *testing.T) {
	payload := []byte{
		0xff, 0xfe, // keyLength = 0xFFFE
	}
	// only 2 bytes available: the huge key cannot be satisfied.
	in := bytes.NewByteBuffer(payload)
	m := decodeHeapMap(in, 100)
	assert.Empty(t, m)
}

// TestWriteRejectsOversizedHeadMap: the wire stores HeadLength in a uint16;
// a head map pushing it past 65535 must error instead of truncating silently.
func TestWriteRejectsOversizedHeadMap(t *testing.T) {
	h := &RpcPackageHandler{}
	big := make(map[string]string, 1)
	big[string(make([]byte, 30000))] = string(make([]byte, 40000)) // head map ~70KB
	_, err := h.Write(nil, message.RpcMessage{
		ID:      1,
		Type:    message.RequestTypeHeartbeatRequest,
		Codec:   byte(codec.CodecTypeSeata),
		HeadMap: big,
	})
	assert.Error(t, err)
}
