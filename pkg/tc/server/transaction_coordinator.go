package server

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/tc/config"
	"github.com/opentrx/seata-golang/v2/pkg/tc/event"
	"github.com/opentrx/seata-golang/v2/pkg/tc/holder"
	"github.com/opentrx/seata-golang/v2/pkg/tc/lock"
	"github.com/opentrx/seata-golang/v2/pkg/tc/model"
	"github.com/opentrx/seata-golang/v2/pkg/tc/storage/driver/factory"
	"github.com/opentrx/seata-golang/v2/pkg/util/common"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
	"github.com/opentrx/seata-golang/v2/pkg/util/runtime"
	time2 "github.com/opentrx/seata-golang/v2/pkg/util/time"
	"github.com/opentrx/seata-golang/v2/pkg/util/uuid"
)

const AlwaysRetryBoundary = 0

type TransactionCoordinator struct {
	sync.Mutex
	maxCommitRetryTimeout            int64
	maxRollbackRetryTimeout          int64
	rollbackRetryTimeoutUnlockEnable bool

	asyncCommittingRetryPeriod time.Duration
	committingRetryPeriod      time.Duration
	rollingBackRetryPeriod     time.Duration
	timeoutRetryPeriod         time.Duration

	holder             *holder.SessionHolder
	resourceDataLocker *lock.Manager
	locker             GlobalSessionLocker

	keepaliveClientParameters keepalive.ClientParameters
	tcServiceClients          map[string]apis.BranchTransactionServiceClient
}

func NewTransactionCoordinator(conf *config.Configuration) *TransactionCoordinator {
	driver, err := factory.Create(conf.Storage.Type(), conf.Storage.Parameters())
	if err != nil {
		log.Fatalf("failed to construct %s driver: %v", conf.Storage.Type(), err)
		os.Exit(1)
	}
	tc := &TransactionCoordinator{
		maxCommitRetryTimeout:            conf.Server.MaxCommitRetryTimeout,
		maxRollbackRetryTimeout:          conf.Server.MaxRollbackRetryTimeout,
		rollbackRetryTimeoutUnlockEnable: conf.Server.RollbackRetryTimeoutUnlockEnable,

		asyncCommittingRetryPeriod: conf.Server.AsyncCommittingRetryPeriod,
		committingRetryPeriod:      conf.Server.CommittingRetryPeriod,
		rollingBackRetryPeriod:     conf.Server.RollingBackRetryPeriod,
		timeoutRetryPeriod:         conf.Server.TimeoutRetryPeriod,

		holder:             holder.NewSessionHolder(driver),
		resourceDataLocker: lock.NewLockManager(driver),
		locker:             new(UnimplementedGlobalSessionLocker),

		keepaliveClientParameters: conf.GetClientParameters(),
		tcServiceClients:          make(map[string]apis.BranchTransactionServiceClient),
	}
	go tc.processTimeoutCheck()
	go tc.processAsyncCommitting()
	go tc.processRetryCommitting()
	go tc.processRetryRollingBack()

	return tc
}

func (tc *TransactionCoordinator) Begin(ctx context.Context, request *apis.GlobalBeginRequest) (*apis.GlobalBeginResponse, error) {
	transactionID := uuid.NextID()
	xid := common.GenerateXID(request.Addressing, transactionID)
	gt := model.GlobalTransaction{
		GlobalSession: &apis.GlobalSession{
			Addressing:      request.Addressing,
			XID:             xid,
			TransactionID:   transactionID,
			TransactionName: request.TransactionName,
			Timeout:         request.Timeout,
		},
	}
	gt.Begin()
	err := tc.holder.AddGlobalSession(gt.GlobalSession)
	if err != nil {
		return &apis.GlobalBeginResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.BeginFailed,
			Message:       err.Error(),
		}, nil
	}

	runtime.GoWithRecover(func() {
		evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime, 0, gt.Status)
		event.Bus.GlobalTransactionEventChannel <- evt
	}, nil)

	log.Infof("successfully begin global transaction xid = {}", gt.XID)
	return &apis.GlobalBeginResponse{
		ResultCode: apis.ResultCodeSuccess,
		XID:        xid,
	}, nil
}

func (tc *TransactionCoordinator) GetStatus(ctx context.Context, request *apis.GlobalStatusRequest) (*apis.GlobalStatusResponse, error) {
	gs := tc.holder.FindGlobalSession(request.XID)
	if gs != nil {
		return &apis.GlobalStatusResponse{
			ResultCode:   apis.ResultCodeSuccess,
			GlobalStatus: gs.Status,
		}, nil
	}
	return &apis.GlobalStatusResponse{
		ResultCode:   apis.ResultCodeSuccess,
		GlobalStatus: apis.Finished,
	}, nil
}

func (tc *TransactionCoordinator) GlobalReport(ctx context.Context, request *apis.GlobalReportRequest) (*apis.GlobalReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GlobalReport not implemented")
}

func (tc *TransactionCoordinator) Commit(ctx context.Context, request *apis.GlobalCommitRequest) (*apis.GlobalCommitResponse, error) {
	gt := tc.holder.FindGlobalTransaction(request.XID)
	if gt == nil {
		return &apis.GlobalCommitResponse{
			ResultCode:   apis.ResultCodeSuccess,
			GlobalStatus: apis.Finished,
		}, nil
	}
	shouldCommit, err := func(gt *model.GlobalTransaction) (bool, error) {
		result, err := tc.locker.TryLock(gt.GlobalSession, time.Duration(gt.Timeout)*time.Millisecond)
		if err != nil {
			return false, err
		}
		if result {
			defer tc.locker.Unlock(gt.GlobalSession)
			if gt.Active {
				// Active need persistence
				// Highlight: Firstly, close the session, then no more branch can be registered.
				err = tc.holder.InactiveGlobalSession(gt.GlobalSession)
				if err != nil {
					return false, fmt.Errorf("InactiveGlobalSession: %+v", err)
				}
			}
			tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
			if gt.Status == apis.Begin {
				err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.Committing)
				if err != nil {
					return false, fmt.Errorf("UpdateGlobalSessionStatus: %+v", err)
				}
				return true, nil
			}
			return false, nil
		}
		return false, fmt.Errorf("failed to lock global transaction xid = %s", request.XID)
	}(gt)

	if err != nil {
		return &apis.GlobalCommitResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.FailedLockGlobalTransaction,
			Message:       err.Error(),
			GlobalStatus:  gt.Status,
		}, nil
	}

	if !shouldCommit {
		if gt.Status == apis.AsyncCommitting {
			return &apis.GlobalCommitResponse{
				ResultCode:   apis.ResultCodeSuccess,
				GlobalStatus: apis.Committed,
			}, nil
		}
		return &apis.GlobalCommitResponse{
			ResultCode:   apis.ResultCodeSuccess,
			GlobalStatus: gt.Status,
		}, nil
	}

	if gt.CanBeCommittedAsync() {
		err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.AsyncCommitting)

		return &apis.GlobalCommitResponse{
			ResultCode:   apis.ResultCodeSuccess,
			GlobalStatus: apis.Committed,
		}, err
	}
	_, err = tc.doGlobalCommit(gt, false)
	if err != nil {
		return &apis.GlobalCommitResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.UnknownErr,
			Message:       err.Error(),
			GlobalStatus:  gt.Status,
		}, nil
	}
	return &apis.GlobalCommitResponse{
		ResultCode:   apis.ResultCodeSuccess,
		GlobalStatus: apis.Committed,
	}, nil
}

func (tc *TransactionCoordinator) doGlobalCommit(gt *model.GlobalTransaction, retrying bool) (bool, error) {
	var err error

	runtime.GoWithRecover(func() {
		evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime, 0, gt.Status)
		event.Bus.GlobalTransactionEventChannel <- evt
	}, nil)

	if gt.IsSaga() {
		return false, status.Errorf(codes.Unimplemented, "method Commit not supported saga mode")
	}

	for bs := range gt.BranchSessions {
		if bs.Status == apis.PhaseOneFailed {
			tc.resourceDataLocker.ReleaseLock(bs)
			delete(gt.BranchSessions, bs)
			err := tc.holder.RemoveBranchSession(gt.GlobalSession, bs)
			if err != nil {
				return false, err
			}
			continue
		}
		branchStatus, err1 := tc.branchCommit(bs)
		if err1 != nil {
			log.Errorf("exception committing branch %v, err: %s", bs, err1.Error())
			if !retrying {
				err := tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.CommitRetrying)
				if err != nil {
					return false, err
				}
			}
			return false, err1
		}
		switch branchStatus {
		case apis.PhaseTwoCommitted:
			tc.resourceDataLocker.ReleaseLock(bs)
			delete(gt.BranchSessions, bs)
			err := tc.holder.RemoveBranchSession(gt.GlobalSession, bs)
			if err != nil {
				return false, err
			}
			continue
		case apis.PhaseTwoCommitFailedCanNotRetry:
			{
				if gt.CanBeCommittedAsync() {
					log.Errorf("by [%s], failed to commit branch %v", bs.Status.String(), bs)
					continue
				} else {
					// change status first, if need retention global session data,
					// might not remove global session, then, the status is very important.
					err := tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.CommitFailed)
					if err != nil {
						return false, err
					}
					tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
					err = tc.holder.RemoveGlobalTransaction(gt)
					if err != nil {
						return false, err
					}
					log.Errorf("finally, failed to commit global[%d] since branch[%d] commit failed", gt.XID, bs.BranchID)
					return false, nil
				}
			}
		default:
			{
				if !retrying {
					err := tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.CommitRetrying)
					if err != nil {
						return false, err
					}
					return false, nil
				}
				if gt.CanBeCommittedAsync() {
					log.Errorf("by [%s], failed to commit branch %v", bs.Status.String(), bs)
					continue
				} else {
					log.Errorf("failed to commit global[%d] since branch[%d] commit failed, will retry later.", gt.XID, bs.BranchID)
					return false, nil
				}
			}
		}
	}
	gs := tc.holder.FindGlobalTransaction(gt.XID)
	if gs != nil && gs.HasBranch() {
		log.Infof("global[%d] committing is NOT done.", gt.XID)
		return false, nil
	}

	// change status first, if need retention global session data,
	// might not remove global session, then, the status is very important.
	err2 := tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.Committed)
	if err2 != nil {
		return false, err2
	}
	tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
	err2 = tc.holder.RemoveGlobalTransaction(gt)
	if err2 != nil {
		return false, err2
	}
	runtime.GoWithRecover(func() {
		evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime,
			int64(time2.CurrentTimeMillis()), gt.Status)
		event.Bus.GlobalTransactionEventChannel <- evt
	}, nil)
	log.Infof("global[%d] committing is successfully done.", gt.XID)

	return true, err
}

func (tc *TransactionCoordinator) branchCommit(bs *apis.BranchSession) (apis.BranchSession_BranchStatus, error) {
	client, err := tc.getTransactionCoordinatorServiceClient(bs.Addressing)
	if err != nil {
		return bs.Status, err
	}

	response, err := client.BranchCommit(context.Background(), &apis.BranchCommitRequest{
		XID:             bs.XID,
		BranchID:        bs.BranchID,
		ResourceID:      bs.ResourceID,
		LockKey:         bs.LockKey,
		BranchType:      bs.Type,
		ApplicationData: bs.ApplicationData,
	})

	if err != nil {
		return bs.Status, err
	}

	if response.ResultCode == apis.ResultCodeSuccess {
		return response.BranchStatus, nil
	}
	return bs.Status, fmt.Errorf(response.Message)
}

func (tc *TransactionCoordinator) Rollback(ctx context.Context, request *apis.GlobalRollbackRequest) (*apis.GlobalRollbackResponse, error) {
	gt := tc.holder.FindGlobalTransaction(request.XID)
	if gt == nil {
		return &apis.GlobalRollbackResponse{
			ResultCode:   apis.ResultCodeSuccess,
			GlobalStatus: apis.Finished,
		}, nil
	}
	shouldRollBack, err := func(gt *model.GlobalTransaction) (bool, error) {
		result, err := tc.locker.TryLock(gt.GlobalSession, time.Duration(gt.Timeout)*time.Millisecond)
		if err != nil {
			return false, err
		}
		if result {
			defer tc.locker.Unlock(gt.GlobalSession)
			if gt.Active {
				// Active need persistence
				// Highlight: Firstly, close the session, then no more branch can be registered.
				err = tc.holder.InactiveGlobalSession(gt.GlobalSession)
				if err != nil {
					return false, fmt.Errorf("InactiveGlobalSession Err: %+v", err)
				}
			}
			if gt.Status == apis.Begin {
				err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RollingBack)
				if err != nil {
					return false, fmt.Errorf("UpdateGlobalSessionStatus Err: %+v", err)
				}
				return true, nil
			}
			return false, nil
		}
		return false, fmt.Errorf("failed to lock global transaction xid = %s", request.XID)
	}(gt)

	if err != nil {
		return &apis.GlobalRollbackResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.FailedLockGlobalTransaction,
			Message:       err.Error(),
			GlobalStatus:  gt.Status,
		}, nil
	}

	if !shouldRollBack {
		return &apis.GlobalRollbackResponse{
			ResultCode:   apis.ResultCodeSuccess,
			GlobalStatus: gt.Status,
		}, nil
	}

	_, err = tc.doGlobalRollback(gt, false)
	if err != nil {
		return nil, err
	}
	return &apis.GlobalRollbackResponse{
		ResultCode:   apis.ResultCodeSuccess,
		GlobalStatus: gt.Status,
	}, nil
}

func (tc *TransactionCoordinator) doGlobalRollback(gt *model.GlobalTransaction, retrying bool) (bool, error) {
	var err error

	runtime.GoWithRecover(func() {
		evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime, 0, gt.Status)
		event.Bus.GlobalTransactionEventChannel <- evt
	}, nil)

	if gt.IsSaga() {
		return false, status.Errorf(codes.Unimplemented, "method Commit not supported saga mode")
	}

	for bs := range gt.BranchSessions {
		if bs.Status == apis.PhaseOneFailed {
			tc.resourceDataLocker.ReleaseLock(bs)
			delete(gt.BranchSessions, bs)
			err = tc.holder.RemoveBranchSession(gt.GlobalSession, bs)
			if err != nil {
				return false, err
			}
			continue
		}
		branchStatus, err1 := tc.branchRollback(bs)
		if err1 != nil {
			log.Errorf("exception rolling back branch xid=%d branchID=%d", gt.XID, bs.BranchID)
			if !retrying {
				if gt.IsTimeoutGlobalStatus() {
					err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.TimeoutRollbackRetrying)
					if err != nil {
						return false, err
					}
				} else {
					err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RollbackRetrying)
					if err != nil {
						return false, err
					}
				}
			}
			return false, err1
		}
		switch branchStatus {
		case apis.PhaseTwoRolledBack:
			tc.resourceDataLocker.ReleaseLock(bs)
			delete(gt.BranchSessions, bs)
			err = tc.holder.RemoveBranchSession(gt.GlobalSession, bs)
			if err != nil {
				return false, err
			}
			log.Infof("successfully rollback branch xid=%d branchID=%d", gt.XID, bs.BranchID)
			continue
		case apis.PhaseTwoRollbackFailedCanNotRetry:
			if gt.IsTimeoutGlobalStatus() {
				err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.TimeoutRollbackFailed)
				if err != nil {
					return false, err
				}
			} else {
				err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RollbackFailed)
				if err != nil {
					return false, err
				}
			}
			tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
			err = tc.holder.RemoveGlobalTransaction(gt)
			if err != nil {
				return false, err
			}
			log.Infof("failed to rollback branch and stop retry xid=%d branchID=%d", gt.XID, bs.BranchID)
			return false, nil
		default:
			log.Infof("failed to rollback branch xid=%d branchID=%d", gt.XID, bs.BranchID)
			if !retrying {
				if gt.IsTimeoutGlobalStatus() {
					err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.TimeoutRollbackRetrying)
					if err != nil {
						return false, err
					}
				} else {
					err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RollbackRetrying)
					if err != nil {
						return false, err
					}
				}
			}
			return false, nil
		}
	}

	// In db mode, there is a problem of inconsistent data in multiple copies, resulting in new branch
	// transaction registration when rolling back.
	// 1. New branch transaction and rollback branch transaction have no data association
	// 2. New branch transaction has data association with rollback branch transaction
	// The second query can solve the first problem, and if it is the second problem, it may cause a rollback
	// failure due to data changes.
	gs := tc.holder.FindGlobalTransaction(gt.XID)
	if gs != nil && gs.HasBranch() {
		log.Infof("Global[%d] rolling back is NOT done.", gt.XID)
		return false, nil
	}

	if gt.IsTimeoutGlobalStatus() {
		err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.TimeoutRolledBack)
		if err != nil {
			return false, err
		}
	} else {
		err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RolledBack)
		if err != nil {
			return false, err
		}
	}
	tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
	err = tc.holder.RemoveGlobalTransaction(gt)
	if err != nil {
		return false, err
	}
	runtime.GoWithRecover(func() {
		evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime,
			int64(time2.CurrentTimeMillis()), gt.Status)
		event.Bus.GlobalTransactionEventChannel <- evt
	}, nil)
	log.Infof("successfully rollback global, xid = %d", gt.XID)

	return true, err
}

func (tc *TransactionCoordinator) branchRollback(bs *apis.BranchSession) (apis.BranchSession_BranchStatus, error) {
	client, err := tc.getTransactionCoordinatorServiceClient(bs.Addressing)
	if err != nil {
		return bs.Status, err
	}

	response, err := client.BranchRollback(context.Background(), &apis.BranchRollbackRequest{
		XID:             bs.XID,
		BranchID:        bs.BranchID,
		ResourceID:      bs.ResourceID,
		LockKey:         bs.LockKey,
		BranchType:      bs.Type,
		ApplicationData: bs.ApplicationData,
	})

	if err != nil {
		return bs.Status, err
	}

	if response.ResultCode == apis.ResultCodeSuccess {
		return response.BranchStatus, nil
	}
	return bs.Status, fmt.Errorf(response.Message)
}

func (tc *TransactionCoordinator) BranchRegister(ctx context.Context, request *apis.BranchRegisterRequest) (*apis.BranchRegisterResponse, error) {
	gt := tc.holder.FindGlobalTransaction(request.XID)
	if gt == nil {
		log.Errorf("could not found global transaction xid = %s", request.XID)
		return &apis.BranchRegisterResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.GlobalTransactionNotExist,
			Message:       fmt.Sprintf("could not found global transaction xid = %s", request.XID),
		}, nil
	}

	result, err := tc.locker.TryLock(gt.GlobalSession, time.Duration(gt.Timeout)*time.Millisecond)
	if err != nil {
		return &apis.BranchRegisterResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.FailedLockGlobalTransaction,
			Message:       fmt.Sprintf("could not found global transaction xid = %s", request.XID),
		}, nil
	}
	if result {
		defer tc.locker.Unlock(gt.GlobalSession)
		if !gt.Active {
			return &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.GlobalTransactionNotActive,
				Message:       fmt.Sprintf("could not register branch into global session xid = %s status = %d", gt.XID, gt.Status),
			}, nil
		}
		if gt.Status != apis.Begin {
			return &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.GlobalTransactionStatusInvalid,
				Message: fmt.Sprintf("could not register branch into global session xid = %s status = %d while expecting %d",
					gt.XID, gt.Status, apis.Begin),
			}, nil
		}

		bs := &apis.BranchSession{
			Addressing:      request.Addressing,
			XID:             request.XID,
			BranchID:        uuid.NextID(),
			TransactionID:   gt.TransactionID,
			ResourceID:      request.ResourceID,
			LockKey:         request.LockKey,
			Type:            request.BranchType,
			Status:          apis.Registered,
			ApplicationData: request.ApplicationData,
		}

		if bs.Type == apis.AT {
			result := tc.resourceDataLocker.AcquireLock(bs)
			if !result {
				return &apis.BranchRegisterResponse{
					ResultCode:    apis.ResultCodeFailed,
					ExceptionCode: apis.LockKeyConflict,
					Message: fmt.Sprintf("branch lock acquire failed xid = %s resourceId = %s, lockKey = %s",
						request.XID, request.ResourceID, request.LockKey),
				}, nil
			}
		}

		err := tc.holder.AddBranchSession(gt.GlobalSession, bs)
		if err != nil {
			return &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.BranchRegisterFailed,
				Message:       fmt.Sprintf("branch register failed,xid = %s, branchID = %d", gt.XID, bs.BranchID),
			}, nil
		}

		if !gt.IsSaga() {
			_, err = tc.getTransactionCoordinatorServiceClient(bs.Addressing)
			if err != nil {
				return nil, err
			}
		}

		return &apis.BranchRegisterResponse{
			ResultCode: apis.ResultCodeSuccess,
			BranchID:   bs.BranchID,
		}, nil
	}

	return &apis.BranchRegisterResponse{
		ResultCode:    apis.ResultCodeFailed,
		ExceptionCode: apis.FailedLockGlobalTransaction,
		Message:       fmt.Sprintf("failed to lock global transaction xid = %s", request.XID),
	}, nil
}

func (tc *TransactionCoordinator) BranchReport(ctx context.Context, request *apis.BranchReportRequest) (*apis.BranchReportResponse, error) {
	gt := tc.holder.FindGlobalTransaction(request.XID)
	if gt == nil {
		log.Errorf("could not found global transaction xid = %s", request.XID)
		return &apis.BranchReportResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.GlobalTransactionNotExist,
			Message:       fmt.Sprintf("could not found global transaction xid = %s", request.XID),
		}, nil
	}

	bs := gt.GetBranch(request.BranchID)
	if bs == nil {
		return &apis.BranchReportResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.BranchTransactionNotExist,
			Message:       fmt.Sprintf("could not found branch session xid = %s branchID = %d", gt.XID, request.BranchID),
		}, nil
	}

	err := tc.holder.UpdateBranchSessionStatus(bs, request.BranchStatus)
	if err != nil {
		return &apis.BranchReportResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.BranchReportFailed,
			Message:       fmt.Sprintf("branch report failed,xid = %s, branchID = %d", gt.XID, bs.BranchID),
		}, nil
	}

	return &apis.BranchReportResponse{
		ResultCode: apis.ResultCodeSuccess,
	}, nil
}

func (tc *TransactionCoordinator) LockQuery(ctx context.Context, request *apis.GlobalLockQueryRequest) (*apis.GlobalLockQueryResponse, error) {
	result := tc.resourceDataLocker.IsLockable(request.XID, request.ResourceID, request.LockKey)
	return &apis.GlobalLockQueryResponse{
		ResultCode: apis.ResultCodeSuccess,
		Lockable:   result,
	}, nil
}

func (tc *TransactionCoordinator) getTransactionCoordinatorServiceClient(addressing string) (apis.BranchTransactionServiceClient, error) {
	client1, ok1 := tc.tcServiceClients[addressing]
	if ok1 {
		return client1, nil
	}

	tc.Mutex.Lock()
	defer tc.Mutex.Unlock()
	client2, ok2 := tc.tcServiceClients[addressing]
	if ok2 {
		return client2, nil
	}

	conn, err := grpc.Dial(addressing, grpc.WithInsecure(), grpc.WithKeepaliveParams(tc.keepaliveClientParameters))
	if err != nil {
		return nil, err
	}
	client := apis.NewBranchTransactionServiceClient(conn)
	tc.tcServiceClients[addressing] = client
	return client, nil
}

func (tc *TransactionCoordinator) processTimeoutCheck() {
	for {
		timer := time.NewTimer(tc.timeoutRetryPeriod)
		<-timer.C
		tc.timeoutCheck()
		timer.Stop()
	}
}

func (tc *TransactionCoordinator) processRetryRollingBack() {
	for {
		timer := time.NewTimer(tc.rollingBackRetryPeriod)
		<-timer.C
		tc.handleRetryRollingBack()
		timer.Stop()
	}
}

func (tc *TransactionCoordinator) processRetryCommitting() {
	for {
		timer := time.NewTimer(tc.committingRetryPeriod)
		<-timer.C
		tc.handleRetryCommitting()
		timer.Stop()
	}
}

func (tc *TransactionCoordinator) processAsyncCommitting() {
	for {
		timer := time.NewTimer(tc.asyncCommittingRetryPeriod)
		<-timer.C
		tc.handleAsyncCommitting()
		timer.Stop()
	}
}

func (tc *TransactionCoordinator) timeoutCheck() {
	allSessions := tc.holder.AllSessions()
	if len(allSessions) == 0 {
		return
	}
	for _, globalSession := range allSessions {
		if globalSession.Status == apis.Begin && isGlobalSessionTimeout(globalSession) {
			result, err := tc.locker.TryLock(globalSession, time.Duration(globalSession.Timeout)*time.Millisecond)
			if err == nil && result {
				if globalSession.Active {
					// Active need persistence
					// Highlight: Firstly, close the session, then no more branch can be registered.
					err = tc.holder.InactiveGlobalSession(globalSession)
					if err != nil {
						return
					}
				}
				if globalSession.Status == apis.Begin {
					err = tc.holder.UpdateGlobalSessionStatus(globalSession, apis.TimeoutRollingBack)
					if err != nil {
						return
					}
				}
				tc.locker.Unlock(globalSession)
				evt := event.NewGlobalTransactionEvent(globalSession.TransactionID, event.RoleTC, globalSession.TransactionName, globalSession.BeginTime, 0, globalSession.Status)
				event.Bus.GlobalTransactionEventChannel <- evt
			}
		}
	}
}

func (tc *TransactionCoordinator) handleRetryRollingBack() {
	rollbackTransactions := tc.holder.FindRetryRollbackGlobalTransactions()
	if len(rollbackTransactions) == 0 {
		return
	}
	now := time2.CurrentTimeMillis()
	for _, transaction := range rollbackTransactions {
		if transaction.Status == apis.RollingBack && !transaction.IsRollingBackDead() {
			continue
		}
		if isRetryTimeout(int64(now), tc.maxRollbackRetryTimeout, transaction.BeginTime) {
			if tc.rollbackRetryTimeoutUnlockEnable {
				tc.resourceDataLocker.ReleaseGlobalSessionLock(transaction)
			}
			err := tc.holder.RemoveGlobalTransaction(transaction)
			if err != nil {
				return
			}
			log.Errorf("GlobalSession rollback retry timeout and removed [%s]", transaction.XID)
			continue
		}
		_, err := tc.doGlobalRollback(transaction, true)
		if err != nil {
			log.Errorf("failed to retry rollback [%s]", transaction.XID)
		}
	}
}

func isRetryTimeout(now int64, timeout int64, beginTime int64) bool {
	if timeout >= AlwaysRetryBoundary && now-beginTime > timeout {
		return true
	}
	return false
}

func (tc *TransactionCoordinator) handleRetryCommitting() {
	committingTransactions := tc.holder.FindRetryCommittingGlobalTransactions()
	if len(committingTransactions) == 0 {
		return
	}
	now := time2.CurrentTimeMillis()
	for _, transaction := range committingTransactions {
		if isRetryTimeout(int64(now), tc.maxCommitRetryTimeout, transaction.BeginTime) {
			err := tc.holder.RemoveGlobalTransaction(transaction)
			if err != nil {
				return
			}
			log.Errorf("GlobalSession commit retry timeout and removed [%s]", transaction.XID)
			continue
		}
		_, err := tc.doGlobalCommit(transaction, true)
		if err != nil {
			log.Errorf("failed to retry committing [%s]", transaction.XID)
		}
	}
}

func (tc *TransactionCoordinator) handleAsyncCommitting() {
	asyncCommittingTransactions := tc.holder.FindAsyncCommittingGlobalTransactions()
	if len(asyncCommittingTransactions) == 0 {
		return
	}
	for _, transaction := range asyncCommittingTransactions {
		if transaction.Status != apis.AsyncCommitting {
			continue
		}
		_, err := tc.doGlobalCommit(transaction, true)
		if err != nil {
			log.Errorf("failed to async committing [%s]", transaction.XID)
		}
	}
}

func isGlobalSessionTimeout(gt *apis.GlobalSession) bool {
	return (time2.CurrentTimeMillis() - uint64(gt.BeginTime)) > uint64(gt.Timeout)
}
