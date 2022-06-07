package handler

import (
	"context"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/resource"
)

type RMInboundHandler interface {
	// Handle branch commit response.
	HandleBranchCommitRequest(ctx context.Context, request message.BranchCommitRequest) (*message.BranchCommitResponse, error)
	// Handle branch rollback response.
	HandleBranchRollbackRequest(ctx context.Context, request message.BranchRollbackRequest) (*message.BranchRollbackResponse, error)
	// Handle delete undo log .
	HandleUndoLogDeleteRequest(ctx context.Context, request message.UndoLogDeleteRequest) error
}

type CommonRMHandler struct {
	rmGetter resource.ResourceManagerGetter
}

func (h *CommonRMHandler) SetRMGetter(rmGetter resource.ResourceManagerGetter) {
	h.rmGetter = rmGetter
}

// Handle branch commit response.
func (h *CommonRMHandler) HandleBranchCommitRequest(ctx context.Context, request message.BranchCommitRequest) (*message.BranchCommitResponse, error) {
	xid := request.Xid
	branchID := request.BranchId
	resourceID := request.ResourceId
	applicationData := request.ApplicationData
	log.Infof("Branch committing: xid %s, branchID %s, resourceID %s, applicationData %s", xid, branchID, resourceID, applicationData)

	status, err := h.rmGetter.GetResourceManager().BranchCommit(ctx, request.BranchType, xid, branchID, resourceID, applicationData)
	if err != nil {
		// TODO: handle error
		return nil, err
	}
	return &message.BranchCommitResponse{
		AbstractBranchEndResponse: message.AbstractBranchEndResponse{
			Xid:          xid,
			BranchId:     branchID,
			BranchStatus: status,
		},
	}, nil
}

// Handle branch rollback response.
// TODO
func (h *CommonRMHandler) HandleBranchRollbackRequest(ctx context.Context, request message.BranchRollbackRequest) (*message.BranchRollbackResponse, error) {
	return nil, nil
}

// Handle delete undo log .
// TODO
func (h *CommonRMHandler) HandleUndoLogDeleteRequest(ctx context.Context, request message.UndoLogDeleteRequest) error {
	return nil
}

func (h *CommonRMHandler) GetBranchType() branch.BranchType {
	return h.rmGetter.GetResourceManager().GetBranchType()
}
