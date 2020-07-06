package server

import "github.com/dk-lockdown/seata-golang/base/protocal"

type TCInboundHandler interface {
	doGlobalBegin(request protocal.GlobalBeginRequest, ctx RpcContext) protocal.GlobalBeginResponse
	doGlobalStatus(request protocal.GlobalStatusRequest, ctx RpcContext) protocal.GlobalStatusResponse
	doGlobalReport(request protocal.GlobalReportRequest, ctx RpcContext) protocal.GlobalReportResponse
	doGlobalCommit(request protocal.GlobalCommitRequest, ctx RpcContext) protocal.GlobalCommitResponse
	doGlobalRollback(request protocal.GlobalRollbackRequest, ctx RpcContext) protocal.GlobalRollbackResponse
	doBranchRegister(request protocal.BranchRegisterRequest, ctx RpcContext) protocal.BranchRegisterResponse
	doBranchReport(request protocal.BranchReportRequest, ctx RpcContext) protocal.BranchReportResponse
	doLockCheck(request protocal.GlobalLockQueryRequest, ctx RpcContext) protocal.GlobalLockQueryResponse
}
