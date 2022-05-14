package rm

import (
	"sync"
)

import (
	"github.com/seata/seata-go/pkg/common/model"
	"github.com/seata/seata-go/pkg/protocol"
)

var (
	onceRMHandlerFacade = &sync.Once{}
	rmHandler           *RMHandlerFacade
)

type RMHandlerFacade struct {
	rmHandlerMap sync.Map
}

func GetRMHandlerFacadeInstance() *RMHandlerFacade {
	if rmHandler == nil {
		onceRMFacade.Do(func() {
			rmHandler = &RMHandlerFacade{}
		})
	}
	return rmHandler
}

// Handle branch commit response.
func (h *RMHandlerFacade) HandleBranchCommitRequest(request protocol.BranchCommitRequest) (*protocol.BranchCommitResponse, error) {
	return h.getRMHandler(request.BranchType).HandleBranchCommitRequest(request)
}

// Handle branch rollback response.
// TODO
func (h *RMHandlerFacade) HandleBranchRollbackRequest(request protocol.BranchRollbackRequest) (*protocol.BranchRollbackResponse, error) {
	return h.getRMHandler(request.BranchType).HandleBranchRollbackRequest(request)
}

// Handle delete undo log .
// TODO
func (h *RMHandlerFacade) HandleUndoLogDeleteRequest(request protocol.UndoLogDeleteRequest) error {
	return h.getRMHandler(request.BranchType).HandleUndoLogDeleteRequest(request)
}

func (h *RMHandlerFacade) RegisteRMHandler(handler *CommonRMHandler) {
	if handler == nil {
		return
	}
	h.rmHandlerMap.Store(handler.GetBranchType(), handler)
}

func (h *RMHandlerFacade) getRMHandler(branchType model.BranchType) *CommonRMHandler {
	if handler, ok := h.rmHandlerMap.Load(branchType); ok {
		return handler.(*CommonRMHandler)
	}
	return nil
}
