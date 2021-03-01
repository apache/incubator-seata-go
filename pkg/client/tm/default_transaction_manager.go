package tm

import (
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/client/rpc_client"
)

type DefaultTransactionManager struct {
	rpcClient *rpc_client.RpcRemoteClient
}

func (manager DefaultTransactionManager) Begin(applicationID string, transactionServiceGroup string, name string, timeout int32) (string, error) {
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
	globalCommit := protocal.GlobalCommitRequest{AbstractGlobalEndRequest: protocal.AbstractGlobalEndRequest{XID: xid}}
	resp, err := manager.syncCall(globalCommit)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	response := resp.(protocal.GlobalCommitResponse)
	return response.GlobalStatus, nil
}

func (manager DefaultTransactionManager) Rollback(xid string) (meta.GlobalStatus, error) {
	globalRollback := protocal.GlobalRollbackRequest{AbstractGlobalEndRequest: protocal.AbstractGlobalEndRequest{XID: xid}}
	resp, err := manager.syncCall(globalRollback)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	response := resp.(protocal.GlobalRollbackResponse)
	return response.GlobalStatus, nil
}

func (manager DefaultTransactionManager) GetStatus(xid string) (meta.GlobalStatus, error) {
	queryGlobalStatus := protocal.GlobalStatusRequest{AbstractGlobalEndRequest: protocal.AbstractGlobalEndRequest{XID: xid}}
	resp, err := manager.syncCall(queryGlobalStatus)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	response := resp.(protocal.GlobalStatusResponse)
	return response.GlobalStatus, nil
}

func (manager DefaultTransactionManager) GlobalReport(xid string, globalStatus meta.GlobalStatus) (meta.GlobalStatus, error) {
	globalReport := protocal.GlobalReportRequest{
		AbstractGlobalEndRequest: protocal.AbstractGlobalEndRequest{XID: xid},
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
