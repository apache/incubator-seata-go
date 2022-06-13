package codec

import (
	"github.com/fagongzi/goetty"
)

import (
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/transaction"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalBeginResponseCodec{})
}

type GlobalBeginResponseCodec struct {
}

func (c *GlobalBeginResponseCodec) Encode(in interface{}) []byte {
	buf := goetty.NewByteBuf(0)
	resp := in.(message.GlobalBeginResponse)

	buf.WriteByte(byte(resp.ResultCode))
	if resp.ResultCode == message.ResultCodeFailed {
		var msg string
		if len(resp.Msg) > 128 {
			msg = resp.Msg[:128]
		} else {
			msg = resp.Msg
		}
		Write16String(msg, buf)
	}
	buf.WriteByte(byte(resp.TransactionExceptionCode))
	Write16String(resp.Xid, buf)
	Write16String(string(resp.ExtraData), buf)

	return buf.RawBuf()
}

func (g *GlobalBeginResponseCodec) Decode(in []byte) interface{} {
	var lenth uint16
	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)
	msg := message.GlobalBeginResponse{}

	resultCode := ReadByte(buf)
	msg.ResultCode = message.ResultCode(resultCode)
	if msg.ResultCode == message.ResultCodeFailed {
		lenth = ReadUInt16(buf)
		if lenth > 0 {
			bytes := make([]byte, lenth)
			msg.Msg = string(Read(buf, bytes))
		}
	}

	exceptionCode := ReadByte(buf)
	msg.TransactionExceptionCode = transaction.TransactionExceptionCode(exceptionCode)

	lenth = ReadUInt16(buf)
	if lenth > 0 {
		bytes := make([]byte, lenth)
		msg.Xid = string(Read(buf, bytes))
	}

	lenth = ReadUInt16(buf)
	if lenth > 0 {
		bytes := make([]byte, lenth)
		msg.ExtraData = Read(buf, bytes)
	}

	return msg
}

func (g *GlobalBeginResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalBeginResult
}
