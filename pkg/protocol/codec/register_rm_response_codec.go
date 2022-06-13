package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &RegisterRMResponseCodec{})
}

type RegisterRMResponseCodec struct {
	AbstractIdentifyResponseCodec
}

func (g *RegisterRMResponseCodec) Decode(in []byte) interface{} {
	req := g.AbstractIdentifyResponseCodec.Decode(in)
	abstractIdentifyResponse := req.(message.AbstractIdentifyResponse)
	return message.RegisterRMResponse{
		AbstractIdentifyResponse: abstractIdentifyResponse,
	}
}

func (g *RegisterRMResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_RegRmResult
}
