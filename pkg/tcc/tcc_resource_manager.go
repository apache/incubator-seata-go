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
	"github.com/transaction-wg/seata-golang/base/meta"
	"github.com/transaction-wg/seata-golang/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/client"
	"github.com/transaction-wg/seata-golang/pkg/context"
	"github.com/transaction-wg/seata-golang/pkg/proxy"
	"github.com/transaction-wg/seata-golang/pkg/rm"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

var (
	TCC_ACTION_CONTEXT = "actionContext"
)

var tccResourceManager TCCResourceManager

func InitTCCResourceManager() {
	tccResourceManager = TCCResourceManager{
		AbstractResourceManager: rm.NewAbstractResourceManager(client.GetRpcRemoteClient()),
	}
	go tccResourceManager.handleBranchCommit()
	go tccResourceManager.handleBranchRollback()
}

type TCCResourceManager struct {
	rm.AbstractResourceManager
}

func (resourceManager TCCResourceManager) BranchCommit(branchType meta.BranchType, xid string, branchId int64,
	resourceId string, applicationData []byte) (meta.BranchStatus, error) {
	resource := resourceManager.ResourceCache[resourceId]
	if resource == nil {
		log.Errorf("TCC resource is not exist, resourceId: %s", resourceId)
		return 0, errors.Errorf("TCC resource is not exist, resourceId: %s", resourceId)
	}
	tccResource := resource.(*TCCResource)
	if tccResource.CommitMethod == nil {
		log.Errorf("TCC resource is not available, resourceId: %s", resourceId)
		return 0, errors.Errorf("TCC resource is not available, resourceId: %s", resourceId)
	}

	result := false
	businessActionContext := getBusinessActionContext(xid, branchId, resourceId, applicationData)
	args := make([]interface{}, 0)
	args = append(args, businessActionContext)
	returnValues := proxy.Invoke(tccResource.CommitMethod, nil, args)
	log.Infof("TCC resource commit result : %v, xid: %s, branchId: %d, resourceId: %s", returnValues, xid, branchId, resourceId)
	if returnValues != nil && len(returnValues) == 1 {
		result = returnValues[0].Interface().(bool)
	}
	if result {
		return meta.BranchStatusPhasetwoCommitted, nil
	} else {
		return meta.BranchStatusPhasetwoCommitFailedRetryable, nil
	}
}

func (resourceManager TCCResourceManager) BranchRollback(branchType meta.BranchType, xid string, branchId int64,
	resourceId string, applicationData []byte) (meta.BranchStatus, error) {
	resource := resourceManager.ResourceCache[resourceId]
	if resource == nil {
		return 0, errors.Errorf("TCC resource is not exist, resourceId: %s", resourceId)
	}
	tccResource := resource.(*TCCResource)
	if tccResource.RollbackMethod == nil {
		return 0, errors.Errorf("TCC resource is not available, resourceId: %s", resourceId)
	}

	result := false
	businessActionContext := getBusinessActionContext(xid, branchId, resourceId, applicationData)
	args := make([]interface{}, 0)
	args = append(args, businessActionContext)
	returnValues := proxy.Invoke(tccResource.RollbackMethod, nil, args)
	log.Infof("TCC resource rollback result : %v, xid: %s, branchId: %d, resourceId: %s", returnValues, xid, branchId, resourceId)
	if returnValues != nil && len(returnValues) == 1 {
		result = returnValues[0].Interface().(bool)
	}
	if result {
		return meta.BranchStatusPhasetwoRollbacked, nil
	} else {
		return meta.BranchStatusPhasetwoRollbackFailedRetryable, nil
	}
}

func (resourceManager TCCResourceManager) GetBranchType() meta.BranchType {
	return meta.BranchTypeTCC
}

func getBusinessActionContext(xid string, branchId int64, resourceId string, applicationData []byte) *context.BusinessActionContext {
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
		Xid:           xid,
		BranchId:      strconv.FormatInt(branchId, 10),
		ActionName:    resourceId,
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

	log.Infof("Branch committing: %s %d %s %s", request.Xid, request.BranchId, request.ResourceId, request.ApplicationData)
	status, err := resourceManager.BranchCommit(request.BranchType, request.Xid, request.BranchId, request.ResourceId, request.ApplicationData)
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
	resp.Xid = request.Xid
	resp.BranchId = request.BranchId
	resp.BranchStatus = status
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (resourceManager TCCResourceManager) doBranchRollback(request protocal.BranchRollbackRequest) protocal.BranchRollbackResponse {
	var resp = protocal.BranchRollbackResponse{}

	log.Infof("Branch rollbacking: %s %d %s", request.Xid, request.BranchId, request.ResourceId)
	status, err := resourceManager.BranchRollback(request.BranchType, request.Xid, request.BranchId, request.ResourceId, request.ApplicationData)
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
	resp.Xid = request.Xid
	resp.BranchId = request.BranchId
	resp.BranchStatus = status
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}
