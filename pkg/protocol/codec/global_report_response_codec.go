package codec

import (
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalReportResponseCodec{})
}

type GlobalReportResponseCodec struct {
	CommonGlobalEndResponseCodec
}

func (g *GlobalReportResponseCodec) Decode(in []byte) interface{} {
	req := g.CommonGlobalEndResponseCodec.Decode(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndResponse)
	return message.GlobalReportResponse{
		AbstractGlobalEndResponse: abstractGlobalEndRequest,
	}
}

func (g *GlobalReportResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalReportResult
}
