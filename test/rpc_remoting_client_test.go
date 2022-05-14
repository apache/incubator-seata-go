package test

import (
	_ "github.com/seata/seata-go/pkg/imports"
	"testing"
	"time"
)

func TestSendMsgWithResponse(test *testing.T) {
	//request := protocol.RegisterRMRequest{
	//	ResourceIds: "1111",
	//	AbstractIdentifyRequest: protocol.AbstractIdentifyRequest{
	//		ApplicationId:           "ApplicationID",
	//		TransactionServiceGroup: "TransactionServiceGroup",
	//	},
	//}
	//mergedMessage := protocol.MergedWarpMessage{
	//	Msgs:   []protocol.MessageTypeAware{request},
	//	MsgIds: []int32{1212},
	//}
	//handler := GetGettyClientHandlerInstance()
	//handler.sendMergedMessage(mergedMessage)
	time.Sleep(100000 * time.Second)
}
