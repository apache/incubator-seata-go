package exec

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/client/at/undo/manager"
	"github.com/transaction-wg/seata-golang/pkg/client/rm"
	"github.com/transaction-wg/seata-golang/pkg/client/rpc_client"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

type DataSourceManager struct {
	rm.AbstractResourceManager
}

var dataSourceManager DataSourceManager

func InitDataResourceManager() {
	dataSourceManager = DataSourceManager{
		AbstractResourceManager: rm.NewAbstractResourceManager(rpc_client.GetRpcRemoteClient()),
	}
	go dataSourceManager.handleBranchCommit()
	go dataSourceManager.handleBranchRollback()
}

func NewDataResourceManager(client *rpc_client.RpcRemoteClient) *DataSourceManager{
	dataSourceManager := &DataSourceManager{
		AbstractResourceManager: rm.NewAbstractResourceManager(client),
	}
	go dataSourceManager.handleBranchCommit()
	go dataSourceManager.handleBranchRollback()

	return dataSourceManager
}


func (resourceManager DataSourceManager) LockQuery(branchType meta.BranchType, resourceID string, xid string,
	lockKeys string) (bool, error) {
	request := protocal.GlobalLockQueryRequest{
		BranchRegisterRequest: protocal.BranchRegisterRequest{
			XID:        xid,
			ResourceID: resourceID,
			LockKey:    lockKeys,
		}}

	var response protocal.GlobalLockQueryResponse
	resp, err := resourceManager.RpcClient.SendMsgWithResponse(request)
	if err != nil {
		return false, errors.WithStack(err)
	}
	response = resp.(protocal.GlobalLockQueryResponse)

	if response.ResultCode == protocal.ResultCodeFailed {
		return false, errors.Errorf("Response[ %s ]", response.Msg)
	}
	return response.Lockable, nil
}

func (resourceManager DataSourceManager) BranchCommit(branchType meta.BranchType, xid string, branchID int64,
	resourceID string, applicationData []byte) (meta.BranchStatus, error) {
	//todo 改为异步批量操作
	undoLogManager := manager.GetUndoLogManager()
	db := resourceManager.getDB(resourceID)
	err := undoLogManager.DeleteUndoLog(db.DB, xid, branchID)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return meta.BranchStatusPhasetwoCommitted, nil
}

func (resourceManager DataSourceManager) BranchRollback(branchType meta.BranchType, xid string, branchID int64,
	resourceID string, applicationData []byte) (meta.BranchStatus, error) {
	//todo 使用前镜数据覆盖当前数据
	undoLogManager := manager.GetUndoLogManager()
	db := resourceManager.getDB(resourceID)
	err := undoLogManager.Undo(db.DB, xid, branchID, db.GetResourceID())
	if err != nil {
		log.Errorf("[stacktrace]branchRollback failed. branchType:[%d], xid:[%s], branchID:[%d], resourceID:[%s], applicationData:[%v]",
			branchType, xid, branchID, resourceID, applicationData)
		log.Error(err)
		return meta.BranchStatusPhasetwoCommitFailedRetryable, nil
	}
	return meta.BranchStatusPhasetwoRollbacked, nil
}

func (resourceManager DataSourceManager) GetBranchType() meta.BranchType {
	return meta.BranchTypeAT
}

func (resourceManager DataSourceManager) getDB(resourceID string) *DB {
	resource := resourceManager.ResourceCache[resourceID]
	db := resource.(*DB)
	return db
}

func (resourceManager DataSourceManager) handleBranchCommit() {
	for {
		rpcRMMessage := <-resourceManager.RpcClient.BranchCommitRequestChannel
		rpcMessage := rpcRMMessage.RpcMessage
		serviceAddress := rpcRMMessage.ServerAddress

		req := rpcMessage.Body.(protocal.BranchCommitRequest)
		resp := resourceManager.doBranchCommit(req)
		resourceManager.RpcClient.SendResponse(rpcMessage, serviceAddress, resp)
	}
}

func (resourceManager DataSourceManager) handleBranchRollback() {
	for {
		rpcRMMessage := <-resourceManager.RpcClient.BranchRollbackRequestChannel
		rpcMessage := rpcRMMessage.RpcMessage
		serviceAddress := rpcRMMessage.ServerAddress

		req := rpcMessage.Body.(protocal.BranchRollbackRequest)
		resp := resourceManager.doBranchRollback(req)
		resourceManager.RpcClient.SendResponse(rpcMessage, serviceAddress, resp)
	}
}

func (resourceManager DataSourceManager) doBranchCommit(request protocal.BranchCommitRequest) protocal.BranchCommitResponse {
	var resp = protocal.BranchCommitResponse{}

	log.Infof("Branch committing: %s %d %s %s", request.XID, request.BranchID, request.ResourceID, request.ApplicationData)
	status, err := resourceManager.BranchCommit(request.BranchType, request.XID, request.BranchID, request.ResourceID, request.ApplicationData)
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
	resp.XID = request.XID
	resp.BranchID = request.BranchID
	resp.BranchStatus = status
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (resourceManager DataSourceManager) doBranchRollback(request protocal.BranchRollbackRequest) protocal.BranchRollbackResponse {
	var resp = protocal.BranchRollbackResponse{}

	log.Infof("Branch rollbacking: %s %d %s", request.XID, request.BranchID, request.ResourceID)
	status, err := resourceManager.BranchRollback(request.BranchType, request.XID, request.BranchID, request.ResourceID, request.ApplicationData)
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
	resp.XID = request.XID
	resp.BranchID = request.BranchID
	resp.BranchStatus = status
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}
