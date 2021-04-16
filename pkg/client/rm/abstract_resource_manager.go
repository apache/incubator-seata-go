package rm

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/base/model"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/client/context"
	"github.com/transaction-wg/seata-golang/pkg/client/rpc_client"
)

var (
	DBKEYS_SPLIT_CHAR = ","
)

type AbstractResourceManager struct {
	RpcClient     *rpc_client.RpcRemoteClient
	ResourceCache map[string]model.IResource
}

func NewAbstractResourceManager(client *rpc_client.RpcRemoteClient) AbstractResourceManager {
	resourceManager := AbstractResourceManager{
		RpcClient:     client,
		ResourceCache: make(map[string]model.IResource),
	}
	go resourceManager.handleRegisterRM()
	return resourceManager
}

func (resourceManager AbstractResourceManager) RegisterResource(resource model.IResource) {
	resourceManager.ResourceCache[resource.GetResourceID()] = resource
}

func (resourceManager AbstractResourceManager) UnregisterResource(resource model.IResource) {

}

func (resourceManager AbstractResourceManager) GetManagedResources() map[string]model.IResource {
	return resourceManager.ResourceCache
}

func (resourceManager AbstractResourceManager) BranchRegister(branchType meta.BranchType, resourceID string,
	clientID string, xid string, applicationData []byte, lockKeys string) (int64, error) {
	request := protocal.BranchRegisterRequest{
		XID:             xid,
		BranchType:      branchType,
		ResourceID:      resourceID,
		LockKey:         lockKeys,
		ApplicationData: applicationData,
	}
	resp, err := resourceManager.RpcClient.SendMsgWithResponse(request)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	response := resp.(protocal.BranchRegisterResponse)
	if response.ResultCode == protocal.ResultCodeSuccess {
		return response.BranchID, nil
	} else {
		return 0, response.GetError()
	}
}

func (resourceManager AbstractResourceManager) BranchReport(branchType meta.BranchType, xid string, branchID int64,
	status meta.BranchStatus, applicationData []byte) error {
	request := protocal.BranchReportRequest{
		XID:             xid,
		BranchID:        branchID,
		Status:          status,
		ApplicationData: applicationData,
	}
	resp, err := resourceManager.RpcClient.SendMsgWithResponse(request)
	if err != nil {
		return errors.WithStack(err)
	}
	response := resp.(protocal.BranchReportResponse)
	if response.ResultCode == protocal.ResultCodeFailed {
		return response.GetError()
	}
	return nil
}

func (resourceManager AbstractResourceManager) LockQuery(ctx *context.RootContext, branchType meta.BranchType, resourceID string, xid string,
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
	cfg := resourceManager.RpcClient.GetConf()
	message := protocal.RegisterRMRequest{
		AbstractIdentifyRequest: protocal.AbstractIdentifyRequest{
			Version:                 cfg.SeataVersion,
			ApplicationID:           cfg.ApplicationID,
			TransactionServiceGroup: cfg.TransactionServiceGroup,
		},
		ResourceIDs: resourceManager.getMergedResourceKeys(),
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
