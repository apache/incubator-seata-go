package tcc

import (
	"encoding/json"
	"fmt"
	"strconv"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/client/context"
	"github.com/transaction-wg/seata-golang/pkg/client/proxy"
	"github.com/transaction-wg/seata-golang/pkg/client/rm"
	"github.com/transaction-wg/seata-golang/pkg/client/rpc_client"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

var (
	TCC_ACTION_CONTEXT = "actionContext"
)

var tccResourceManager TCCResourceManager

func InitTCCResourceManager() {
	tccResourceManager = TCCResourceManager{
		AbstractResourceManager: rm.NewAbstractResourceManager(rpc_client.GetRpcRemoteClient()),
	}
	go tccResourceManager.handleBranchCommit()
	go tccResourceManager.handleBranchRollback()
}

type TCCResourceManager struct {
	rm.AbstractResourceManager
}

func (resourceManager TCCResourceManager) BranchCommit(branchType meta.BranchType, xid string, branchID int64,
	resourceID string, applicationData []byte) (meta.BranchStatus, error) {
	resource := resourceManager.ResourceCache[resourceID]
	if resource == nil {
		log.Errorf("TCC resource is not exist, resourceID: %s", resourceID)
		return 0, errors.Errorf("TCC resource is not exist, resourceID: %s", resourceID)
	}
	tccResource := resource.(*TCCResource)
	if tccResource.CommitMethod == nil {
		log.Errorf("TCC resource is not available, resourceID: %s", resourceID)
		return 0, errors.Errorf("TCC resource is not available, resourceID: %s", resourceID)
	}

	result := false
	businessActionContext := getBusinessActionContext(xid, branchID, resourceID, applicationData)
	args := make([]interface{}, 0)
	args = append(args, businessActionContext)
	returnValues := proxy.Invoke(tccResource.CommitMethod, nil, args)
	log.Infof("TCC resource commit result : %v, xid: %s, branchID: %d, resourceID: %s", returnValues, xid, branchID, resourceID)
	if returnValues != nil && len(returnValues) == 1 {
		result = returnValues[0].Interface().(bool)
	}
	if result {
		return meta.BranchStatusPhaseTwoCommitted, nil
	} else {
		return meta.BranchStatusPhaseTwoCommitFailedRetryable, nil
	}
}

func (resourceManager TCCResourceManager) BranchRollback(branchType meta.BranchType, xid string, branchID int64,
	resourceID string, applicationData []byte) (meta.BranchStatus, error) {
	resource := resourceManager.ResourceCache[resourceID]
	if resource == nil {
		return 0, errors.Errorf("TCC resource is not exist, resourceID: %s", resourceID)
	}
	tccResource := resource.(*TCCResource)
	if tccResource.RollbackMethod == nil {
		return 0, errors.Errorf("TCC resource is not available, resourceID: %s", resourceID)
	}

	result := false
	businessActionContext := getBusinessActionContext(xid, branchID, resourceID, applicationData)
	args := make([]interface{}, 0)
	args = append(args, businessActionContext)
	returnValues := proxy.Invoke(tccResource.RollbackMethod, nil, args)
	log.Infof("TCC resource rollback result : %v, xid: %s, branchID: %d, resourceID: %s", returnValues, xid, branchID, resourceID)
	if returnValues != nil && len(returnValues) == 1 {
		result = returnValues[0].Interface().(bool)
	}
	if result {
		return meta.BranchStatusPhaseTwoRolledBack, nil
	} else {
		return meta.BranchStatusPhaseTwoRollbackFailedRetryable, nil
	}
}

func (resourceManager TCCResourceManager) GetBranchType() meta.BranchType {
	return meta.BranchTypeTCC
}

func getBusinessActionContext(xid string, branchID int64, resourceID string, applicationData []byte) *context.BusinessActionContext {
	var (
		tccContext       = make(map[string]interface{})
		actionContextMap = make(map[string]interface{})
	)
	if len(applicationData) > 0 {
		err := json.Unmarshal(applicationData, &tccContext)
		if err != nil {
			log.Errorf("getBusinessActionContext, unmarshal applicationData err=%v", err)
		}
	}

	acMap := tccContext[TCC_ACTION_CONTEXT]
	if acMap != nil {
		actionContextMap = acMap.(map[string]interface{})
	}

	businessActionContext := &context.BusinessActionContext{
		XID:           xid,
		BranchID:      strconv.FormatInt(branchID, 10),
		ActionName:    resourceID,
		ActionContext: actionContextMap,
	}
	return businessActionContext
}

func (resourceManager TCCResourceManager) handleBranchCommit() {
	for {
		rpcRMMessage := <-resourceManager.RpcClient.BranchCommitRequestChannel
		rpcMessage := rpcRMMessage.RpcMessage
		serviceAddress := rpcRMMessage.ServerAddress

		req := rpcMessage.Body.(protocal.BranchCommitRequest)
		resp := resourceManager.doBranchCommit(req)
		resourceManager.RpcClient.SendResponse(rpcMessage, serviceAddress, resp)
	}
}

func (resourceManager TCCResourceManager) handleBranchRollback() {
	for {
		rpcRMMessage := <-resourceManager.RpcClient.BranchRollbackRequestChannel
		rpcMessage := rpcRMMessage.RpcMessage
		serviceAddress := rpcRMMessage.ServerAddress

		req := rpcMessage.Body.(protocal.BranchRollbackRequest)
		resp := resourceManager.doBranchRollback(req)
		resourceManager.RpcClient.SendResponse(rpcMessage, serviceAddress, resp)
	}
}

func (resourceManager TCCResourceManager) doBranchCommit(request protocal.BranchCommitRequest) protocal.BranchCommitResponse {
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

func (resourceManager TCCResourceManager) doBranchRollback(request protocal.BranchRollbackRequest) protocal.BranchRollbackResponse {
	var resp = protocal.BranchRollbackResponse{}

	log.Infof("Branch rolling back: %s %d %s", request.XID, request.BranchID, request.ResourceID)
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
