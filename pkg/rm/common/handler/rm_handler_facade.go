package handler

import (
	"context"
	"sync"
)

import (
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
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
		onceRMHandlerFacade.Do(func() {
			rmHandler = &RMHandlerFacade{}
		})
	}
	return rmHandler
}

// Handle branch commit response.
func (h *RMHandlerFacade) HandleBranchCommitRequest(ctx context.Context, request message.BranchCommitRequest) (*message.BranchCommitResponse, error) {
	return h.getRMHandler(request.BranchType).HandleBranchCommitRequest(ctx, request)
}

// Handle branch rollback response.
// TODO
func (h *RMHandlerFacade) HandleBranchRollbackRequest(ctx context.Context, request message.BranchRollbackRequest) (*message.BranchRollbackResponse, error) {
	return h.getRMHandler(request.BranchType).HandleBranchRollbackRequest(ctx, request)
}

// Handle delete undo log .
// TODO
func (h *RMHandlerFacade) HandleUndoLogDeleteRequest(ctx context.Context, request message.UndoLogDeleteRequest) error {
	return h.getRMHandler(request.BranchType).HandleUndoLogDeleteRequest(ctx, request)
}

func (h *RMHandlerFacade) RegisteRMHandler(handler *CommonRMHandler) {
	if handler == nil {
		return
	}
	h.rmHandlerMap.Store(handler.GetBranchType(), handler)
}

func (h *RMHandlerFacade) getRMHandler(branchType branch.BranchType) *CommonRMHandler {
	if handler, ok := h.rmHandlerMap.Load(branchType); ok {
		return handler.(*CommonRMHandler)
	}
	return nil
}
