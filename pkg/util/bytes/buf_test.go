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

package bytes

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytes(t *testing.T) {
	bytes := []byte{2, 3, 4, 5, 6}
	byteBuffer := NewByteBuffer(bytes)
	assert.Equal(t, byteBuffer.Bytes(), bytes)
}

func TestRead(t *testing.T) {
	bytes := []byte{2, 3, 4, 5, 6}
	byteBuffer := NewByteBuffer(bytes)
	newBytes := make([]byte, len(bytes))
	_, _ = byteBuffer.Read(newBytes)
	assert.Equal(t, bytes, newBytes)
}

func TestReadByte(t *testing.T) {
	bytes := []byte{2, 3, 4, 5, 6}
	byteBuffer := NewByteBuffer(bytes)
	for _, byteItem := range bytes {
		readByte, _ := byteBuffer.ReadByte()
		assert.Equal(t, byteItem, readByte)
	}
}

func TestReadInt64(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{127, 255, 255, 255, 255, 255, 255, 255})
	readInt64, _ := byteBuffer.ReadInt64()
	assert.Equal(t, readInt64, int64(math.MaxInt64))
}

func TestReadUint16(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{255, 255})
	readInt16, _ := byteBuffer.ReadUint16()
	assert.Equal(t, readInt16, uint16(math.MaxUint16))
}

func TestReadUint32(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{255, 255, 255, 255})
	readInt32, _ := byteBuffer.ReadUint32()
	assert.Equal(t, readInt32, uint32(math.MaxUint32))
}

func TestReadUint64(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{255, 255, 255, 255, 255, 255, 255, 255})
	readUint64, _ := byteBuffer.ReadUint64()
	assert.Equal(t, readUint64, uint64(math.MaxUint64))
}

func TestWrite(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{})
	_, err := byteBuffer.Write([]byte{255, 255})
	if err != nil {
		t.Error()
	}
	readUint16, _ := byteBuffer.ReadUint16()
	assert.Equal(t, readUint16, uint16(math.MaxUint16))
}

func TestWriteString(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{})
	_, err := byteBuffer.WriteString("seata")
	if err != nil {
		t.Error()
		return
	}
	seataBytes := []byte{115, 101, 97, 116, 97}
	for _, byteItem := range seataBytes {
		readByte, _ := byteBuffer.ReadByte()
		assert.Equal(t, byteItem, readByte)
	}
}

func TestWriteByte(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{})
	_ = byteBuffer.WriteByte(1)
	readByte, _ := byteBuffer.ReadByte()
	assert.Equal(t, readByte, byte(1))
}

func TestWriteUint16(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{})
	_, _ = byteBuffer.WriteUint16(uint16(math.MaxUint16))
	readUint16, _ := byteBuffer.ReadUint16()
	assert.Equal(t, uint16(math.MaxUint16), readUint16)
}

func TestWriteUint32(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{})
	_, _ = byteBuffer.WriteUint32(uint32(math.MaxUint32))
	readUint32, _ := byteBuffer.ReadUint32()
	assert.Equal(t, uint32(math.MaxUint32), readUint32)
}

func TestWriteUint64(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{})
	_, _ = byteBuffer.WriteUint64(uint64(math.MaxUint64))
	readUint64, _ := byteBuffer.ReadUint64()
	assert.Equal(t, uint64(math.MaxUint64), readUint64)
}

func TestWriteInt64(t *testing.T) {
	byteBuffer := NewByteBuffer([]byte{})
	_, _ = byteBuffer.WriteInt64(int64(math.MaxInt64))
	readInt64, _ := byteBuffer.ReadInt64()
	assert.Equal(t, int64(math.MaxInt64), readInt64)
}

func TestByte2Int64(t *testing.T) {
	byte2Int64 := Byte2Int64([]byte{127, 255, 255, 255, 255, 255, 255, 255})
	assert.Equal(t, byte2Int64, int64(math.MaxInt64))
}

func TestByte2UInt16(t *testing.T) {
	byte2UInt16 := Byte2UInt16([]byte{255, 255})
	assert.Equal(t, byte2UInt16, uint16(math.MaxUint16))
}

func TestByte2UInt32(t *testing.T) {
	byte2UInt32 := Byte2UInt32([]byte{255, 255, 255, 255})
	assert.Equal(t, byte2UInt32, uint32(math.MaxUint32))
}

func TestInt2BytesTo(t *testing.T) {
	bytes := make([]byte, 4)
	Int2BytesTo(math.MaxInt64, bytes)
	assert.Equal(t, bytes, []byte{255, 255, 255, 255})
}

func TestInt2Bytes(t *testing.T) {
	bytes := Int2Bytes(math.MaxInt64)
	assert.Equal(t, bytes, []byte{255, 255, 255, 255})
}

func TestInt64ToBytesTo(t *testing.T) {
	bytes := make([]byte, 8)
	Int64ToBytesTo(math.MaxInt64, bytes)
	assert.Equal(t, bytes, []byte{127, 255, 255, 255, 255, 255, 255, 255})
}

func TestUint64ToBytesTo(t *testing.T) {
	bytes := make([]byte, 8)
	Uint64ToBytesTo(math.MaxUint64, bytes)
	assert.Equal(t, bytes, []byte{255, 255, 255, 255, 255, 255, 255, 255})
}

func TestInt64ToBytes(t *testing.T) {
	bytes := Int64ToBytes(math.MaxInt64)
	assert.Equal(t, bytes, []byte{127, 255, 255, 255, 255, 255, 255, 255})
}

func TestUint32ToBytesTo(t *testing.T) {
	bytes := make([]byte, 4)
	Uint32ToBytesTo(math.MaxUint32, bytes)
	assert.Equal(t, bytes, []byte{255, 255, 255, 255})
}

func TestUInt32ToBytes(t *testing.T) {
	bytes := UInt32ToBytes(math.MaxUint32)
	assert.Equal(t, bytes, []byte{255, 255, 255, 255})
}

func TestUint16ToBytesTo(t *testing.T) {
	bytes := make([]byte, 2)
	Uint16ToBytesTo(math.MaxUint16, bytes)
	assert.Equal(t, bytes, []byte{255, 255})
}

func TestUInt16ToBytes(t *testing.T) {
	bytes := UInt16ToBytes(math.MaxUint16)
	assert.Equal(t, bytes, []byte{255, 255})
}

func TestUInt64ToBytes(t *testing.T) {
	bytes := UInt64ToBytes(math.MaxUint64)
	assert.Equal(t, bytes, []byte{255, 255, 255, 255, 255, 255, 255, 255})
}
