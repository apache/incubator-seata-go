package codec

import (
	"github.com/fagongzi/goetty"
)

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &RegisterRMRequestCodec{})
}

type RegisterRMRequestCodec struct {
}

func (g *RegisterRMRequestCodec) Decode(in []byte) interface{} {
	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)
	msg := message.RegisterRMRequest{}

	length := ReadUInt16(buf)
	if length > 0 {
		bytes := make([]byte, length)
		msg.Version = string(Read(buf, bytes))
	}

	length = ReadUInt16(buf)
	if length > 0 {
		bytes := make([]byte, length)
		msg.ApplicationId = string(Read(buf, bytes))
	}

	length = ReadUInt16(buf)
	if length > 0 {
		bytes := make([]byte, length)
		msg.TransactionServiceGroup = string(Read(buf, bytes))
	}

	length = ReadUInt16(buf)
	if length > 0 {
		bytes := make([]byte, length)
		msg.ExtraData = Read(buf, bytes)
	}

	length32 := ReadUInt32(buf)
	if length32 > 0 {
		bytes := make([]byte, length32)
		msg.ResourceIds = string(Read(buf, bytes))
	}

	return msg
}

func (c *RegisterRMRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.RegisterRMRequest)
	buf := goetty.NewByteBuf(0)

	Write16String(req.Version, buf)
	Write16String(req.ApplicationId, buf)
	Write16String(req.TransactionServiceGroup, buf)
	Write16String(string(req.ExtraData), buf)
	Write16String(req.ResourceIds, buf)

	return buf.RawBuf()
}

func (g *RegisterRMRequestCodec) GetMessageType() message.MessageType {
	return message.MessageType_RegRm
}
