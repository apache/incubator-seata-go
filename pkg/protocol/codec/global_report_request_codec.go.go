package codec

//func init() {
//	GetCodecManager().RegisterCodec(CodeTypeSeata, &GlobalReportRequestCodec{})
//}
//
//type GlobalReportRequestCodec struct {
//	CommonGlobalEndRequestCodec
//}
//
//func (g *GlobalReportRequestCodec) Decode(in []byte) interface{} {
//	req := g.CommonGlobalEndRequestCodec.Decode(in)
//	abstractGlobalEndRequest := req.(message.AbstractGlobalEndRequest)
//	return message.GlobalCommitRequest{
//		AbstractGlobalEndRequest: abstractGlobalEndRequest,
//	}
//}
//
//func (g *GlobalReportRequestCodec) GetMessageType() message.MessageType {
//	return message.MessageType_GlobalCommit
//}
