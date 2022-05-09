package rpc11

import (
	"testing"
	"time"
)

import (
	"github.com/seata/seata-go/pkg/protocol"
)

func TestSendMergedMessage(test *testing.T) {
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
	handler := NewClientEventHandler()
	handler.sendMergedMessage(mergedMessage)
	time.Sleep(100000 * time.Second)
}
