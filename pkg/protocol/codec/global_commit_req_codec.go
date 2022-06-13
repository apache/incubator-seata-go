package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalCommitRequestCodec{})
}

type GlobalCommitRequestCodec struct {
	CommonGlobalEndRequestCodec
}

func (g *GlobalCommitRequestCodec) Decode(in []byte) interface{} {
	req := g.CommonGlobalEndRequestCodec.Decode(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndRequest)
	return message.GlobalCommitRequest{
		AbstractGlobalEndRequest: abstractGlobalEndRequest,
	}
}

func (g *GlobalCommitRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.GlobalCommitRequest)
	return g.CommonGlobalEndRequestCodec.Encode(req.AbstractGlobalEndRequest)
}

func (g *GlobalCommitRequestCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalCommit
}
