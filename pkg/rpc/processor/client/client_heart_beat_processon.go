package client

import (
	"context"
)

import (
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/rpc/getty"
	"github.com/seata/seata-go/pkg/utils/log"
)

func init() {
	getty.GetGettyClientHandlerInstance().RegisterProcessor(protocol.MessageTypeHeartbeatMsg, &clientHeartBeatProcesson{})
}

type clientHeartBeatProcesson struct{}

func (f *clientHeartBeatProcesson) Process(ctx context.Context, rpcMessage protocol.RpcMessage) error {
	if _, ok := rpcMessage.Body.(protocol.HeartBeatMessage); ok {
		// TODO 如何从context中获取远程服务的信息？
		//log.Infof("received PONG from {}", ctx.channel().remoteAddress())
		log.Infof("received PONG from {}", ctx)
	}
	return nil
}
