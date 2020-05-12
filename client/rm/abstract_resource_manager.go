package rm

import (
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/base/protocal"
	"github.com/dk-lockdown/seata-golang/client/getty"
)

type AbstractResourceManager struct {
	RpcClient *getty.RpcRemoteClient
}

func NewAbstractResourceManager(client *getty.RpcRemoteClient) AbstractResourceManager {
	return AbstractResourceManager{RpcClient: client}
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
	resp,err := resourceManager.RpcClient.SendMsgWithResponse(request)
	if err != nil {
		return 0,errors.WithStack(err)
	}
	response := resp.(protocal.BranchRegisterResponse)
	return response.BranchId,nil
}


func (resourceManager AbstractResourceManager) BranchReport(branchType meta.BranchType, xid string, branchId int64,
	status meta.BranchStatus, applicationData []byte) error {
	request := protocal.BranchReportRequest{
		Xid:             xid,
		BranchId:        branchId,
		Status:          status,
		ApplicationData: applicationData,
	}
	resp,err := resourceManager.RpcClient.SendMsgWithResponse(request)
	if err != nil {
		return errors.WithStack(err)
	}
	response := resp.(protocal.BranchReportResponse)
	if response.ResultCode == protocal.ResultCodeFailed {
		return errors.Errorf("Response[ %s ]", response.Msg)
	}
	return nil
}

func (resourceManager AbstractResourceManager) LockQuery(branchType meta.BranchType, resourceId string, xid string,
	lockKeys string) (bool, error) {
	return false,nil
}