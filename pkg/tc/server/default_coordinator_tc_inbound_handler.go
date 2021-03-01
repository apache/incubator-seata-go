package server

import (
	"errors"
	"fmt"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/tc/holder"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

func (coordinator *DefaultCoordinator) doGlobalBegin(request protocal.GlobalBeginRequest, ctx RpcContext) protocal.GlobalBeginResponse {
	var resp = protocal.GlobalBeginResponse{}
	xid, err := coordinator.core.Begin(ctx.ApplicationID, ctx.TransactionServiceGroup, request.TransactionName, request.Timeout)
	if err != nil {
		trxException, ok := err.(*meta.TransactionException)
		resp.ResultCode = protocal.ResultCodeFailed
		if ok {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			log.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		log.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.Xid = xid
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doGlobalStatus(request protocal.GlobalStatusRequest, ctx RpcContext) protocal.GlobalStatusResponse {
	var resp = protocal.GlobalStatusResponse{}
	globalStatus, err := coordinator.core.GetStatus(request.XID)
	if err != nil {
		resp.ResultCode = protocal.ResultCodeFailed
		var trxException *meta.TransactionException
		if errors.As(err, &trxException) {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			log.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		log.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.GlobalStatus = globalStatus
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doGlobalReport(request protocal.GlobalReportRequest, ctx RpcContext) protocal.GlobalReportResponse {
	var resp = protocal.GlobalReportResponse{}
	globalStatus, err := coordinator.core.GlobalReport(request.XID, request.GlobalStatus)
	if err != nil {
		resp.ResultCode = protocal.ResultCodeFailed
		var trxException *meta.TransactionException
		if errors.As(err, &trxException) {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			log.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		log.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.GlobalStatus = globalStatus
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doGlobalCommit(request protocal.GlobalCommitRequest, ctx RpcContext) protocal.GlobalCommitResponse {
	var resp = protocal.GlobalCommitResponse{}
	globalStatus, err := coordinator.core.Commit(request.XID)
	if err != nil {
		resp.ResultCode = protocal.ResultCodeFailed
		var trxException *meta.TransactionException
		if errors.As(err, &trxException) {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			log.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		log.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.GlobalStatus = globalStatus
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doGlobalRollback(request protocal.GlobalRollbackRequest, ctx RpcContext) protocal.GlobalRollbackResponse {
	var resp = protocal.GlobalRollbackResponse{}
	globalStatus, err := coordinator.core.Rollback(request.XID)
	if err != nil {
		resp.ResultCode = protocal.ResultCodeFailed
		globalSession := holder.GetSessionHolder().FindGlobalSessionWithBranchSessions(request.XID, false)
		if globalSession == nil {
			resp.GlobalStatus = meta.GlobalStatusFinished
		} else {
			resp.GlobalStatus = globalSession.Status
		}

		var trxException *meta.TransactionException
		if errors.As(err, &trxException) {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			log.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		log.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.GlobalStatus = globalStatus
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doBranchRegister(request protocal.BranchRegisterRequest, ctx RpcContext) protocal.BranchRegisterResponse {
	var resp = protocal.BranchRegisterResponse{}
	branchID, err := coordinator.core.BranchRegister(request.BranchType, request.ResourceID, ctx.ClientID, request.XID, request.ApplicationData, request.LockKey)
	if err != nil {
		resp.ResultCode = protocal.ResultCodeFailed
		var trxException *meta.TransactionException
		if errors.As(err, &trxException) {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			log.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		log.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.BranchID = branchID
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doBranchReport(request protocal.BranchReportRequest, ctx RpcContext) protocal.BranchReportResponse {
	var resp = protocal.BranchReportResponse{}
	err := coordinator.core.BranchReport(request.BranchType, request.XID, request.BranchID, request.Status, request.ApplicationData)
	if err != nil {
		resp.ResultCode = protocal.ResultCodeFailed
		var trxException *meta.TransactionException
		if errors.As(err, &trxException) {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			log.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		log.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doLockCheck(request protocal.GlobalLockQueryRequest, ctx RpcContext) protocal.GlobalLockQueryResponse {
	var resp = protocal.GlobalLockQueryResponse{}
	result, err := coordinator.core.LockQuery(request.BranchType, request.ResourceID, request.XID, request.LockKey)
	if err != nil {
		resp.ResultCode = protocal.ResultCodeFailed
		var trxException *meta.TransactionException
		if errors.As(err, &trxException) {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			log.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		log.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.Lockable = result
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}
