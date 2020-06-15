package tm

import (
	"github.com/pkg/errors"
	"github.com/xiaobudongzhang/seata-golang/base/meta"
	"github.com/xiaobudongzhang/seata-golang/base/protocal"
	"github.com/xiaobudongzhang/seata-golang/client/getty"
)

type DefaultTransactionManager struct {
	rpcClient *getty.RpcRemoteClient
}

func (manager DefaultTransactionManager) Begin(applicationId string, transactionServiceGroup string, name string, timeout int32) (string, error) {
	request := protocal.GlobalBeginRequest{
		Timeout:         timeout,
		TransactionName: name,
	}
	resp, err := manager.syncCall(request)
	if err != nil {
		return "", errors.WithStack(err)
	}
	response := resp.(protocal.GlobalBeginResponse)
	return response.Xid, nil
}

func (manager DefaultTransactionManager) Commit(xid string) (meta.GlobalStatus, error) {
	globalCommit := protocal.GlobalCommitRequest{AbstractGlobalEndRequest: protocal.AbstractGlobalEndRequest{Xid: xid}}
	resp, err := manager.syncCall(globalCommit)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	response := resp.(protocal.GlobalCommitResponse)
	return response.GlobalStatus, nil
}

func (manager DefaultTransactionManager) Rollback(xid string) (meta.GlobalStatus, error) {
	globalRollback := protocal.GlobalRollbackRequest{AbstractGlobalEndRequest: protocal.AbstractGlobalEndRequest{Xid: xid}}
	resp, err := manager.syncCall(globalRollback)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	response := resp.(protocal.GlobalRollbackResponse)
	return response.GlobalStatus, nil
}

func (manager DefaultTransactionManager) GetStatus(xid string) (meta.GlobalStatus, error) {
	queryGlobalStatus := protocal.GlobalStatusRequest{AbstractGlobalEndRequest: protocal.AbstractGlobalEndRequest{Xid: xid}}
	resp, err := manager.syncCall(queryGlobalStatus)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	response := resp.(protocal.GlobalStatusResponse)
	return response.GlobalStatus, nil
}

func (manager DefaultTransactionManager) GlobalReport(xid string, globalStatus meta.GlobalStatus) (meta.GlobalStatus, error) {
	globalReport := protocal.GlobalReportRequest{
		AbstractGlobalEndRequest: protocal.AbstractGlobalEndRequest{Xid: xid},
		GlobalStatus:             globalStatus,
	}
	resp, err := manager.syncCall(globalReport)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	response := resp.(protocal.GlobalReportResponse)
	return response.GlobalStatus, nil
}

func (manager DefaultTransactionManager) syncCall(request interface{}) (interface{}, error) {
	return manager.rpcClient.SendMsgWithResponse(request)
}
