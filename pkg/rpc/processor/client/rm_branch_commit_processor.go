package client

import (
	"context"
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/rpc/getty"
	"github.com/seata/seata-go/pkg/utils/log"
)

func init() {
	rmBranchCommitProcessor := &rmBranchCommitProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(protocol.MessageTypeBranchCommit, rmBranchCommitProcessor)
}

type rmBranchCommitProcessor struct {
}

func (f *rmBranchCommitProcessor) Process(ctx context.Context, rpcMessage protocol.RpcMessage) error {
	log.Infof("rm client handle branch commit process %v", rpcMessage)
	request := rpcMessage.Body.(protocol.BranchCommitRequest)
	xid := request.Xid
	branchID := request.BranchId
	resourceID := request.ResourceId
	applicationData := request.ApplicationData
	log.Infof("Branch committing: xid %s, branchID %s, resourceID %s, applicationData %s", xid, branchID, resourceID, applicationData)

	status, err := rm.GetResourceManagerFacadeInstance().GetResourceManager(request.BranchType).BranchCommit(ctx, request.BranchType, xid, branchID, resourceID, applicationData)
	if err != nil {
		log.Infof("Branch commit error: %s", err.Error())
		return err
	}

	// reply commit response to tc server
	response := protocol.BranchCommitResponse{
		AbstractBranchEndResponse: protocol.AbstractBranchEndResponse{
			Xid:          xid,
			BranchId:     branchID,
			BranchStatus: status,
		},
	}
	err = getty.GetGettyRemotingClient().SendAsyncResponse(response)
	if err != nil {
		log.Error("BranchCommitResponse error: {%#v}", err.Error())
		return err
	}
	return nil
}
