package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &RegisterTMResponseCodec{})
}

type RegisterTMResponseCodec struct {
	AbstractIdentifyResponseCodec
}

func (g *RegisterTMResponseCodec) Decode(in []byte) interface{} {
	req := g.AbstractIdentifyResponseCodec.Decode(in)
	abstractIdentifyResponse := req.(message.AbstractIdentifyResponse)
	return message.RegisterTMResponse{
		AbstractIdentifyResponse: abstractIdentifyResponse,
	}
}

func (c *RegisterTMResponseCodec) Encode(in interface{}) []byte {
	resp := in.(message.RegisterTMResponse)
	return c.AbstractIdentifyResponseCodec.Encode(resp.AbstractIdentifyResponse)
}

func (g *RegisterTMResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_RegCltResult
}
