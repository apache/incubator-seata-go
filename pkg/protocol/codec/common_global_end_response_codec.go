package codec

import (
	"github.com/fagongzi/goetty"
)

import (
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/transaction"
)

type CommonGlobalEndResponseCodec struct {
}

func (c *CommonGlobalEndResponseCodec) Encode(in interface{}) []byte {
	buf := goetty.NewByteBuf(0)
	resp := in.(message.AbstractGlobalEndResponse)

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
	buf.WriteByte(byte(resp.GlobalStatus))

	return buf.RawBuf()
}

func (c *CommonGlobalEndResponseCodec) Decode(in []byte) interface{} {
	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)

	msg := message.AbstractGlobalEndResponse{}

	resultCode := ReadByte(buf)
	msg.ResultCode = message.ResultCode(resultCode)
	if msg.ResultCode == message.ResultCodeFailed {
		length := ReadUInt16(buf)
		if length > 0 {
			bytes := make([]byte, length)
			msg.Msg = string(Read(buf, bytes))
		}
	}

	exceptionCode := ReadByte(buf)
	msg.TransactionExceptionCode = transaction.TransactionExceptionCode(exceptionCode)

	globalStatus := ReadByte(buf)
	msg.GlobalStatus = transaction.GlobalStatus(globalStatus)

	return msg
}
