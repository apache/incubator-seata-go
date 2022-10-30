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

func TestReadBytes(t *testing.T) {
	bytes := []byte{2, 3, 4, 5, 6}
	byteBuffer := NewByteBuffer(bytes)
	readBytes := ReadBytes(len(bytes), byteBuffer)
	assert.Equal(t, bytes, readBytes)
}

func TestReadUint8(t *testing.T) {
	bytes := []byte{255}
	byteBuffer := NewByteBuffer(bytes)
	readUint8 := ReadUint8(byteBuffer)
	assert.Equal(t, readUint8, bytes[0])
}

func TestReadUInt16(t *testing.T) {
	bytes := []byte{255, 255}
	byteBuffer := NewByteBuffer(bytes)
	readUint16 := ReadUInt16(byteBuffer)
	assert.Equal(t, readUint16, uint16(math.MaxUint16))
}

func TestReadUInt32(t *testing.T) {
	bytes := []byte{255, 255, 255, 255}
	byteBuffer := NewByteBuffer(bytes)
	readUInt32 := ReadUInt32(byteBuffer)
	assert.Equal(t, readUInt32, uint32(math.MaxUint32))
}

func TestReadUInt64(t *testing.T) {
	bytes := []byte{255, 255, 255, 255, 255, 255, 255, 255}
	byteBuffer := NewByteBuffer(bytes)
	readUInt64 := ReadUInt64(byteBuffer)
	assert.Equal(t, readUInt64, uint64(math.MaxUint64))
}

func TestReadString8(t *testing.T) {
	bytes := []byte("A")
	byteBuffer := NewByteBuffer(bytes)
	readString8 := ReadString8(byteBuffer)
	assert.Equal(t, readString8, "A")
}

func TestRead1String16(t *testing.T) {
	bytes := []byte("seata_test")
	byteBuffer := NewByteBuffer(bytes)
	readString16 := Read1String16(byteBuffer)
	assert.Equal(t, readString16, "se")
}

func TestReadString32(t *testing.T) {
	bytes := []byte("seata_test")
	byteBuffer := NewByteBuffer(bytes)
	readString32 := ReadString32(byteBuffer)
	assert.Equal(t, readString32, "seat")
}

func TestReadString64(t *testing.T) {
	bytes := []byte("seata_test")
	byteBuffer := NewByteBuffer(bytes)
	readString64 := ReadString64(byteBuffer)
	assert.Equal(t, readString64, "seata_te")
}

func generateStringByLength(length int) string {
	// a-z  loop generate
	// a == 97
	// z == 122
	letterCount := 26
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		bytes[i] = byte(97 + (i % letterCount))
	}
	return string(bytes)
}

func ReadStringNLengthTest(
	t *testing.T,
	length int,
	writeLengthFun func(length int, byteBuffer ByteBuffer),
	readStringFun func(buf *ByteBuffer) string,
) {
	str := generateStringByLength(length)
	byteBuffer := NewByteBuffer([]byte{})

	// write length and content
	writeLengthFun(len(str), *byteBuffer)
	_, _ = byteBuffer.WriteString(str)

	readString := readStringFun(byteBuffer)
	assert.Equal(t, readString, generateStringByLength(length))
}

func TestReadString8Length(t *testing.T) {
	writeLengthFun := func(length int, byteBuffer ByteBuffer) {
		_ = byteBuffer.WriteByte(byte(length))
	}

	ReadStringNLengthTest(t, 0, writeLengthFun, ReadString8Length)
	ReadStringNLengthTest(t, 255, writeLengthFun, ReadString8Length)
}

func TestReadString16Length(t *testing.T) {
	writeFun := func(length int, byteBuffer ByteBuffer) {
		_, _ = byteBuffer.WriteUint16(uint16(length))
	}

	ReadStringNLengthTest(t, 0, writeFun, ReadString16Length)
	ReadStringNLengthTest(t, math.MaxInt16, writeFun, ReadString16Length)
}

func TestReadString32Length(t *testing.T) {
	writeFun := func(length int, byteBuffer ByteBuffer) {
		_, _ = byteBuffer.WriteUint32(uint32(length))
	}

	ReadStringNLengthTest(t, 0, writeFun, ReadString32Length)
	ReadStringNLengthTest(t, math.MaxInt8, writeFun, ReadString32Length)
}

func TestReadString64Length(t *testing.T) {
	writeFun := func(length int, byteBuffer ByteBuffer) {
		_, _ = byteBuffer.WriteUint64(uint64(length))
	}

	ReadStringNLengthTest(t, 0, writeFun, ReadString64Length)
	ReadStringNLengthTest(t, math.MaxInt8, writeFun, ReadString64Length)
}

func WriteStringNLengthTest(
	t *testing.T,
	length int,
	writeStringFun func(value string, buf *ByteBuffer),
	readStringFun func(buf *ByteBuffer) string,
) {
	str := generateStringByLength(length)
	byteBuffer := NewByteBuffer([]byte{})

	// write length and content
	writeStringFun(str, byteBuffer)

	readString := readStringFun(byteBuffer)
	assert.Equal(t, readString, generateStringByLength(length))
}

func TestWriteString8Length(t *testing.T) {
	WriteStringNLengthTest(t, 0, WriteString8Length, ReadString8Length)
	WriteStringNLengthTest(t, math.MaxInt8, WriteString8Length, ReadString8Length)
}

func TestWriteString16Length(t *testing.T) {
	WriteStringNLengthTest(t, 0, WriteString16Length, ReadString16Length)
	WriteStringNLengthTest(t, math.MaxInt8, WriteString16Length, ReadString16Length)
}

func TestWriteString32Length(t *testing.T) {
	WriteStringNLengthTest(t, 0, WriteString32Length, ReadString32Length)
	WriteStringNLengthTest(t, math.MaxInt8, WriteString32Length, ReadString32Length)
}

func TestWriteString64Length(t *testing.T) {
	WriteStringNLengthTest(t, 0, WriteString64Length, ReadString64Length)
	WriteStringNLengthTest(t, math.MaxInt8, WriteString64Length, ReadString64Length)
}
