package client

import (
	"context"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/remoting/getty"
)

func init() {
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_HeartbeatMsg, &clientHeartBeatProcesson{})
}

type clientHeartBeatProcesson struct{}

func (f *clientHeartBeatProcesson) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	if _, ok := rpcMessage.Body.(message.HeartBeatMessage); ok {
		// TODO 如何从context中获取远程服务的信息？
		//log.Infof("received PONG from {}", ctx.channel().remoteAddress())
		log.Infof("received PONG from {}", ctx)
	}
	return nil
}
