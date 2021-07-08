package rm

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/types"
	common2 "github.com/opentrx/seata-golang/v2/pkg/common"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
	"github.com/opentrx/seata-golang/v2/pkg/util/runtime"
	"go.uber.org/atomic"
	"io"
	"sync"
	"time"

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

	idGenerator    *atomic.Uint64
	branchMessages chan *apis.BranchMessage
	futures        *sync.Map
}

func InitResourceManager(addressing string, client apis.ResourceManagerServiceClient) {
	defaultResourceManager = &ResourceManager{
		addressing:     addressing,
		rpcClient:      client,
		managers:       make(map[apis.BranchSession_BranchType]ResourceManagerInterface),
		idGenerator:    &atomic.Uint64{},
		branchMessages: make(chan *apis.BranchMessage, 1000),
		futures:        &sync.Map{},
	}
	runtime.GoWithRecover(func() {
		defaultResourceManager.branchCommunicate()
	}, nil)
}

func RegisterTransactionServiceServer(rm ResourceManagerInterface) {
	defaultResourceManager.managers[rm.GetBranchType()] = rm
}

func GetResourceManager() *ResourceManager {
	return defaultResourceManager
}

func (manager *ResourceManager) branchCommunicate() {
	for {
		stream, err := manager.rpcClient.BranchCommunicate(context.Background())
		if err != nil {
			continue
		}
		register := &apis.BranchStreamRegisterMessage{Addressing: manager.addressing}
		data, err := types.MarshalAny(register)
		if err != nil {
			log.Error(err)
			continue
		}
		err = stream.Send(&apis.BranchMessage{
			ID:                0,
			BranchMessageType: apis.TypeBranchStreamRegister,
			Message:           data,
		})
		if err != nil {
			continue
		}
		done := make(chan bool)
		runtime.GoWithRecover(func() {
			for {
				select {
				case <- done:
					return
				case msg := <- manager.branchMessages:
					err := stream.Send(msg)
					if err != nil {
						resp, loaded := manager.futures.Load(msg.ID)
						if loaded {
							future := resp.(*common2.MessageFuture)
							future.Err = err
							future.Done <- true
							manager.futures.Delete(msg.ID)
						}
						return
					}
				default:
					continue
				}
			}
		}, nil)

		withResponse := func (manager *ResourceManager, msgID int64, response interface{}) {
			resp, loaded := manager.futures.Load(msgID)
			if loaded {
				future := resp.(*common2.MessageFuture)
				future.Response = response
				future.Done <- true
				manager.futures.Delete(msgID)
			}
		}
FOR:
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				done <- true
				break
			}
			if err != nil {
				done <- true
				break
			}
			switch msg.BranchMessageType {
			case apis.TypeBranchRegisterResult:
				response := &apis.BranchRegisterResponse{}
				data := msg.GetMessage().GetValue()
				err := response.Unmarshal(data)
				if err != nil {
					log.Error(err)
					continue
				}
				withResponse(manager, msg.ID, response)
			case apis.TypeBranchReportResult:
				response := &apis.BranchReportResponse{}
				data := msg.GetMessage().GetValue()
				err := response.Unmarshal(data)
				if err != nil {
					log.Error(err)
					continue
				}
				withResponse(manager, msg.ID, response)
			case apis.TypeBranchCommit:
				request := &apis.BranchCommitRequest{}
				data := msg.GetMessage().GetValue()
				err := request.Unmarshal(data)
				if err != nil {
					log.Error(err)
					continue
				}
				response, err := manager.BranchCommit(context.Background(), request)
				if err == nil {
					content, err := types.MarshalAny(response)
					if err == nil {
						err := stream.Send(&apis.BranchMessage{
							ID:                msg.ID,
							BranchMessageType: apis.TypeBranchCommitResult,
							Message:           content,
						})
						if err != nil {
							done <- true
							break FOR
						}
					}
				}
			case apis.TypeBranchRollback:
				request := &apis.BranchRollbackRequest{}
				data := msg.GetMessage().GetValue()
				err := request.Unmarshal(data)
				if err != nil {
					log.Error(err)
					continue
				}
				response, err := manager.BranchRollback(context.Background(), request)
				if err == nil {
					content, err := types.MarshalAny(response)
					if err == nil {
						err := stream.Send(&apis.BranchMessage{
							ID:                msg.ID,
							BranchMessageType: apis.TypeBranchRollBackResult,
							Message:           content,
						})
						if err != nil {
							done <- true
							break FOR
						}
					}
				}
			}
		}
		stream.CloseSend()
	}
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

	content, err := types.MarshalAny(request)
	if err != nil {
		return 0, err
	}

	message := &apis.BranchMessage{
		ID:                int64(manager.idGenerator.Inc()),
		BranchMessageType: apis.TypeBranchRegister,
		Message:           content,
	}

	manager.branchMessages <- message

	resp := common2.NewMessageFuture(message)
	manager.futures.Store(message.ID, resp)

	timer := time.NewTimer(5*time.Second)
	select {
	case <- timer.C:
		manager.futures.Delete(resp.ID)
		timer.Stop()
		return 0, fmt.Errorf("timeout")
	case <- resp.Done:
		timer.Stop()
	}

	if resp.Err != nil {
		return 0, resp.Err
	}

	response := resp.Response.(*apis.BranchRegisterResponse)
	if response.ResultCode == apis.ResultCodeSuccess {
		return response.BranchID, nil
	} else {
		return 0, &exception.TransactionException{
			Code:    response.GetExceptionCode(),
			Message: response.GetMessage(),
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

	content, err := types.MarshalAny(request)
	if err != nil {
		return err
	}

	message := &apis.BranchMessage{
		ID:                int64(manager.idGenerator.Inc()),
		BranchMessageType: apis.TypeBranchRegister,
		Message:           content,
	}

	manager.branchMessages <- message

	resp := common2.NewMessageFuture(message)
	manager.futures.Store(message.ID, resp)

	timer := time.NewTimer(5*time.Second)
	select {
	case <- timer.C:
		manager.futures.Delete(resp.ID)
		timer.Stop()
		return fmt.Errorf("timeout")
	case <- resp.Done:
		timer.Stop()
	}

	if resp.Err != nil {
		return resp.Err
	}

	response := resp.Response.(*apis.BranchReportResponse)
	if response.ResultCode == apis.ResultCodeFailed {
		return &exception.TransactionException{
			Code:    response.GetExceptionCode(),
			Message: response.GetMessage(),
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
