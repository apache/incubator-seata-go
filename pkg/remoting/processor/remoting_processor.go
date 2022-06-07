package processor

import (
	"context"
	"github.com/seata/seata-go/pkg/protocol/message"
)

type RemotingProcessor interface {
	Process(ctx context.Context, rpcMessage message.RpcMessage) error
}
