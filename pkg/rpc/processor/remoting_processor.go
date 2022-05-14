package processor

import (
	"context"
)

import (
	"github.com/seata/seata-go/pkg/protocol"
)

type RemotingProcessor interface {
	Process(ctx context.Context, rpcMessage protocol.RpcMessage) error
}
