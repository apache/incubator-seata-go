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

const ALWAYS_RETRY_BOUNDARY = 0

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
	resourceDataLocker *lock.LockManager
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

func (tc TransactionCoordinator) Begin(ctx context.Context, request *apis.GlobalBeginRequest) (*apis.GlobalBeginResponse, error) {
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
		event.EventBus.GlobalTransactionEventChannel <- evt
	}, nil)

	log.Infof("successfully begin global transaction xid = {}", gt.XID)
	return &apis.GlobalBeginResponse{
		ResultCode: apis.ResultCodeSuccess,
		XID:        xid,
	}, nil
}

func (tc TransactionCoordinator) GetStatus(ctx context.Context, request *apis.GlobalStatusRequest) (*apis.GlobalStatusResponse, error) {
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

func (tc TransactionCoordinator) GlobalReport(ctx context.Context, request *apis.GlobalReportRequest) (*apis.GlobalReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GlobalReport not implemented")
}

func (tc TransactionCoordinator) Commit(ctx context.Context, request *apis.GlobalCommitRequest) (*apis.GlobalCommitResponse, error) {
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
				tc.holder.InactiveGlobalSession(gt.GlobalSession)
			}
			tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
			if gt.Status == apis.Begin {
				tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.Committing)
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
		return &apis.GlobalCommitResponse{
			ResultCode:   apis.ResultCodeSuccess,
			GlobalStatus: gt.Status,
		}, nil
	}

	if gt.CanBeCommittedAsync() {
		tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.AsyncCommitting)
		return &apis.GlobalCommitResponse{
			ResultCode:   apis.ResultCodeSuccess,
			GlobalStatus: apis.Committed,
		}, nil
	} else {
		_, err := tc.doGlobalCommit(gt, false)
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
}

func (tc TransactionCoordinator) doGlobalCommit(gt *model.GlobalTransaction, retrying bool) (bool, error) {
	var (
		success = true
		err     error
	)

	runtime.GoWithRecover(func() {
		evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime, 0, gt.Status)
		event.EventBus.GlobalTransactionEventChannel <- evt
	}, nil)

	if gt.IsSaga() {
		return false, status.Errorf(codes.Unimplemented, "method Commit not supported saga mode")
	} else {
		for bs := range gt.BranchSessions {
			if bs.Status == apis.PhaseOneFailed {
				tc.resourceDataLocker.ReleaseLock(bs)
				delete(gt.BranchSessions, bs)
				tc.holder.RemoveBranchSession(gt.GlobalSession, bs)
				continue
			}
			branchStatus, err1 := tc.branchCommit(bs)
			if err1 != nil {
				log.Errorf("exception committing branch %v, err: %s", bs, err1.Error())
				if !retrying {
					tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.CommitRetrying)
				}
				return false, err1
			}
			switch branchStatus {
			case apis.PhaseTwoCommitted:
				tc.resourceDataLocker.ReleaseLock(bs)
				delete(gt.BranchSessions, bs)
				tc.holder.RemoveBranchSession(gt.GlobalSession, bs)
				continue
			case apis.PhaseTwoCommitFailedCanNotRetry:
				{
					if gt.CanBeCommittedAsync() {
						log.Errorf("by [%s], failed to commit branch %v", bs.Status.String(), bs)
						continue
					} else {
						// change status first, if need retention global session data,
						// might not remove global session, then, the status is very important.
						tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.CommitFailed)
						tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
						tc.holder.RemoveGlobalTransaction(gt)
						log.Errorf("finally, failed to commit global[%d] since branch[%d] commit failed", gt.XID, bs.BranchID)
						return false, nil
					}
				}
			default:
				{
					if !retrying {
						tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.CommitRetrying)
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
	}
	if success {
		// change status first, if need retention global session data,
		// might not remove global session, then, the status is very important.
		tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.Committed)
		tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
		tc.holder.RemoveGlobalTransaction(gt)

		runtime.GoWithRecover(func() {
			evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime,
				int64(time2.CurrentTimeMillis()), gt.Status)
			event.EventBus.GlobalTransactionEventChannel <- evt
		}, nil)

		log.Infof("global[%d] committing is successfully done.", gt.XID)
	}
	return success, err
}

func (tc TransactionCoordinator) branchCommit(bs *apis.BranchSession) (apis.BranchSession_BranchStatus, error) {
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
	} else {
		return bs.Status, fmt.Errorf(response.Message)
	}
}

func (tc TransactionCoordinator) Rollback(ctx context.Context, request *apis.GlobalRollbackRequest) (*apis.GlobalRollbackResponse, error) {
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
				tc.holder.InactiveGlobalSession(gt.GlobalSession)
			}
			if gt.Status == apis.Begin {
				tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RollingBack)
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

	tc.doGlobalRollback(gt, false)
	return &apis.GlobalRollbackResponse{
		ResultCode:   apis.ResultCodeSuccess,
		GlobalStatus: gt.Status,
	}, nil
}

func (tc TransactionCoordinator) doGlobalRollback(gt *model.GlobalTransaction, retrying bool) (bool, error) {
	var (
		success = true
		err     error
	)

	runtime.GoWithRecover(func() {
		evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime, 0, gt.Status)
		event.EventBus.GlobalTransactionEventChannel <- evt
	}, nil)

	if gt.IsSaga() {
		return false, status.Errorf(codes.Unimplemented, "method Commit not supported saga mode")
	} else {
		for bs := range gt.BranchSessions {
			if bs.Status == apis.PhaseOneFailed {
				tc.resourceDataLocker.ReleaseLock(bs)
				delete(gt.BranchSessions, bs)
				tc.holder.RemoveBranchSession(gt.GlobalSession, bs)
				continue
			}
			branchStatus, err1 := tc.branchRollback(bs)
			if err1 != nil {
				log.Errorf("exception rolling back branch xid=%d branchID=%d", gt.XID, bs.BranchID)
				if !retrying {
					if gt.IsTimeoutGlobalStatus() {
						tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.TimeoutRollbackRetrying)
					} else {
						tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RollbackRetrying)
					}
				}
				return false, err1
			}
			switch branchStatus {
			case apis.PhaseTwoRolledBack:
				tc.resourceDataLocker.ReleaseLock(bs)
				delete(gt.BranchSessions, bs)
				tc.holder.RemoveBranchSession(gt.GlobalSession, bs)
				log.Infof("successfully rollback branch xid=%d branchID=%d", gt.XID, bs.BranchID)
				continue
			case apis.PhaseTwoRollbackFailedCanNotRetry:
				if gt.IsTimeoutGlobalStatus() {
					tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.TimeoutRollbackFailed)
				} else {
					tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RollbackFailed)
				}
				tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
				tc.holder.RemoveGlobalTransaction(gt)
				log.Infof("failed to rollback branch and stop retry xid=%d branchID=%d", gt.XID, bs.BranchID)
				return false, nil
			default:
				log.Infof("failed to rollback branch xid=%d branchID=%d", gt.XID, bs.BranchID)
				if !retrying {
					if gt.IsTimeoutGlobalStatus() {
						tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.TimeoutRollbackRetrying)
					} else {
						tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RollbackRetrying)
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
	}
	if success {
		if gt.IsTimeoutGlobalStatus() {
			tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.TimeoutRolledBack)
		} else {
			tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RolledBack)
		}
		tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
		tc.holder.RemoveGlobalTransaction(gt)

		runtime.GoWithRecover(func() {
			evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime,
				int64(time2.CurrentTimeMillis()), gt.Status)
			event.EventBus.GlobalTransactionEventChannel <- evt
		}, nil)

		log.Infof("successfully rollback global, xid = %d", gt.XID)
	}
	return success, err
}

func (tc TransactionCoordinator) branchRollback(bs *apis.BranchSession) (apis.BranchSession_BranchStatus, error) {
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
	} else {
		return bs.Status, fmt.Errorf(response.Message)
	}
}

func (tc TransactionCoordinator) BranchRegister(ctx context.Context, request *apis.BranchRegisterRequest) (*apis.BranchRegisterResponse, error) {
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
			tc.getTransactionCoordinatorServiceClient(bs.Addressing)
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

func (tc TransactionCoordinator) BranchReport(ctx context.Context, request *apis.BranchReportRequest) (*apis.BranchReportResponse, error) {
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

func (tc TransactionCoordinator) LockQuery(ctx context.Context, request *apis.GlobalLockQueryRequest) (*apis.GlobalLockQueryResponse, error) {
	result := tc.resourceDataLocker.IsLockable(request.XID, request.ResourceID, request.LockKey)
	return &apis.GlobalLockQueryResponse{
		ResultCode: apis.ResultCodeSuccess,
		Lockable:   result,
	}, nil
}

func (tc TransactionCoordinator) getTransactionCoordinatorServiceClient(addressing string) (apis.BranchTransactionServiceClient, error) {
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

func (tc TransactionCoordinator) processTimeoutCheck() {
	for {
		timer := time.NewTimer(tc.timeoutRetryPeriod)
		select {
		case <-timer.C:
			tc.timeoutCheck()
		}
		timer.Stop()
	}
}

func (tc TransactionCoordinator) processRetryRollingBack() {
	for {
		timer := time.NewTimer(tc.rollingBackRetryPeriod)
		select {
		case <-timer.C:
			tc.handleRetryRollingBack()
		}
		timer.Stop()
	}
}

func (tc TransactionCoordinator) processRetryCommitting() {
	for {
		timer := time.NewTimer(tc.committingRetryPeriod)
		select {
		case <-timer.C:
			tc.handleRetryCommitting()
		}
		timer.Stop()
	}
}

func (tc TransactionCoordinator) processAsyncCommitting() {
	for {
		timer := time.NewTimer(tc.asyncCommittingRetryPeriod)
		select {
		case <-timer.C:
			tc.handleRetryCommitting()
		}
		timer.Stop()
	}
}

func (tc TransactionCoordinator) timeoutCheck() {
	allSessions := tc.holder.AllSessions()
	if allSessions == nil && len(allSessions) <= 0 {
		return
	}
	for _, globalSession := range allSessions {
		if globalSession.Status == apis.Begin && isGlobalSessionTimeout(globalSession) {
			result, err := tc.locker.TryLock(globalSession, time.Duration(globalSession.Timeout)*time.Millisecond)
			if err == nil && result {
				if globalSession.Active {
					// Active need persistence
					// Highlight: Firstly, close the session, then no more branch can be registered.
					tc.holder.InactiveGlobalSession(globalSession)
				}
				if globalSession.Status == apis.Begin {
					tc.holder.UpdateGlobalSessionStatus(globalSession, apis.TimeoutRollingBack)
				}
				tc.locker.Unlock(globalSession)
				evt := event.NewGlobalTransactionEvent(globalSession.TransactionID, event.RoleTC, globalSession.TransactionName, globalSession.BeginTime, 0, globalSession.Status)
				event.EventBus.GlobalTransactionEventChannel <- evt
			}
		}
	}
}

func (tc TransactionCoordinator) handleRetryRollingBack() {
	rollbackTransactions := tc.holder.FindRetryRollbackGlobalTransactions()
	if rollbackTransactions == nil && len(rollbackTransactions) <= 0 {
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
			tc.holder.RemoveGlobalTransaction(transaction)
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
	if timeout >= ALWAYS_RETRY_BOUNDARY && now-beginTime > timeout {
		return true
	}
	return false
}

func (tc TransactionCoordinator) handleRetryCommitting() {
	committingTransactions := tc.holder.FindRetryCommittingGlobalTransactions()
	if committingTransactions == nil && len(committingTransactions) <= 0 {
		return
	}
	now := time2.CurrentTimeMillis()
	for _, transaction := range committingTransactions {
		if isRetryTimeout(int64(now), tc.maxCommitRetryTimeout, transaction.BeginTime) {
			tc.holder.RemoveGlobalTransaction(transaction)
			log.Errorf("GlobalSession commit retry timeout and removed [%s]", transaction.XID)
			continue
		}
		_, err := tc.doGlobalCommit(transaction, true)
		if err != nil {
			log.Errorf("failed to retry committing [%s]", transaction.XID)
		}
	}
}

func (tc TransactionCoordinator) handleAsyncCommitting() {
	asyncCommittingTransactions := tc.holder.FindAsyncCommittingGlobalTransactions()
	if asyncCommittingTransactions == nil && len(asyncCommittingTransactions) <= 0 {
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
