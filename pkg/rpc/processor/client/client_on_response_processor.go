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
	clientOnResponseProcessor := &clientOnResponseProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(protocol.MessageTypeSeataMergeResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(protocol.MessageTypeBranchRegisterResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(protocol.MessageTypeBranchStatusReportResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(protocol.MessageTypeGlobalLockQueryResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(protocol.MessageTypeRegRmResult, clientOnResponseProcessor)
}

type clientOnResponseProcessor struct {
}

func (f *clientOnResponseProcessor) Process(ctx context.Context, rpcMessage protocol.RpcMessage) error {
	// 如果是合并的结果消息，直接通知已经处理完成
	if mergedResult, ok := rpcMessage.Body.(protocol.MergeResultMessage); ok {

		mergedMessage := getty.GetGettyRemotingInstance().GetMergedMessage(rpcMessage.ID)
		if mergedMessage != nil {
			for i := 0; i < len(mergedMessage.Msgs); i++ {
				msgID := mergedMessage.MsgIds[i]
				response := getty.GetGettyRemotingInstance().GetMessageFuture(msgID)
				if response != nil {
					response.Response = mergedResult.Msgs[i]
					response.Done <- true
					getty.GetGettyRemotingInstance().RemoveMessageFuture(msgID)
				}
			}
			getty.GetGettyRemotingInstance().RemoveMergedMessageFuture(rpcMessage.ID)
		}
		return nil
	} else {
		// 如果是请求消息，做处理逻辑
		msgFuture := getty.GetGettyRemotingInstance().GetMessageFuture(rpcMessage.ID)
		if msgFuture != nil {
			getty.GetGettyRemotingInstance().NotifytRpcMessageResponse(rpcMessage)
			getty.GetGettyRemotingInstance().RemoveMessageFuture(rpcMessage.ID)
		} else {
			if _, ok := rpcMessage.Body.(protocol.AbstractResultMessage); ok {
				log.Infof("the rm client received response msg [{}] from tc server.", msgFuture)
			}
		}
	}
	return nil
}
