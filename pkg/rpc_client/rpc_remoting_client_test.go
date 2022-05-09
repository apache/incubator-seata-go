package rpc_client

import (
	"github.com/seata/seata-go/pkg/protocol"
	"testing"
	"time"
)

func TestSendMsgWithResponse(test *testing.T) {
	request := protocol.RegisterRMRequest{
		ResourceIds: "1111",
		AbstractIdentifyRequest: protocol.AbstractIdentifyRequest{
			ApplicationId:           "ApplicationID",
			TransactionServiceGroup: "TransactionServiceGroup",
		},
	}
	mergedMessage := protocol.MergedWarpMessage{
		Msgs:   []protocol.MessageTypeAware{request},
		MsgIds: []int32{1212},
	}
	handler := GetRpcRemoteClient()
	handler.sendMergedMessage(mergedMessage)
	time.Sleep(100000 * time.Second)
}
