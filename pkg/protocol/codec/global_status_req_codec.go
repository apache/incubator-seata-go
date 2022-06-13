package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalStatusRequestCodec{})
}

type GlobalStatusRequestCodec struct {
	CommonGlobalEndRequestCodec
}

func (g *GlobalStatusRequestCodec) Decode(in []byte) interface{} {
	req := g.CommonGlobalEndRequestCodec.Decode(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndRequest)
	return message.GlobalStatusRequest{
		AbstractGlobalEndRequest: abstractGlobalEndRequest,
	}
}

func (g *GlobalStatusRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.GlobalStatusRequest)
	return g.CommonGlobalEndRequestCodec.Encode(req.AbstractGlobalEndRequest)
}

func (g *GlobalStatusRequestCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalStatus
}
