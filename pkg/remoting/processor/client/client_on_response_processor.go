package client

import (
	"context"
)

import (
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"
	getty2 "github.com/seata/seata-go/pkg/remoting/getty"
)

func init() {
	clientOnResponseProcessor := &clientOnResponseProcessor{}
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_SeataMergeResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchRegisterResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchStatusReportResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalLockQueryResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_RegRmResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalBeginResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalCommitResult, clientOnResponseProcessor)
}

type clientOnResponseProcessor struct {
}

func (f *clientOnResponseProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	// 如果是合并的结果消息，直接通知已经处理完成
	if mergedResult, ok := rpcMessage.Body.(message.MergeResultMessage); ok {

		mergedMessage := getty2.GetGettyRemotingInstance().GetMergedMessage(rpcMessage.ID)
		if mergedMessage != nil {
			for i := 0; i < len(mergedMessage.Msgs); i++ {
				msgID := mergedMessage.MsgIds[i]
				response := getty2.GetGettyRemotingInstance().GetMessageFuture(msgID)
				if response != nil {
					response.Response = mergedResult.Msgs[i]
					response.Done <- true
					getty2.GetGettyRemotingInstance().RemoveMessageFuture(msgID)
				}
			}
			getty2.GetGettyRemotingInstance().RemoveMergedMessageFuture(rpcMessage.ID)
		}
		return nil
	} else {
		// 如果是请求消息，做处理逻辑
		msgFuture := getty2.GetGettyRemotingInstance().GetMessageFuture(rpcMessage.ID)
		if msgFuture != nil {
			getty2.GetGettyRemotingInstance().NotifyRpcMessageResponse(rpcMessage)
			getty2.GetGettyRemotingInstance().RemoveMessageFuture(rpcMessage.ID)
		} else {
			if _, ok := rpcMessage.Body.(message.AbstractResultMessage); ok {
				log.Infof("the rm client received response msg [{}] from tc server.", msgFuture)
			}
		}
	}
	return nil
}
