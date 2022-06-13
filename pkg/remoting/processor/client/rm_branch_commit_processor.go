package client

import (
	"context"
)

import (
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"
	getty2 "github.com/seata/seata-go/pkg/remoting/getty"
	"github.com/seata/seata-go/pkg/rm"
)

func init() {
	rmBranchCommitProcessor := &rmBranchCommitProcessor{}
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchCommit, rmBranchCommitProcessor)
}

type rmBranchCommitProcessor struct {
}

func (f *rmBranchCommitProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	log.Infof("rm client handle branch commit process %v", rpcMessage)
	request := rpcMessage.Body.(message.BranchCommitRequest)
	xid := request.Xid
	branchID := request.BranchId
	resourceID := request.ResourceId
	applicationData := request.ApplicationData
	log.Infof("Branch committing: xid %s, branchID %s, resourceID %s, applicationData %s", xid, branchID, resourceID, applicationData)

	status, err := rm.GetResourceManagerInstance().GetResourceManager(request.BranchType).BranchCommit(ctx, request.BranchType, xid, branchID, resourceID, applicationData)
	if err != nil {
		log.Infof("Branch commit error: %s", err.Error())
		return err
	}

	// reply commit response to tc server
	response := message.BranchCommitResponse{
		AbstractBranchEndResponse: message.AbstractBranchEndResponse{
			Xid:          xid,
			BranchId:     branchID,
			BranchStatus: status,
		},
	}
	err = getty2.GetGettyRemotingClient().SendAsyncResponse(response)
	if err != nil {
		log.Error("BranchCommitResponse error: {%#v}", err.Error())
		return err
	}
	return nil
}
