package rpc_client

import (
	"time"
)

import (
	"github.com/seata/seata-go/pkg/protocol"
)

type ClientMessageSender interface {

	// Send msg with response object.
	SendMsgWithResponse(msg interface{}) (interface{}, error)

	// Send msg with response object.
	SendMsgWithResponseAndTimeout(msg interface{}, timeout time.Duration) (interface{}, error)

	// Send response.
	SendResponse(request protocol.RpcMessage, serverAddress string, msg interface{})
}
