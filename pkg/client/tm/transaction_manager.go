package tm

import (
	"context"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/client/base/exception"
)

var defaultTransactionManager *TransactionManager

type TransactionManagerInterface interface {
	// GlobalStatus_Begin a new global transaction.
	Begin(ctx context.Context, name string, timeout int32) (string, error)

	// Global commit.
	Commit(ctx context.Context, xid string) (apis.GlobalSession_GlobalStatus, error)

	// Global rollback.
	Rollback(ctx context.Context, xid string) (apis.GlobalSession_GlobalStatus, error)

	// Get current status of the give transaction.
	GetStatus(ctx context.Context, xid string) (apis.GlobalSession_GlobalStatus, error)

	// Global report.
	GlobalReport(ctx context.Context, xid string, globalStatus apis.GlobalSession_GlobalStatus) (apis.GlobalSession_GlobalStatus, error)
}

type TransactionManager struct {
	addressing string
	rpcClient  apis.TransactionManagerServiceClient
}

func InitTransactionManager(addressing string, client apis.TransactionManagerServiceClient) {
	defaultTransactionManager = &TransactionManager{
		addressing: addressing,
		rpcClient:  client,
	}
}

func GetTransactionManager() *TransactionManager {
	return defaultTransactionManager
}

func (manager *TransactionManager) Begin(ctx context.Context, name string, timeout int32) (string, error) {
	request := &apis.GlobalBeginRequest{
		Addressing:      manager.addressing,
		Timeout:         timeout,
		TransactionName: name,
	}
	resp, err := manager.rpcClient.Begin(ctx, request)
	if err != nil {
		return "", err
	}
	if resp.ResultCode == apis.ResultCodeSuccess {
		return resp.XID, nil
	}
	return "", &exception.TransactionException{
		Code:    resp.GetExceptionCode(),
		Message: resp.GetMessage(),
	}
}

func (manager *TransactionManager) Commit(ctx context.Context, xid string) (apis.GlobalSession_GlobalStatus, error) {
	request := &apis.GlobalCommitRequest{XID: xid}
	resp, err := manager.rpcClient.Commit(ctx, request)
	if err != nil {
		return 0, err
	}
	if resp.ResultCode == apis.ResultCodeSuccess {
		return resp.GlobalStatus, nil
	}
	return 0, &exception.TransactionException{
		Code:    resp.GetExceptionCode(),
		Message: resp.GetMessage(),
	}
}

func (manager *TransactionManager) Rollback(ctx context.Context, xid string) (apis.GlobalSession_GlobalStatus, error) {
	request := &apis.GlobalRollbackRequest{XID: xid}
	resp, err := manager.rpcClient.Rollback(ctx, request)
	if err != nil {
		return 0, err
	}
	if resp.ResultCode == apis.ResultCodeSuccess {
		return resp.GlobalStatus, nil
	}
	return 0, &exception.TransactionException{
		Code:    resp.GetExceptionCode(),
		Message: resp.GetMessage(),
	}
}

func (manager *TransactionManager) GetStatus(ctx context.Context, xid string) (apis.GlobalSession_GlobalStatus, error) {
	request := &apis.GlobalStatusRequest{XID: xid}
	resp, err := manager.rpcClient.GetStatus(context.Background(), request)
	if err != nil {
		return 0, err
	}
	if resp.ResultCode == apis.ResultCodeSuccess {
		return resp.GlobalStatus, nil
	}
	return 0, &exception.TransactionException{
		Code:    resp.GetExceptionCode(),
		Message: resp.GetMessage(),
	}
}

func (manager *TransactionManager) GlobalReport(ctx context.Context, xid string, globalStatus apis.GlobalSession_GlobalStatus) (apis.GlobalSession_GlobalStatus, error) {
	request := &apis.GlobalReportRequest{
		XID:          xid,
		GlobalStatus: globalStatus,
	}
	resp, err := manager.rpcClient.GlobalReport(ctx, request)
	if err != nil {
		return 0, err
	}
	if resp.ResultCode == apis.ResultCodeSuccess {
		return resp.GlobalStatus, nil
	}
	return 0, &exception.TransactionException{
		Code:    resp.GetExceptionCode(),
		Message: resp.GetMessage(),
	}
}
