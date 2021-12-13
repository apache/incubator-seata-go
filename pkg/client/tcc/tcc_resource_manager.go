package tcc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	ctx "github.com/opentrx/seata-golang/v2/pkg/client/base/context"
	"github.com/opentrx/seata-golang/v2/pkg/client/base/model"
	"github.com/opentrx/seata-golang/v2/pkg/client/proxy"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
)

var (
	TccActionContext = "actionContext"
)

var tccResourceManager TCCResourceManager

type TCCResourceManager struct {
	ResourceCache map[string]model.Resource
}

func init() {
	tccResourceManager = TCCResourceManager{ResourceCache: make(map[string]model.Resource)}
}

func GetTCCResourceManager() TCCResourceManager {
	return tccResourceManager
}

func (resourceManager TCCResourceManager) BranchCommit(ctx context.Context, request *apis.BranchCommitRequest) (*apis.BranchCommitResponse, error) {
	resource := resourceManager.ResourceCache[request.ResourceID]
	if resource == nil {
		log.Errorf("TCC resource is not exist, resourceID: %s", request.ResourceID)
		return &apis.BranchCommitResponse{
			ResultCode: apis.ResultCodeFailed,
			Message:    fmt.Sprintf("TCC resource is not exist, resourceID: %s", request.ResourceID),
		}, nil
	}
	tccResource := resource.(*TCCResource)
	if tccResource.CommitMethod == nil {
		log.Errorf("TCC resource is not available, resourceID: %s", request.ResourceID)
		return &apis.BranchCommitResponse{
			ResultCode: apis.ResultCodeFailed,
			Message:    fmt.Sprintf("TCC resource is not available, resourceID: %s", request.ResourceID),
		}, nil
	}

	result := false
	businessActionContext := getBusinessActionContext(request.XID, request.BranchID, request.ResourceID, request.ApplicationData)
	args := make([]interface{}, 0)
	args = append(args, businessActionContext)
	returnValues := proxy.Invoke(tccResource.CommitMethod, nil, args)
	log.Debugf("TCC resource commit result : %v, xid: %s, branchID: %d, resourceID: %s", returnValues, request.XID, request.BranchID, request.ResourceID)
	if len(returnValues) == 1 {
		result = returnValues[0].Interface().(bool)
	}
	if result {
		return &apis.BranchCommitResponse{
			ResultCode:   apis.ResultCodeSuccess,
			XID:          request.XID,
			BranchID:     request.BranchID,
			BranchStatus: apis.PhaseTwoCommitted,
		}, nil
	}
	return &apis.BranchCommitResponse{
		ResultCode:   apis.ResultCodeSuccess,
		XID:          request.XID,
		BranchID:     request.BranchID,
		BranchStatus: apis.PhaseTwoCommitFailedRetryable,
	}, nil
}

func (resourceManager TCCResourceManager) BranchRollback(ctx context.Context, request *apis.BranchRollbackRequest) (*apis.BranchRollbackResponse, error) {
	resource := resourceManager.ResourceCache[request.ResourceID]
	if resource == nil {
		return &apis.BranchRollbackResponse{
			ResultCode: apis.ResultCodeFailed,
			Message:    fmt.Sprintf("TCC resource is not exist, resourceID: %s", request.ResourceID),
		}, nil
	}
	tccResource := resource.(*TCCResource)
	if tccResource.RollbackMethod == nil {
		return &apis.BranchRollbackResponse{
			ResultCode: apis.ResultCodeFailed,
			Message:    fmt.Sprintf("TCC resource is not available, resourceID: %s", request.ResourceID),
		}, nil
	}

	result := false
	businessActionContext := getBusinessActionContext(request.XID, request.BranchID, request.ResourceID, request.ApplicationData)
	args := make([]interface{}, 0)
	args = append(args, businessActionContext)
	returnValues := proxy.Invoke(tccResource.RollbackMethod, nil, args)
	log.Debugf("TCC resource rollback result : %v, xid: %s, branchID: %d, resourceID: %s", returnValues, request.XID, request.BranchID, request.ResourceID)
	if len(returnValues) == 1 {
		result = returnValues[0].Interface().(bool)
	}
	if result {
		return &apis.BranchRollbackResponse{
			ResultCode:   apis.ResultCodeSuccess,
			XID:          request.XID,
			BranchID:     request.BranchID,
			BranchStatus: apis.PhaseTwoRolledBack,
		}, nil
	}
	return &apis.BranchRollbackResponse{
		ResultCode:   apis.ResultCodeSuccess,
		XID:          request.XID,
		BranchID:     request.BranchID,
		BranchStatus: apis.PhaseTwoRollbackFailedRetryable,
	}, nil
}

func getBusinessActionContext(xid string, branchID int64, resourceID string, applicationData []byte) *ctx.BusinessActionContext {
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

	acMap := tccContext[TccActionContext]
	if acMap != nil {
		actionContextMap = acMap.(map[string]interface{})
	}

	businessActionContext := &ctx.BusinessActionContext{
		XID:           xid,
		BranchID:      branchID,
		ActionName:    resourceID,
		ActionContext: actionContextMap,
	}
	return businessActionContext
}

func (resourceManager TCCResourceManager) RegisterResource(resource model.Resource) {
	resourceManager.ResourceCache[resource.GetResourceID()] = resource
}

func (resourceManager TCCResourceManager) UnregisterResource(resource model.Resource) {
	delete(resourceManager.ResourceCache, resource.GetResourceID())
}

func (resourceManager TCCResourceManager) GetBranchType() apis.BranchSession_BranchType {
	return apis.TCC
}
