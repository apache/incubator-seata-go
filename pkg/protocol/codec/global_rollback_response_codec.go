package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalRollbackResponseCodec{})
}

type GlobalRollbackResponseCodec struct {
	CommonGlobalEndResponseCodec
}

func (g *GlobalRollbackResponseCodec) Decode(in []byte) interface{} {
	req := g.CommonGlobalEndResponseCodec.Decode(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndResponse)
	return message.GlobalRollbackResponse{
		AbstractGlobalEndResponse: abstractGlobalEndRequest,
	}
}

func (g *GlobalRollbackResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalRollbackResult
}
