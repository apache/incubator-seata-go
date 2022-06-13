package codec

import (
	"github.com/fagongzi/goetty"
)

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

type CommonGlobalEndRequestCodec struct {
}

func (c *CommonGlobalEndRequestCodec) Encode(in interface{}) []byte {
	req, _ := in.(message.AbstractGlobalEndRequest)
	buf := goetty.NewByteBuf(0)

	Write16String(req.Xid, buf)
	Write16String(string(req.ExtraData), buf)

	return buf.RawBuf()
}

func (c *CommonGlobalEndRequestCodec) Decode(in []byte) interface{} {
	res := message.AbstractGlobalEndRequest{}

	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)

	var xidLen int
	if buf.Readable() >= 2 {
		xidLen = int(ReadUInt16(buf))
	}
	if buf.Readable() >= xidLen {
		xidBytes := make([]byte, xidLen)
		xidBytes = Read(buf, xidBytes)
		res.Xid = string(xidBytes)
	}

	var extraDataLen int
	if buf.Readable() >= 2 {
		extraDataLen = int(ReadUInt16(buf))
	}
	if buf.Readable() >= extraDataLen {
		extraDataBytes := make([]byte, xidLen)
		extraDataBytes = Read(buf, extraDataBytes)
		res.ExtraData = extraDataBytes
	}

	return res
}
