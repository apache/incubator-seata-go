package codec

import (
	"github.com/fagongzi/goetty"

	"github.com/fagongzi/util/hack"
)

// Write16String write string value with 16 byte length
func Write16String(value string, buf *goetty.ByteBuf) {
	if value != "" {
		buf.WriteUInt16(uint16(len(value)))
		buf.WriteString(value)
	} else {
		buf.WriteUInt16(uint16(0))
	}
}

// Write16String write string value with 16 byte length
func Write32String(value string, buf *goetty.ByteBuf) {
	if value != "" {
		buf.WriteUInt32(uint32(len(value)))
		buf.WriteString(value)
	} else {
		buf.WriteUInt32(uint32(0))
	}
}

// Write8String write string value with 8 byte length
func Write8String(value string, buf *goetty.ByteBuf) {
	if value != "" {
		buf.WriteByte(uint8(len(value)))
		buf.WriteString(value)
	} else {
		buf.WriteByte(uint8(0))
	}
}

// ReadString read string value
func ReadString(buf *goetty.ByteBuf) string {
	size := ReadUInt16(buf)
	if size == 0 {
		return ""
	}

	_, value, _ := buf.ReadBytes(int(size))
	return hack.SliceToString(value)
}

// MaybeReadString maybe read string value
func MaybeReadString(buf *goetty.ByteBuf) (string, bool) {
	if buf.Readable() < 2 {
		return "", false
	}

	size := ReadUInt16(buf)
	if size == 0 {
		return "", true
	}

	if buf.Readable() < int(size) {
		return "", false
	}

	_, value, _ := buf.ReadBytes(int(size))
	return hack.SliceToString(value), true
}

// WriteBigString write big string
func WriteBigString(value string, buf *goetty.ByteBuf) {
	if value != "" {
		buf.WriteInt(len(value))
		buf.WriteString(value)
	} else {
		buf.WriteInt(0)
	}
}

// ReadBigString read big string
func ReadBigString(buf *goetty.ByteBuf) string {
	size := ReadInt(buf)
	if size == 0 {
		return ""
	}

	_, value, _ := buf.ReadBytes(size)
	return hack.SliceToString(value)
}

// MaybeReadBigString maybe read string value
func MaybeReadBigString(buf *goetty.ByteBuf) (string, bool) {
	if buf.Readable() < 4 {
		return "", false
	}

	size := ReadInt(buf)
	if size == 0 {
		return "", true
	}

	if buf.Readable() < size {
		return "", false
	}

	_, value, _ := buf.ReadBytes(int(size))
	return hack.SliceToString(value), true
}

// ReadUInt64 read uint64 value
func ReadUInt64(buf *goetty.ByteBuf) uint64 {
	value, _ := buf.ReadUInt64()
	return value
}

// ReadUInt16 read uint16 value
func ReadUInt16(buf *goetty.ByteBuf) uint16 {
	value, _ := buf.ReadUInt16()
	return value
}

// ReadUInt32 read uint16 value
func ReadUInt32(buf *goetty.ByteBuf) uint32 {
	value, _ := buf.ReadUInt32()
	return value
}

// ReadUInt32 read uint16 value
func Read(buf *goetty.ByteBuf, p []byte) []byte {
	buf.Read(p)
	return p
}

// ReadInt read int value
func ReadInt(buf *goetty.ByteBuf) int {
	value, _ := buf.ReadInt()
	return value
}

// ReadByte read byte value
func ReadByte(buf *goetty.ByteBuf) byte {
	value, _ := buf.ReadByte()
	return value
}

// ReadBytes read bytes value
func ReadBytes(n int, buf *goetty.ByteBuf) []byte {
	_, value, _ := buf.ReadBytes(n)
	return value
}

// WriteBool write bool value
func WriteBool(value bool, out *goetty.ByteBuf) {
	out.WriteByte(boolToByte(value))
}

// WriteSlice write slice value
func WriteSlice(value []byte, buf *goetty.ByteBuf) {
	buf.WriteUInt16(uint16(len(value)))
	if len(value) > 0 {
		buf.Write(value)
	}
}

// ReadSlice read slice value
func ReadSlice(buf *goetty.ByteBuf) []byte {
	l, _ := buf.ReadUInt16()
	if l == 0 {
		return nil
	}

	_, data, _ := buf.ReadBytes(int(l))
	return data
}

func boolToByte(value bool) byte {
	if value {
		return 1
	}

	return 0
}

func byteToBool(value byte) bool {
	if value == 1 {
		return true
	}

	return false
}
