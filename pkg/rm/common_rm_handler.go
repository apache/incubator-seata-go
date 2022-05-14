package rm

import (
	"github.com/seata/seata-go/pkg/common/model"
	"github.com/seata/seata-go/pkg/protocol"
)

type CommonRMHandler struct {
	rmGetter model.ResourceManagerGetter
}

func (h *CommonRMHandler) SetRMGetter(rmGetter model.ResourceManagerGetter) {
	h.rmGetter = rmGetter
}

// Handle branch commit response.
func (h *CommonRMHandler) HandleBranchCommitRequest(request protocol.BranchCommitRequest) (*protocol.BranchCommitResponse, error) {
	xid := request.Xid
	branchID := request.BranchId
	resourceID := request.ResourceId
	applicationData := request.ApplicationData

	status, err := h.rmGetter.GetResourceManager().BranchCommit(request.BranchType, xid, branchID, resourceID, applicationData)
	if err != nil {
		// TODO: handle error
		return nil, err
	}
	return &protocol.BranchCommitResponse{
		AbstractBranchEndResponse: protocol.AbstractBranchEndResponse{
			Xid:          xid,
			BranchId:     branchID,
			BranchStatus: status,
		},
	}, nil
}

// Handle branch rollback response.
// TODO
func (h *CommonRMHandler) HandleBranchRollbackRequest(request protocol.BranchRollbackRequest) (*protocol.BranchRollbackResponse, error) {
	return nil, nil
}

// Handle delete undo log .
// TODO
func (h *CommonRMHandler) HandleUndoLogDeleteRequest(request protocol.UndoLogDeleteRequest) error {
	return nil
}

func (h *CommonRMHandler) GetBranchType() model.BranchType {
	return h.rmGetter.GetResourceManager().GetBranchType()
}
