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
	"encoding/binary"
	"io"

	gxbytes "github.com/dubbogo/gost/bytes"
)

type ByteBuffer struct {
	buf *gxbytes.Buffer
}

func NewByteBuffer(buf []byte) *ByteBuffer {
	return &ByteBuffer{
		buf: gxbytes.NewBuffer(buf),
	}
}

func (b *ByteBuffer) Bytes() []byte {
	return b.buf.Bytes()
}

func (b *ByteBuffer) Read(p []byte) (n int, err error) {
	return b.buf.Read(p)
}

func (b *ByteBuffer) ReadByte() (byte, error) {
	data := make([]byte, 1)
	n, err := b.Read(data)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, io.ErrShortBuffer
	}
	return data[0], nil
}

func (b *ByteBuffer) ReadInt64() (int64, error) {
	data := make([]byte, 8)
	n, err := b.Read(data)
	if err != nil {
		return 0, err
	}
	if n < 8 {
		return 0, io.ErrShortBuffer
	}
	return Byte2Int64(data), nil
}

func (b *ByteBuffer) ReadUint16() (uint16, error) {
	data := make([]byte, 2)
	n, err := b.Read(data)
	if err != nil {
		return 0, err
	}
	if n < 2 {
		return 0, io.ErrShortBuffer
	}
	return Byte2UInt16(data), nil
}

func (b *ByteBuffer) ReadUint32() (uint32, error) {
	data := make([]byte, 4)
	n, err := b.Read(data)
	if err != nil {
		return 0, err
	}
	if n < 2 {
		return 0, io.ErrShortBuffer
	}
	return Byte2UInt32(data), nil
}

func (b *ByteBuffer) ReadUint64() (uint64, error) {
	data := make([]byte, 8)
	n, err := b.Read(data)
	if err != nil {
		return 0, err
	}
	if n < 2 {
		return 0, io.ErrShortBuffer
	}
	return Byte2UInt64(data), nil
}

func (b *ByteBuffer) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

func (b *ByteBuffer) WriteString(str string) (n int, err error) {
	return b.buf.Write([]byte(str))
}

func (b *ByteBuffer) WriteByte(p byte) error {
	return b.buf.WriteByte(p)
}

func (b *ByteBuffer) WriteUint16(p uint16) (n int, err error) {
	return b.buf.Write(UInt16ToBytes(p))
}

func (b *ByteBuffer) WriteUint32(p uint32) (n int, err error) {
	return b.buf.Write(UInt32ToBytes(p))
}

func (b *ByteBuffer) WriteUint64(p uint64) (n int, err error) {
	return b.buf.Write(UInt64ToBytes(p))
}

func (b *ByteBuffer) WriteInt64(p int64) (n int, err error) {
	return b.buf.Write(Int64ToBytes(p))
}

// Byte2Int64 byte array to int64 value using big order
func Byte2Int64(data []byte) int64 {
	return int64((int64(data[0])&0xff)<<56 |
		(int64(data[1])&0xff)<<48 |
		(int64(data[2])&0xff)<<40 |
		(int64(data[3])&0xff)<<32 |
		(int64(data[4])&0xff)<<24 |
		(int64(data[5])&0xff)<<16 |
		(int64(data[6])&0xff)<<8 |
		(int64(data[7]) & 0xff))
}

// Byte2UInt64 byte array to int64 value using big order
func Byte2UInt64(data []byte) uint64 {
	return binary.BigEndian.Uint64(data)
}

// Byte2UInt16 byte array to uint16 value using big order
func Byte2UInt16(data []byte) uint16 {
	return binary.BigEndian.Uint16(data)
}

// Byte2UInt32 byte array to uint32 value using big order
func Byte2UInt32(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

// Int2BytesTo int value to bytes array using big order
func Int2BytesTo(v int, ret []byte) {
	ret[0] = byte(v >> 24)
	ret[1] = byte(v >> 16)
	ret[2] = byte(v >> 8)
	ret[3] = byte(v)
}

// Int2Bytes int value to bytes array using big order
func Int2Bytes(v int) []byte {
	ret := make([]byte, 4)
	Int2BytesTo(v, ret)
	return ret
}

// Int64ToBytesTo int64 value to bytes array using big order
func Int64ToBytesTo(v int64, ret []byte) {
	ret[0] = byte(v >> 56)
	ret[1] = byte(v >> 48)
	ret[2] = byte(v >> 40)
	ret[3] = byte(v >> 32)
	ret[4] = byte(v >> 24)
	ret[5] = byte(v >> 16)
	ret[6] = byte(v >> 8)
	ret[7] = byte(v)
}

// Uint64ToBytesTo uint64 value to bytes array using big order
func Uint64ToBytesTo(v uint64, ret []byte) {
	binary.BigEndian.PutUint64(ret, v)
}

// Int64ToBytes int64 value to bytes array using big order
func Int64ToBytes(v int64) []byte {
	ret := make([]byte, 8)
	Int64ToBytesTo(v, ret)
	return ret
}

// Uint32ToBytesTo uint32 value to bytes array using big order
func Uint32ToBytesTo(v uint32, ret []byte) {
	binary.BigEndian.PutUint32(ret, v)
}

// UInt32ToBytes uint32 value to bytes array using big order
func UInt32ToBytes(v uint32) []byte {
	ret := make([]byte, 4)
	Uint32ToBytesTo(v, ret)
	return ret
}

// Uint16ToBytesTo uint16 value to bytes array using big order
func Uint16ToBytesTo(v uint16, ret []byte) {
	binary.BigEndian.PutUint16(ret, v)
}

// UInt16ToBytes uint16 value to bytes array using big order
func UInt16ToBytes(v uint16) []byte {
	ret := make([]byte, 2)
	Uint16ToBytesTo(v, ret)
	return ret
}

// UInt16ToBytes uint16 value to bytes array using big order
func UInt64ToBytes(v uint64) []byte {
	ret := make([]byte, 8)
	Uint64ToBytesTo(v, ret)
	return ret
}
