package rpc_client

import (
	"github.com/seata/seata-go/pkg/protocol"
)

type RpcRMMessage struct {
	RpcMessage    protocol.RpcMessage
	ServerAddress string
}
