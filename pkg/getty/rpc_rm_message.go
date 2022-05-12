package getty

import (
	"github.com/seata/seata-go/pkg/protocol"
)

type RpcRMMessage struct {
	RpcMessage    protocol.RpcMessage
	ServerAddress string
}
