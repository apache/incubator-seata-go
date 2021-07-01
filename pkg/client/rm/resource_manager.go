package rm

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/types"
	"go.uber.org/atomic"
	"sync"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/client/base/exception"
	"github.com/opentrx/seata-golang/v2/pkg/client/base/model"
)

var defaultResourceManager *ResourceManager

type ResourceManagerOutbound interface {
	// Branch register long.
	BranchRegister(ctx context.Context, xid string, resourceID string, branchType apis.BranchSession_BranchType,
		applicationData []byte, lockKeys string) (int64, error)

	// Branch report.
	BranchReport(ctx context.Context, xid string, branchID int64, branchType apis.BranchSession_BranchType,
		status apis.BranchSession_BranchStatus, applicationData []byte) error

	// Lock query boolean.
	LockQuery(ctx context.Context, xid string, resourceID string, branchType apis.BranchSession_BranchType, lockKeys string) (bool, error)
}

type ResourceManagerInterface interface {
	BranchCommit(ctx context.Context, request *apis.BranchCommitRequest) (*apis.BranchCommitResponse, error)

	BranchRollback(ctx context.Context, request *apis.BranchRollbackRequest) (*apis.BranchRollbackResponse, error)

	// Register a Resource to be managed by Resource Manager.
	RegisterResource(resource model.Resource)

	// Unregister a Resource from the Resource Manager.
	UnregisterResource(resource model.Resource)

	// Get the BranchType.
	GetBranchType() apis.BranchSession_BranchType
}

type ResourceManager struct {
	addressing string
	rpcClient  apis.ResourceManagerServiceClient
	managers   map[apis.BranchSession_BranchType]ResourceManagerInterface

	idGenerator atomic.Uint64
	futures     *sync.Map
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
	stream, err := manager.rpcClient.BranchCommunicate(context.Background())
	if err != nil {
		return 0, err
	}

	content, err := types.MarshalAny(request)
	if err != nil {
		return 0, err
	}

	stream.Send(&apis.BranchMessage{
		ID:                int64(manager.idGenerator.Inc()),
		BranchMessageType: apis.TypeBranchRegister,
		Message:           content,
	})

	resp, err := manager.rpcClient.BranchRegister(ctx, request)
	if err != nil {
		return 0, err
	}
	if resp.ResultCode == apis.ResultCodeSuccess {
		return resp.BranchID, nil
	} else {
		return 0, &exception.TransactionException{
			Code:    resp.GetExceptionCode(),
			Message: resp.GetMessage(),
		}
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
	} else {
		return false, &exception.TransactionException{
			Code:    resp.GetExceptionCode(),
			Message: resp.GetMessage(),
		}
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
