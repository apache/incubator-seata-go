package codec

import (
	"github.com/fagongzi/goetty"
)

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

type AbstractIdentifyResponseCodec struct {
}

func (c *AbstractIdentifyResponseCodec) Encode(in interface{}) []byte {
	buf := goetty.NewByteBuf(0)
	resp := in.(message.AbstractIdentifyResponse)

	if resp.Identified {
		buf.WriteByte(byte(1))
	} else {
		buf.WriteByte(byte(0))
	}

	Write16String(resp.Version, buf)
	return buf.RawBuf()
}

func (c *AbstractIdentifyResponseCodec) Decode(in []byte) interface{} {
	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)
	msg := message.AbstractIdentifyResponse{}

	identified, _ := buf.ReadByte()
	if identified == byte(1) {
		msg.Identified = true
	} else if identified == byte(0) {
		msg.Identified = false
	}

	length := ReadUInt16(buf)
	if length > 0 {
		versionBytes := make([]byte, length)
		msg.Version = string(Read(buf, versionBytes))
	}

	return msg
}
