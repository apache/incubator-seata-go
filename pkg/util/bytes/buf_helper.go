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

func ReadBytes(n int, buf *ByteBuffer) []byte {
	bytes := make([]byte, n)
	_, err := buf.Read(bytes)
	if err != nil {
		return nil
	}
	return bytes
}

func ReadByte(buf *ByteBuffer) byte {
	value, _ := buf.ReadByte()
	return value
}

func ReadUint8(buf *ByteBuffer) uint8 {
	value, _ := buf.ReadByte()
	return value
}

func ReadUInt16(buf *ByteBuffer) uint16 {
	value, _ := buf.ReadUint16()
	return value
}

func ReadUInt32(buf *ByteBuffer) uint32 {
	value, _ := buf.ReadUint32()
	return value
}

func ReadUInt64(buf *ByteBuffer) uint64 {
	value, _ := buf.ReadUint64()
	return value
}

func ReadString8(buf *ByteBuffer) string {
	bytes := make([]byte, 1)
	_, err := buf.Read(bytes)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func Read1String16(buf *ByteBuffer) string {
	bytes := make([]byte, 2)
	_, err := buf.Read(bytes)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func ReadString32(buf *ByteBuffer) string {
	bytes := make([]byte, 4)
	_, err := buf.Read(bytes)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func ReadString64(buf *ByteBuffer) string {
	bytes := make([]byte, 8)
	_, err := buf.Read(bytes)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func ReadString8Length(buf *ByteBuffer) string {
	length, _ := buf.ReadByte()
	if length > 0 {
		p := make([]byte, length)
		_, err := buf.Read(p)
		if err != nil {
			return ""
		}
		return string(p)
	}
	return ""
}

func ReadString16Length(buf *ByteBuffer) string {
	length, _ := buf.ReadUint16()
	if length > 0 {
		p := make([]byte, length)
		_, err := buf.Read(p)
		if err != nil {
			return ""
		}
		return string(p)
	}
	return ""
}

func ReadString32Length(buf *ByteBuffer) string {
	length, _ := buf.ReadUint32()
	if length > 0 {
		p := make([]byte, length)
		_, err := buf.Read(p)
		if err != nil {
			return ""
		}
		return string(p)
	}
	return ""
}

func ReadString64Length(buf *ByteBuffer) string {
	length, _ := buf.ReadUint64()
	if length > 0 {
		p := make([]byte, length)
		_, err := buf.Read(p)
		if err != nil {
			return ""
		}
		return string(p)
	}
	return ""
}

func WriteString8Length(value string, buf *ByteBuffer) {
	if value != "" {
		err := buf.WriteByte(byte(len(value)))
		if err != nil {
			return
		}
		_, err = buf.WriteString(value)
		if err != nil {
			return
		}
	} else {
		err := buf.WriteByte(byte(0))
		if err != nil {
			return
		}
	}
}

func WriteString16Length(value string, buf *ByteBuffer) {
	if value != "" {
		_, err := buf.WriteUint16(uint16(len(value)))
		if err != nil {
			return
		}
		_, err = buf.WriteString(value)
		if err != nil {
			return
		}
	} else {
		_, err := buf.WriteUint16(uint16(0))
		if err != nil {
			return
		}
	}
}

func WriteString32Length(value string, buf *ByteBuffer) {
	if value != "" {
		_, err := buf.WriteUint32(uint32(len(value)))
		if err != nil {
			return
		}
		_, err = buf.WriteString(value)
		if err != nil {
			return
		}
	} else {
		_, err := buf.WriteUint32(uint32(0))
		if err != nil {
			return
		}
	}
}

func WriteString64Length(value string, buf *ByteBuffer) {
	if value != "" {
		_, err := buf.WriteUint64(uint64(len(value)))
		if err != nil {
			return
		}
		_, err = buf.WriteString(value)
		if err != nil {
			return
		}
	} else {
		_, err := buf.WriteUint64(uint64(0))
		if err != nil {
			return
		}
	}
}
