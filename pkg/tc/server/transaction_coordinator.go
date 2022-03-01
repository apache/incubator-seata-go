package server

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/gogo/protobuf/types"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	common2 "github.com/opentrx/seata-golang/v2/pkg/common"
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
	rollbackDeadSeconds              int64
	rollbackRetryTimeoutUnlockEnable bool

	asyncCommittingRetryPeriod time.Duration
	committingRetryPeriod      time.Duration
	rollingBackRetryPeriod     time.Duration
	timeoutRetryPeriod         time.Duration

	streamMessageTimeout time.Duration

	holder             holder.SessionHolderInterface
	resourceDataLocker lock.LockManagerInterface
	locker             GlobalSessionLocker

	idGenerator        *atomic.Uint64
	futures            *sync.Map
	activeApplications *sync.Map
	callBackMessages   *sync.Map
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
		rollbackDeadSeconds:              conf.Server.RollbackDeadSeconds,
		rollbackRetryTimeoutUnlockEnable: conf.Server.RollbackRetryTimeoutUnlockEnable,

		asyncCommittingRetryPeriod: conf.Server.AsyncCommittingRetryPeriod,
		committingRetryPeriod:      conf.Server.CommittingRetryPeriod,
		rollingBackRetryPeriod:     conf.Server.RollingBackRetryPeriod,
		timeoutRetryPeriod:         conf.Server.TimeoutRetryPeriod,

		streamMessageTimeout: conf.Server.StreamMessageTimeout,

		holder:             holder.NewSessionHolder(driver),
		resourceDataLocker: lock.NewLockManager(driver),
		locker:             new(UnimplementedGlobalSessionLocker),

		idGenerator:        &atomic.Uint64{},
		futures:            &sync.Map{},
		activeApplications: &sync.Map{},
		callBackMessages:   &sync.Map{},
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
		event.EventBus.GlobalTransactionEventChannel <- evt
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
					return false, err
				}
			}
			tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
			if gt.Status == apis.Begin {
				err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.Committing)
				if err != nil {
					return false, err
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
		if err != nil {
			return nil, err
		}
		return &apis.GlobalCommitResponse{
			ResultCode:   apis.ResultCodeSuccess,
			GlobalStatus: apis.Committed,
		}, nil
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
		event.EventBus.GlobalTransactionEventChannel <- evt
	}, nil)

	if gt.IsSaga() {
		return false, status.Errorf(codes.Unimplemented, "method Commit not supported saga mode")
	}

	for bs := range gt.BranchSessions {
		branchStatus, err1 := tc.branchCommit(bs)
		if err1 != nil {
			log.Errorf("exception committing branch xid=%d branchID=%d, err: %v", bs.GetXID(), bs.BranchID, err1)
			if !retrying {
				err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.CommitRetrying)
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
			err = tc.holder.RemoveBranchSession(gt.GlobalSession, bs)
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
					err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.CommitFailed)
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
					err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.CommitRetrying)
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
	err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.Committed)
	if err != nil {
		return false, err
	}
	tc.resourceDataLocker.ReleaseGlobalSessionLock(gt)
	err = tc.holder.RemoveGlobalTransaction(gt)
	if err != nil {
		return false, err
	}
	runtime.GoWithRecover(func() {
		evt := event.NewGlobalTransactionEvent(gt.TransactionID, event.RoleTC, gt.TransactionName, gt.BeginTime,
			int64(time2.CurrentTimeMillis()), gt.Status)
		event.EventBus.GlobalTransactionEventChannel <- evt
	}, nil)
	log.Infof("global[%d] committing is successfully done.", gt.XID)

	return true, err
}

func (tc *TransactionCoordinator) branchCommit(bs *apis.BranchSession) (apis.BranchSession_BranchStatus, error) {
	request := &apis.BranchCommitRequest{
		XID:             bs.XID,
		BranchID:        bs.BranchID,
		ResourceID:      bs.ResourceID,
		LockKey:         bs.LockKey,
		BranchType:      bs.Type,
		ApplicationData: bs.ApplicationData,
	}

	content, err := types.MarshalAny(request)
	if err != nil {
		return bs.Status, err
	}

	message := &apis.BranchMessage{
		ID:                int64(tc.idGenerator.Inc()),
		BranchMessageType: apis.TypeBranchCommit,
		Message:           content,
	}

	queue, _ := tc.callBackMessages.LoadOrStore(bs.Addressing, make(chan *apis.BranchMessage))
	q := queue.(chan *apis.BranchMessage)
	select {
	case q <- message:
	default:
		return bs.Status, err
	}

	resp := common2.NewMessageFuture(message)
	tc.futures.Store(message.ID, resp)

	timer := time.NewTimer(tc.streamMessageTimeout)
	select {
	case <-timer.C:
		tc.futures.Delete(resp.ID)
		return bs.Status, fmt.Errorf("wait branch commit response timeout")
	case <-resp.Done:
		timer.Stop()
	}

	response, ok := resp.Response.(*apis.BranchCommitResponse)
	if !ok {
		log.Infof("rollback response: %v", resp.Response)
		return bs.Status, fmt.Errorf("response type not right")
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
					return false, err
				}
			}
			if gt.Status == apis.Begin {
				err = tc.holder.UpdateGlobalSessionStatus(gt.GlobalSession, apis.RollingBack)
				if err != nil {
					return false, err
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
		event.EventBus.GlobalTransactionEventChannel <- evt
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
			log.Errorf("exception rolling back branch xid=%d branchID=%d, err: %v", gt.XID, bs.BranchID, err1)
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
		event.EventBus.GlobalTransactionEventChannel <- evt
	}, nil)
	log.Infof("successfully rollback global, xid = %d", gt.XID)

	return true, err
}

func (tc *TransactionCoordinator) branchRollback(bs *apis.BranchSession) (apis.BranchSession_BranchStatus, error) {
	request := &apis.BranchRollbackRequest{
		XID:             bs.XID,
		BranchID:        bs.BranchID,
		ResourceID:      bs.ResourceID,
		LockKey:         bs.LockKey,
		BranchType:      bs.Type,
		ApplicationData: bs.ApplicationData,
	}

	content, err := types.MarshalAny(request)
	if err != nil {
		return bs.Status, err
	}
	message := &apis.BranchMessage{
		ID:                int64(tc.idGenerator.Inc()),
		BranchMessageType: apis.TypeBranchRollback,
		Message:           content,
	}

	queue, _ := tc.callBackMessages.LoadOrStore(bs.Addressing, make(chan *apis.BranchMessage))
	q := queue.(chan *apis.BranchMessage)
	select {
	case q <- message:
	default:
		return bs.Status, err
	}

	resp := common2.NewMessageFuture(message)
	tc.futures.Store(message.ID, resp)

	timer := time.NewTimer(tc.streamMessageTimeout)
	select {
	case <-timer.C:
		tc.futures.Delete(resp.ID)
		timer.Stop()
		return bs.Status, fmt.Errorf("wait branch rollback response timeout")
	case <-resp.Done:
		timer.Stop()
	}

	response := resp.Response.(*apis.BranchRollbackResponse)
	if response.ResultCode == apis.ResultCodeSuccess {
		return response.BranchStatus, nil
	}
	return bs.Status, fmt.Errorf(response.Message)
}

func (tc *TransactionCoordinator) BranchCommunicate(stream apis.ResourceManagerService_BranchCommunicateServer) error {
	var addressing string
	done := make(chan bool)

	ctx := stream.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		addressing = md.Get("addressing")[0]
		c, ok := tc.activeApplications.Load(addressing)
		if ok {
			count := c.(int)
			tc.activeApplications.Store(addressing, count+1)
		} else {
			tc.activeApplications.Store(addressing, 1)
		}
		defer func() {
			c, _ := tc.activeApplications.Load(addressing)
			count := c.(int)
			tc.activeApplications.Store(addressing, count-1)
		}()
	}

	queue, _ := tc.callBackMessages.LoadOrStore(addressing, make(chan *apis.BranchMessage))
	q := queue.(chan *apis.BranchMessage)

	runtime.GoWithRecover(func() {
		for {
			select {
			case _, ok := <-done:
				if !ok {
					return
				}
			case msg := <-q:
				err := stream.Send(msg)
				if err != nil {
					return
				}
			}
		}
	}, nil)

	for {
		select {
		case <-ctx.Done():
			close(done)
			return ctx.Err()
		default:
			branchMessage, err := stream.Recv()
			if err == io.EOF {
				close(done)
				return nil
			}
			if err != nil {
				close(done)
				return err
			}
			switch branchMessage.GetBranchMessageType() {
			case apis.TypeBranchCommitResult:
				response := &apis.BranchCommitResponse{}
				data := branchMessage.GetMessage().GetValue()
				err := response.Unmarshal(data)
				if err != nil {
					log.Error(err)
					continue
				}
				resp, loaded := tc.futures.Load(branchMessage.ID)
				if loaded {
					future := resp.(*common2.MessageFuture)
					future.Response = response
					future.Done <- true
					tc.futures.Delete(branchMessage.ID)
				}
			case apis.TypeBranchRollBackResult:
				response := &apis.BranchRollbackResponse{}
				data := branchMessage.GetMessage().GetValue()
				err := response.Unmarshal(data)
				if err != nil {
					log.Error(err)
					continue
				}
				resp, loaded := tc.futures.Load(branchMessage.ID)
				if loaded {
					future := resp.(*common2.MessageFuture)
					future.Response = response
					future.Done <- true
					tc.futures.Delete(branchMessage.ID)
				}
			}
		}
	}
}

func (tc *TransactionCoordinator) BranchRegister(ctx context.Context, request *apis.BranchRegisterRequest) (*apis.BranchRegisterResponse, error) {
	gt := tc.holder.FindGlobalTransaction(request.XID)
	if gt == nil {
		log.Errorf("could not find global transaction xid = %s", request.XID)
		return &apis.BranchRegisterResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.GlobalTransactionNotExist,
			Message:       fmt.Sprintf("could not find global transaction xid = %s", request.XID),
		}, nil
	}

	result, err := tc.locker.TryLock(gt.GlobalSession, time.Duration(gt.Timeout)*time.Millisecond)
	if err != nil {
		return &apis.BranchRegisterResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.FailedLockGlobalTransaction,
			Message:       fmt.Sprintf("could not lock global transaction xid = %s", request.XID),
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

		var asyncCommit bool
		if request.BranchType == apis.AT {
			asyncCommit = true
		} else {
			asyncCommit = request.AsyncCommit
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
			AsyncCommit:     asyncCommit,
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
			log.Error(err)
			return &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.BranchRegisterFailed,
				Message:       fmt.Sprintf("branch register failed, xid = %s, branchID = %d, err: %s", gt.XID, bs.BranchID, err.Error()),
			}, nil
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
		log.Errorf("could not find global transaction xid = %s", request.XID)
		return &apis.BranchReportResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.GlobalTransactionNotExist,
			Message:       fmt.Sprintf("could not find global transaction xid = %s", request.XID),
		}, nil
	}

	bs := gt.GetBranch(request.BranchID)
	if bs == nil {
		return &apis.BranchReportResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.BranchTransactionNotExist,
			Message:       fmt.Sprintf("could not find branch session xid = %s branchID = %d", gt.XID, request.BranchID),
		}, nil
	}

	err := tc.holder.UpdateBranchSessionStatus(bs, request.BranchStatus)
	if err != nil {
		return &apis.BranchReportResponse{
			ResultCode:    apis.ResultCodeFailed,
			ExceptionCode: apis.BranchReportFailed,
			Message:       fmt.Sprintf("branch report failed, xid = %s, branchID = %d, err: %s", gt.XID, bs.BranchID, err.Error()),
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
	sessions := tc.holder.FindGlobalSessions([]apis.GlobalSession_GlobalStatus{apis.Begin})
	if len(sessions) == 0 {
		return
	}
	for _, globalSession := range sessions {
		if isGlobalSessionTimeout(globalSession) {
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

				err = tc.holder.UpdateGlobalSessionStatus(globalSession, apis.TimeoutRollingBack)
				if err != nil {
					return
				}

				tc.locker.Unlock(globalSession)
				evt := event.NewGlobalTransactionEvent(globalSession.TransactionID, event.RoleTC, globalSession.TransactionName, globalSession.BeginTime, 0, globalSession.Status)
				event.EventBus.GlobalTransactionEventChannel <- evt
			}
		}
	}
}

func (tc *TransactionCoordinator) handleRetryRollingBack() {
	addressingIdentities := tc.getAddressingIdentities()
	if len(addressingIdentities) == 0 {
		return
	}
	rollbackTransactions := tc.holder.FindRetryRollbackGlobalTransactions(addressingIdentities)
	if len(rollbackTransactions) == 0 {
		return
	}
	now := time2.CurrentTimeMillis()
	for _, transaction := range rollbackTransactions {
		if transaction.Status == apis.RollingBack && !tc.IsRollingBackDead(transaction) {
			continue
		}
		if isRetryTimeout(int64(now), tc.maxRollbackRetryTimeout, transaction.BeginTime) {
			if tc.rollbackRetryTimeoutUnlockEnable {
				tc.resourceDataLocker.ReleaseGlobalSessionLock(transaction)
			}
			err := tc.holder.RemoveGlobalTransaction(transaction)
			if err != nil {
				log.Error(err)
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

func (tc *TransactionCoordinator) IsRollingBackDead(gt *model.GlobalTransaction) bool {
	return (time2.CurrentTimeMillis() - uint64(gt.BeginTime)) > uint64(tc.rollbackDeadSeconds)
}

func (tc *TransactionCoordinator) handleRetryCommitting() {
	addressingIdentities := tc.getAddressingIdentities()
	if len(addressingIdentities) == 0 {
		return
	}
	committingTransactions := tc.holder.FindRetryCommittingGlobalTransactions(addressingIdentities)
	if len(committingTransactions) == 0 {
		return
	}
	now := time2.CurrentTimeMillis()
	for _, transaction := range committingTransactions {
		if isRetryTimeout(int64(now), tc.maxCommitRetryTimeout, transaction.BeginTime) {
			err := tc.holder.RemoveGlobalTransaction(transaction)
			if err != nil {
				log.Error(err)
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
	addressingIdentities := tc.getAddressingIdentities()
	if len(addressingIdentities) == 0 {
		return
	}
	asyncCommittingTransactions := tc.holder.FindAsyncCommittingGlobalTransactions(addressingIdentities)
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

func (tc *TransactionCoordinator) getAddressingIdentities() []string {
	var addressIdentities []string
	tc.activeApplications.Range(func(key, value interface{}) bool {
		count := value.(int)
		if count > 0 {
			addressing := key.(string)
			addressIdentities = append(addressIdentities, addressing)
		}
		return true
	})
	return addressIdentities
}

func isGlobalSessionTimeout(gt *apis.GlobalSession) bool {
	return (time2.CurrentTimeMillis() - uint64(gt.BeginTime)) > uint64(gt.Timeout)
}
