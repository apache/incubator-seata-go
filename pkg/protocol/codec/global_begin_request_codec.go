package codec

import (
	"github.com/fagongzi/goetty"
)

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalBeginRequestCodec{})
}

type GlobalBeginRequestCodec struct {
}

func (c *GlobalBeginRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.GlobalBeginRequest)
	buf := goetty.NewByteBuf(0)

	buf.WriteUInt32(uint32(req.Timeout))
	Write16String(req.TransactionName, buf)

	return buf.RawBuf()
}

func (g *GlobalBeginRequestCodec) Decode(in []byte) interface{} {
	msg := message.GlobalBeginRequest{}
	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)

	msg.Timeout = int32(ReadUInt32(buf))
	len := ReadUInt16(buf)
	if len > 0 {
		transactionName := make([]byte, len)
		msg.TransactionName = string(Read(buf, transactionName))
	}

	return msg
}

func (g *GlobalBeginRequestCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalBegin
}
