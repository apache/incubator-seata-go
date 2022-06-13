package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalRollbackRequestCodec{})
}

type GlobalRollbackRequestCodec struct {
	CommonGlobalEndRequestCodec
}

func (g *GlobalRollbackRequestCodec) Decode(in []byte) interface{} {
	req := g.CommonGlobalEndRequestCodec.Decode(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndRequest)
	return message.GlobalCommitRequest{
		AbstractGlobalEndRequest: abstractGlobalEndRequest,
	}
}

func (g *GlobalRollbackRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.GlobalRollbackRequest)
	return g.CommonGlobalEndRequestCodec.Encode(req.AbstractGlobalEndRequest)
}

func (g *GlobalRollbackRequestCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalRollback
}
