package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalStatusResponseCodec{})
}

type GlobalStatusResponseCodec struct {
	CommonGlobalEndResponseCodec
}

func (g *GlobalStatusResponseCodec) Decode(in []byte) interface{} {
	req := g.CommonGlobalEndResponseCodec.Decode(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndResponse)
	return message.GlobalStatusResponse{
		AbstractGlobalEndResponse: abstractGlobalEndRequest,
	}
}

func (g *GlobalStatusResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalStatusResult
}
