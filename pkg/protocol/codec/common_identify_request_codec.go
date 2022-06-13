package codec

import (
	"github.com/fagongzi/goetty"
)

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

type AbstractIdentifyRequestCodec struct {
}

func (c *AbstractIdentifyRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.AbstractIdentifyRequest)
	buf := goetty.NewByteBuf(0)

	Write16String(req.Version, buf)
	Write16String(req.ApplicationId, buf)
	Write16String(req.TransactionServiceGroup, buf)
	Write16String(string(req.ExtraData), buf)

	return buf.RawBuf()
}

func (c *AbstractIdentifyRequestCodec) Decode(in []byte) interface{} {
	msg := message.AbstractIdentifyRequest{}
	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)
	var len uint16

	if buf.Readable() < 2 {
		return msg
	}
	len = ReadUInt16(buf)
	if uint16(buf.Readable()) < len {
		return msg
	}
	versionBytes := make([]byte, len)
	msg.Version = string(Read(buf, versionBytes))

	if buf.Readable() < 2 {
		return msg
	}
	len = ReadUInt16(buf)
	if uint16(buf.Readable()) < len {
		return msg
	}
	applicationIdBytes := make([]byte, len)
	msg.ApplicationId = string(Read(buf, applicationIdBytes))

	if buf.Readable() < 2 {
		return msg
	}
	len = ReadUInt16(buf)
	if uint16(buf.Readable()) < len {
		return msg
	}
	transactionServiceGroupBytes := make([]byte, len)
	msg.TransactionServiceGroup = string(Read(buf, transactionServiceGroupBytes))

	if buf.Readable() < 2 {
		return msg
	}
	len = ReadUInt16(buf)
	if len > 0 && uint16(buf.Readable()) > len {
		extraDataBytes := make([]byte, len)
		msg.ExtraData = Read(buf, extraDataBytes)
	}

	return msg
}
