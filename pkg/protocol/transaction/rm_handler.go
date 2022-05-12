package transaction

import (
	"github.com/seata/seata-go/pkg/protocol"
)

type RMInboundHandler interface {

	/**
	 * Handle branch commit response.
	 *
	 * @param request the request
	 * @return the branch commit response
	 */
	HandleBranchCommitRequest(request protocol.BranchCommitRequest) (*protocol.BranchCommitResponse, error)

	/**
	 * Handle branch rollback response.
	 *
	 * @param request the request
	 * @return the branch rollback response
	 */

	HandleBranchRollbackRequest(request protocol.BranchRollbackRequest) (*protocol.BranchRollbackResponse, error)

	/**
	 * Handle delete undo log .
	 *
	 * @param request the request
	 */
	HandleUndoLogDeleteRequest(request protocol.UndoLogDeleteRequest) error
}
