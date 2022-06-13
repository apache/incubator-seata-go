package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &RegisterTMRequestCodec{})
}

type RegisterTMRequestCodec struct {
	AbstractIdentifyRequestCodec
}

func (g *RegisterTMRequestCodec) Decode(in []byte) interface{} {
	req := g.AbstractIdentifyRequestCodec.Decode(in)
	abstractIdentifyRequest := req.(message.AbstractIdentifyRequest)
	return message.RegisterTMRequest{
		AbstractIdentifyRequest: abstractIdentifyRequest,
	}
}

func (c *RegisterTMRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.RegisterTMRequest)
	return c.AbstractIdentifyRequestCodec.Encode(req.AbstractIdentifyRequest)
}

func (g *RegisterTMRequestCodec) GetMessageType() message.MessageType {
	return message.MessageType_RegClt
}
