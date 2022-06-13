package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalCommitResponseCodec{})
}

type GlobalCommitResponseCodec struct {
	CommonGlobalEndResponseCodec
}

func (g *GlobalCommitResponseCodec) Decode(in []byte) interface{} {
	req := g.CommonGlobalEndResponseCodec.Decode(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndResponse)
	return message.GlobalCommitResponse{
		AbstractGlobalEndResponse: abstractGlobalEndRequest,
	}
}

func (g *GlobalCommitResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalCommitResult
}
