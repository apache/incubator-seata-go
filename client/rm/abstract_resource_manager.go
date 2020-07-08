package rm

import (
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/base/model"
	"github.com/dk-lockdown/seata-golang/base/protocal"
	"github.com/dk-lockdown/seata-golang/client/config"
	"github.com/dk-lockdown/seata-golang/client/context"
	"github.com/dk-lockdown/seata-golang/client/getty"
)

var (
	DBKEYS_SPLIT_CHAR = ","
)

type AbstractResourceManager struct {
	RpcClient     *getty.RpcRemoteClient
	ResourceCache map[string]model.IResource
}

func NewAbstractResourceManager(client *getty.RpcRemoteClient) AbstractResourceManager {
	resourceManager := AbstractResourceManager{
		RpcClient:     client,
		ResourceCache: make(map[string]model.IResource),
	}
	go resourceManager.handleRegisterRM()
	return resourceManager
}

func (resourceManager AbstractResourceManager) RegisterResource(resource model.IResource) {
	resourceManager.ResourceCache[resource.GetResourceId()] = resource
}

func (resourceManager AbstractResourceManager) UnregisterResource(resource model.IResource) {

}

func (resourceManager AbstractResourceManager) GetManagedResources() map[string]model.IResource {
	return resourceManager.ResourceCache
}

func (resourceManager AbstractResourceManager) BranchRegister(branchType meta.BranchType, resourceId string,
	clientId string, xid string, applicationData []byte, lockKeys string) (int64, error) {
	request := protocal.BranchRegisterRequest{
		Xid:             xid,
		BranchType:      branchType,
		ResourceId:      resourceId,
		LockKey:         lockKeys,
		ApplicationData: applicationData,
	}
	resp, err := resourceManager.RpcClient.SendMsgWithResponse(request)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	response := resp.(protocal.BranchRegisterResponse)
	if response.ResultCode == protocal.ResultCodeSuccess {
		return response.BranchId, nil
	} else {
		if response.TransactionExceptionCode == meta.TransactionExceptionCodeGlobalTransactionNotExist {
			return 0, &meta.TransactionException{
				Message: response.Msg,
				Code:    meta.TransactionExceptionCodeGlobalTransactionNotExist,
			}
		}
		return 0, errors.New(response.Msg)
	}
}

func (resourceManager AbstractResourceManager) BranchReport(branchType meta.BranchType, xid string, branchId int64,
	status meta.BranchStatus, applicationData []byte) error {
	request := protocal.BranchReportRequest{
		Xid:             xid,
		BranchId:        branchId,
		Status:          status,
		ApplicationData: applicationData,
	}
	resp, err := resourceManager.RpcClient.SendMsgWithResponse(request)
	if err != nil {
		return errors.WithStack(err)
	}
	response := resp.(protocal.BranchReportResponse)
	if response.ResultCode == protocal.ResultCodeFailed {
		return errors.Errorf("Response[ %s ]", response.Msg)
	}
	return nil
}

func (resourceManager AbstractResourceManager) LockQuery(ctx *context.RootContext, branchType meta.BranchType, resourceId string, xid string,
	lockKeys string) (bool, error) {
	return false, nil
}

func (resourceManager AbstractResourceManager) handleRegisterRM() {
	for {
		serverAddress := <-resourceManager.RpcClient.GettySessionOnOpenChannel
		resourceManager.doRegisterResource(serverAddress)
	}
}

func (resourceManager AbstractResourceManager) doRegisterResource(serverAddress string) {
	if resourceManager.ResourceCache == nil || len(resourceManager.ResourceCache) == 0 {
		return
	}
	message := protocal.RegisterRMRequest{
		AbstractIdentifyRequest: protocal.AbstractIdentifyRequest{
			Version:                 config.GetClientConfig().SeataVersion,
			ApplicationId:           config.GetClientConfig().ApplicationId,
			TransactionServiceGroup: config.GetClientConfig().TransactionServiceGroup,
		},
		ResourceIds: resourceManager.getMergedResourceKeys(),
	}

	resourceManager.RpcClient.RegisterResource(serverAddress, message)
}

func (resourceManager AbstractResourceManager) getMergedResourceKeys() string {
	var builder strings.Builder
	if resourceManager.ResourceCache != nil && len(resourceManager.ResourceCache) > 0 {
		for key, _ := range resourceManager.ResourceCache {
			builder.WriteString(key)
			builder.WriteString(DBKEYS_SPLIT_CHAR)
		}
		resourceKeys := builder.String()
		resourceKeys = resourceKeys[:len(resourceKeys)-1]
		return resourceKeys
	}
	return ""
}
