package rm

import (
	"context"
	"fmt"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/client/base/exception"
	"github.com/opentrx/seata-golang/v2/pkg/client/base/model"
)

var defaultResourceManager *ResourceManager

type ResourceManagerOutbound interface {
	// BranchRegister register branch transaction.
	BranchRegister(ctx context.Context, xid string, resourceID string, branchType apis.BranchSession_BranchType,
		applicationData []byte, lockKeys string) (int64, error)

	// BranchReport report branch transaction status.
	BranchReport(ctx context.Context, xid string, branchID int64, branchType apis.BranchSession_BranchType,
		status apis.BranchSession_BranchStatus, applicationData []byte) error

	// LockQuery lock resource by lockKeys.
	LockQuery(ctx context.Context, xid string, resourceID string, branchType apis.BranchSession_BranchType, lockKeys string) (bool, error)
}

type ResourceManagerInterface interface {
	apis.BranchTransactionServiceServer

	// RegisterResource Register a Resource to be managed by Resource Manager.
	RegisterResource(resource model.Resource)

	// UnregisterResource Unregister a Resource from the Resource Manager.
	UnregisterResource(resource model.Resource)

	// GetBranchType ...
	GetBranchType() apis.BranchSession_BranchType
}

type ResourceManager struct {
	addressing string
	rpcClient  apis.ResourceManagerServiceClient
	managers   map[apis.BranchSession_BranchType]ResourceManagerInterface
}

func InitResourceManager(addressing string, client apis.ResourceManagerServiceClient) {
	defaultResourceManager = &ResourceManager{
		addressing: addressing,
		rpcClient:  client,
		managers:   make(map[apis.BranchSession_BranchType]ResourceManagerInterface),
	}
}

func RegisterTransactionServiceServer(rm ResourceManagerInterface) {
	defaultResourceManager.managers[rm.GetBranchType()] = rm
}

func GetResourceManager() *ResourceManager {
	return defaultResourceManager
}

func (manager *ResourceManager) BranchRegister(ctx context.Context, xid string, resourceID string,
	branchType apis.BranchSession_BranchType, applicationData []byte, lockKeys string) (int64, error) {
	request := &apis.BranchRegisterRequest{
		Addressing:      manager.addressing,
		XID:             xid,
		ResourceID:      resourceID,
		LockKey:         lockKeys,
		BranchType:      branchType,
		ApplicationData: applicationData,
	}
	resp, err := manager.rpcClient.BranchRegister(ctx, request)
	if err != nil {
		return 0, err
	}
	if resp.ResultCode == apis.ResultCodeSuccess {
		return resp.BranchID, nil
	}
	return 0, &exception.TransactionException{
		Code:    resp.GetExceptionCode(),
		Message: resp.GetMessage(),
	}
}

func (manager *ResourceManager) BranchReport(ctx context.Context, xid string, branchID int64,
	branchType apis.BranchSession_BranchType, status apis.BranchSession_BranchStatus, applicationData []byte) error {
	request := &apis.BranchReportRequest{
		XID:             xid,
		BranchID:        branchID,
		BranchType:      branchType,
		BranchStatus:    status,
		ApplicationData: applicationData,
	}
	resp, err := manager.rpcClient.BranchReport(ctx, request)
	if err != nil {
		return err
	}
	if resp.ResultCode == apis.ResultCodeFailed {
		return &exception.TransactionException{
			Code:    resp.GetExceptionCode(),
			Message: resp.GetMessage(),
		}
	}
	return nil
}

func (manager *ResourceManager) LockQuery(ctx context.Context, xid string, resourceID string, branchType apis.BranchSession_BranchType,
	lockKeys string) (bool, error) {
	request := &apis.GlobalLockQueryRequest{
		XID:        xid,
		ResourceID: resourceID,
		LockKey:    lockKeys,
		BranchType: branchType,
	}

	resp, err := manager.rpcClient.LockQuery(ctx, request)
	if err != nil {
		return false, err
	}
	if resp.ResultCode == apis.ResultCodeSuccess {
		return resp.Lockable, nil
	}
	return false, &exception.TransactionException{
		Code:    resp.GetExceptionCode(),
		Message: resp.GetMessage(),
	}
}

func (manager ResourceManager) BranchCommit(ctx context.Context, request *apis.BranchCommitRequest) (*apis.BranchCommitResponse, error) {
	rm, ok := manager.managers[request.BranchType]
	if ok {
		return rm.BranchCommit(ctx, request)
	}
	return &apis.BranchCommitResponse{
		ResultCode: apis.ResultCodeFailed,
		Message:    fmt.Sprintf("there is no resource manager for %s", request.BranchType.String()),
	}, nil
}

func (manager *ResourceManager) BranchRollback(ctx context.Context, request *apis.BranchRollbackRequest) (*apis.BranchRollbackResponse, error) {
	rm, ok := manager.managers[request.BranchType]
	if ok {
		return rm.BranchRollback(ctx, request)
	}
	return &apis.BranchRollbackResponse{
		ResultCode: apis.ResultCodeFailed,
		Message:    fmt.Sprintf("there is no resource manager for %s", request.BranchType.String()),
	}, nil
}

func (manager *ResourceManager) RegisterResource(resource model.Resource) {
	rm := manager.managers[resource.GetBranchType()]
	rm.RegisterResource(resource)
}

func (manager *ResourceManager) UnregisterResource(resource model.Resource) {
	rm := manager.managers[resource.GetBranchType()]
	rm.UnregisterResource(resource)
}
