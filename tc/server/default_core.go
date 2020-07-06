package server

import (
	"fmt"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/base/protocal"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/pkg/time"
	"github.com/dk-lockdown/seata-golang/tc/event"
	"github.com/dk-lockdown/seata-golang/tc/holder"
	"github.com/dk-lockdown/seata-golang/tc/lock"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

/**
 *  +--------------------+-----------------------+--------------------+
 *  |         TC         |Method(InBound)        |Method(OutBound)    |
 *  +--------------------+-----------------------+--------------------+
 *  |	                 |Begin                  |                    |
 *  |                    |BranchRegister         |                    |
 *  |       AT&TCC       |BranchReport           |branchCommit        |
 *  |    (DefaultCore)   |Commit                 |branchRollback      |
 *  |                    |Rollback               |                    |
 *  |                    |GetStatus              |                    |
 *  +--------------------+-----------------------+--------------------+
 *  |	      AT         |LockQuery              |                    |
 *  +--------------------+-----------------------+--------------------+
 *  |                    |(GlobalReport)         |                    |
 *  |                    |doGlobalCommit         |                    |
 *  |        SAGA        |doGlobalRollBack       |                    |
 *  |                    |doGlobalReport         |                    |
 *  +--------------------+-----------------------+--------------------+
 *
 * 参考 [effective go 之 Embedding](#https://my.oschina.net/pengfeix/blog/109967)
 * Go does not provide the typical, type-driven notion of subclassing,
 * but it does have the ability to “borrow” pieces of an implementation
 * by embedding types within a struct or interface.
 * Go 没有像其它面向对象语言中的类继承概念，但是，它可以通过在结构体或者接口中嵌入
 * 其它的类型，来使用被嵌入类型的功能。
 *
 * 原本 JAVA 版 Seata Sever 设计了 Core 接口，AbstractCore 实现该接口，ATCore、
 * TccCore、SagaCore 都继承 AbstractCore。使 ATCore、TccCore、SagaCore 每一
 * 个类单独拿出来都是 Core 接口的实现。但 Go 版的 Seata 我不打算这样设计。我们将
 * Core 接口里定义的接口方法拿出来，如上面的表格所示，一个全局事务的周期分别对应 Begin、
 * BranchRegister、BranchReport、Commit、Rollback 接口方法，这些接口方法适用于
 * AT 模式和 TCC 模式（SAGA 模式暂不了解，先不考虑）。AT 模式会多一个 LockQuery
 * 的接口。另外 OutBound 方向上有两个接口 branchCommit、branchRollback。JAVA 版
 * 的设计中 doGlobalCommit、doGlobalRollBack、doGlobalReport 其实是私有方法,
 * 这里用首字母小些开头的方法区分。那么 Go 版本的 DefaultCore 设计就出来了（暂不考虑 SAGA），
 * DefaultCore 内嵌入 ATCore。
 *
 */

type AbstractCore struct {
	MessageSender ServerMessageSender
}

type ATCore struct {
	AbstractCore
}

type SAGACore struct {
	AbstractCore
}

type DefaultCore struct {
	AbstractCore
	ATCore
	SAGACore
	coreMap map[meta.BranchType]interface{}
}

func NewCore(sender ServerMessageSender) TransactionCoordinator {
	return &DefaultCore{
		AbstractCore: AbstractCore{MessageSender: sender},
		ATCore:       ATCore{},
		SAGACore:     SAGACore{},
		coreMap:      make(map[meta.BranchType]interface{}),
	}
}

func (core *ATCore) branchSessionLock(globalSession *session.GlobalSession, branchSession *session.BranchSession) error {
	result := lock.GetLockManager().AcquireLock(branchSession)
	if !result {
		return meta.TransactionException{
			Code: meta.TransactionExceptionCodeLockKeyConflict,
			Message: fmt.Sprintf("Global lock acquire failed xid = %s branchId = %d",
				globalSession.Xid, branchSession.BranchId),
		}
	}
	return nil
}

func (core *ATCore) branchSessionUnlock(branchSession *session.BranchSession) error {
	lock.GetLockManager().ReleaseLock(branchSession)
	return nil
}

func (core *ATCore) LockQuery(branchType meta.BranchType,
	resourceId string,
	xid string,
	lockKeys string) bool {
	return lock.GetLockManager().IsLockable(xid, resourceId, lockKeys)
}

func (core *SAGACore) doGlobalCommit(globalSession *session.GlobalSession, retrying bool) (bool, error) {
	return true, nil
}

func (core *SAGACore) doGlobalRollback(globalSession *session.GlobalSession, retrying bool) (bool, error) {
	return true, nil
}

func (core *SAGACore) doGlobalReport(globalSession *session.GlobalSession, xid string, param meta.GlobalStatus) error {
	return nil
}

func (core *DefaultCore) Begin(applicationId string, transactionServiceGroup string, name string, timeout int32) (string, error) {
	gs := session.NewGlobalSession(
		session.WithGsApplicationId(applicationId),
		session.WithGsTransactionServiceGroup(transactionServiceGroup),
		session.WithGsTransactionName(name),
		session.WithGsTimeout(timeout),
	)

	gs.Begin()
	err := holder.GetSessionHolder().RootSessionManager.AddGlobalSession(gs)
	if err != nil {
		return "", err
	}

	go func() {
		evt := event.NewGlobalTransactionEvent(gs.TransactionId, event.RoleTC, gs.TransactionName, gs.BeginTime, 0, gs.Status)
		event.EventBus.GlobalTransactionEventChannel <- evt
	}()

	logging.Logger.Infof("Successfully begin global transaction xid = {}", gs.Xid)
	return gs.Xid, nil
}

func (core *DefaultCore) BranchRegister(branchType meta.BranchType,
	resourceId string,
	clientId string,
	xid string,
	applicationData []byte,
	lockKeys string) (int64, error) {
	gs, err := assertGlobalSessionNotNull(xid, false)
	if err != nil {
		return 0, err
	}
	defer gs.Unlock()
	gs.Lock()

	err1 := globalSessionStatusCheck(gs)
	if err1 != nil {
		return 0, err
	}

	bs := session.NewBranchSessionByGlobal(*gs,
		session.WithBsBranchType(branchType),
		session.WithBsResourceId(resourceId),
		session.WithBsApplicationData(applicationData),
		session.WithBsLockKey(lockKeys),
		session.WithBsClientId(clientId),
	)

	if branchType == meta.BranchTypeAT {
		core.ATCore.branchSessionLock(gs, bs)
	}
	gs.Add(bs)
	holder.GetSessionHolder().RootSessionManager.AddBranchSession(gs, bs)
	bs.Status = meta.BranchStatusRegistered

	logging.Logger.Infof("Successfully register branch xid = %s, branchId = %d", gs.Xid, bs.BranchId)
	return bs.BranchId, nil
}

func globalSessionStatusCheck(globalSession *session.GlobalSession) error {
	if !globalSession.Active {
		return meta.TransactionException{
			Code:    meta.TransactionExceptionCodeGlobalTransactionNotActive,
			Message: fmt.Sprintf("Could not register branch into global session xid = %s status = %d", globalSession.Xid, globalSession.Status),
		}
	}
	if globalSession.Status != meta.GlobalStatusBegin {
		return meta.TransactionException{
			Code: meta.TransactionExceptionCodeGlobalTransactionStatusInvalid,
			Message: fmt.Sprintf("Could not register branch into global session xid = %s status = %d while expecting %d",
				globalSession.Xid, globalSession.Status, meta.GlobalStatusBegin),
		}
	}
	return nil
}

func assertGlobalSessionNotNull(xid string, withBranchSessions bool) (*session.GlobalSession, error) {
	gs := holder.GetSessionHolder().FindGlobalSessionWithBranchSessions(xid, withBranchSessions)
	if gs == nil {
		logging.Logger.Errorf("Could not found global transaction xid = %s", xid)
		return nil, &meta.TransactionException{
			Code:    meta.TransactionExceptionCodeGlobalTransactionNotExist,
			Message: fmt.Sprintf("Could not found global transaction xid = %s", gs.Xid),
		}
	}
	return gs, nil
}

func (core *DefaultCore) BranchReport(branchType meta.BranchType,
	xid string,
	branchId int64,
	status meta.BranchStatus,
	applicationData []byte) error {
	gs, err := assertGlobalSessionNotNull(xid, true)
	if err != nil {
		return nil
	}

	bs := gs.GetBranch(branchId)
	if bs == nil {
		return &meta.TransactionException{
			Code: meta.TransactionExceptionCodeBranchTransactionNotExist,
			Message: fmt.Sprintf("Could not found branch session xid = %s branchId = %d",
				xid, branchId),
		}
	}

	bs.Status = status
	holder.GetSessionHolder().RootSessionManager.UpdateBranchSessionStatus(bs, status)

	logging.Logger.Infof("Successfully branch report xid = %s, branchId = %d", xid, bs.BranchId)
	return nil
}

func (core *DefaultCore) LockQuery(branchType meta.BranchType, resourceId string, xid string, lockKeys string) (bool, error) {
	return true, nil
}

func (core *DefaultCore) branchCommit(globalSession *session.GlobalSession, branchSession *session.BranchSession) (meta.BranchStatus, error) {
	request := protocal.BranchCommitRequest{}
	request.Xid = branchSession.Xid
	request.BranchId = branchSession.BranchId
	request.ResourceId = branchSession.ResourceId
	request.ApplicationData = branchSession.ApplicationData
	request.BranchType = branchSession.BranchType

	resp, err := core.branchCommitSend(request, globalSession, branchSession)
	if err != nil {
		return 0, &meta.TransactionException{
			Code: meta.TransactionExceptionCodeBranchTransactionNotExist,
			Message: fmt.Sprintf("Send branch commit failed, xid = %s branchId = %d",
				branchSession.Xid, branchSession.BranchId),
		}
	}
	return resp, err
}

func (core *DefaultCore) branchCommitSend(request protocal.BranchCommitRequest,
	globalSession *session.GlobalSession, branchSession *session.BranchSession) (meta.BranchStatus, error) {
	resp, err := core.MessageSender.SendSyncRequest(branchSession.ResourceId, branchSession.ClientId, request)
	if err != nil {
		return 0, err
	}
	response := resp.(protocal.BranchCommitResponse)
	return response.BranchStatus, nil
}

func (core *DefaultCore) branchRollback(globalSession *session.GlobalSession, branchSession *session.BranchSession) (meta.BranchStatus, error) {
	request := protocal.BranchRollbackRequest{}
	request.Xid = branchSession.Xid
	request.BranchId = branchSession.BranchId
	request.ResourceId = branchSession.ResourceId
	request.ApplicationData = branchSession.ApplicationData
	request.BranchType = branchSession.BranchType

	resp, err := core.branchRollbackSend(request, globalSession, branchSession)
	if err != nil {
		return 0, &meta.TransactionException{
			Code: meta.TransactionExceptionCodeBranchTransactionNotExist,
			Message: fmt.Sprintf("Send branch rollback failed, xid = %s branchId = %d",
				branchSession.Xid, branchSession.BranchId),
		}
	}
	return resp, err
}

func (core *DefaultCore) branchRollbackSend(request protocal.BranchRollbackRequest,
	globalSession *session.GlobalSession, branchSession *session.BranchSession) (meta.BranchStatus, error) {
	resp, err := core.MessageSender.SendSyncRequest(branchSession.ResourceId, branchSession.ClientId, request)
	if err != nil {
		return 0, err
	}
	response := resp.(protocal.BranchRollbackResponse)
	return response.BranchStatus, nil
}

func (core *DefaultCore) Commit(xid string) (meta.GlobalStatus, error) {
	globalSession := holder.GetSessionHolder().RootSessionManager.FindGlobalSession(xid)
	if globalSession == nil {
		return meta.GlobalStatusFinished, nil
	}
	shouldCommit := func(gs *session.GlobalSession) bool {
		gs.Lock()
		defer gs.Unlock()
		if gs.Active {
			gs.Active = false
		}
		lock.GetLockManager().ReleaseGlobalSessionLock(gs)
		if gs.Status == meta.GlobalStatusBegin {
			changeGlobalSessionStatus(gs, meta.GlobalStatusCommitting)
			return true
		}
		return false
	}(globalSession)

	if !shouldCommit {
		return globalSession.Status, nil
	}

	if globalSession.CanBeCommittedAsync() {
		asyncCommit(globalSession)
		return meta.GlobalStatusCommitted, nil
	} else {
		_, err := core.doGlobalCommit(globalSession, false)
		if err != nil {
			return 0, err
		}
	}

	return globalSession.Status, nil
}

func (core *DefaultCore) doGlobalCommit(globalSession *session.GlobalSession, retrying bool) (bool, error) {
	var (
		success = true
		err     error
	)

	go func() {
		evt := event.NewGlobalTransactionEvent(globalSession.TransactionId, event.RoleTC, globalSession.TransactionName, globalSession.BeginTime, 0, globalSession.Status)
		event.EventBus.GlobalTransactionEventChannel <- evt
	}()

	if globalSession.IsSaga() {
		success, err = core.SAGACore.doGlobalCommit(globalSession, retrying)
	} else {
		for _, bs := range globalSession.GetSortedBranches() {
			if bs.Status == meta.BranchStatusPhaseoneFailed {
				removeBranchSession(globalSession, bs)
				continue
			}
			branchStatus, err1 := core.branchCommit(globalSession, bs)
			if err1 != nil {
				logging.Logger.Errorf("Exception committing branch %v", bs)
				if !retrying {
					queueToRetryCommit(globalSession)
				}
				return false, err1
			}
			switch branchStatus {
			case meta.BranchStatusPhasetwoCommitted:
				removeBranchSession(globalSession, bs)
				continue
			case meta.BranchStatusPhasetwoCommitFailedUnretryable:
				{
					// 二阶段提交失败且不能 Retry，不能异步提交，则移除 GlobalSession，Why?
					if globalSession.CanBeCommittedAsync() {
						logging.Logger.Errorf("By [%s], failed to commit branch %v", bs.Status.String(), bs)
						continue
					} else {
						endCommitFailed(globalSession)
						logging.Logger.Errorf("Finally, failed to commit global[%d] since branch[%d] commit failed", globalSession.Xid, bs.BranchId)
						return false, nil
					}
				}
			default:
				{
					if !retrying {
						queueToRetryCommit(globalSession)
						return false, nil
					}
					if globalSession.CanBeCommittedAsync() {
						logging.Logger.Errorf("By [%s], failed to commit branch %v", bs.Status.String(), bs)
						continue
					} else {
						logging.Logger.Errorf("ResultCodeFailed to commit global[%d] since branch[%d] commit failed, will retry later.", globalSession.Xid, bs.BranchId)
						return false, nil
					}
				}
			}
		}
		if globalSession.HasBranch() {
			logging.Logger.Infof("Global[%d] committing is NOT done.", globalSession.Xid)
			return false, nil
		}
	}
	if success {
		endCommitted(globalSession)

		go func() {
			evt := event.NewGlobalTransactionEvent(globalSession.TransactionId, event.RoleTC, globalSession.TransactionName, globalSession.BeginTime,
				int64(time.CurrentTimeMillis()), globalSession.Status)
			event.EventBus.GlobalTransactionEventChannel <- evt
		}()

		logging.Logger.Infof("Global[%d] committing is successfully done.", globalSession.Xid)
	}
	return success, err
}

func (core *DefaultCore) Rollback(xid string) (meta.GlobalStatus, error) {
	globalSession := holder.GetSessionHolder().RootSessionManager.FindGlobalSession(xid)
	if globalSession == nil {
		return meta.GlobalStatusFinished, nil
	}
	shouldRollBack := func(gs *session.GlobalSession) bool {
		gs.Lock()
		defer gs.Unlock()
		if gs.Active {
			gs.Active = false // Highlight: Firstly, close the session, then no more branch can be registered.
		}
		lock.GetLockManager().ReleaseGlobalSessionLock(gs)
		if gs.Status == meta.GlobalStatusBegin {
			changeGlobalSessionStatus(gs, meta.GlobalStatusRollbacking)
			return true
		}
		return false
	}(globalSession)

	if !shouldRollBack {
		return globalSession.Status, nil
	}

	core.doGlobalRollback(globalSession, false)
	return globalSession.Status, nil
}

func (core *DefaultCore) doGlobalRollback(globalSession *session.GlobalSession, retrying bool) (bool, error) {
	var (
		success = true
		err     error
	)

	go func() {
		evt := event.NewGlobalTransactionEvent(globalSession.TransactionId, event.RoleTC, globalSession.TransactionName, globalSession.BeginTime, 0, globalSession.Status)
		event.EventBus.GlobalTransactionEventChannel <- evt
	}()

	if globalSession.IsSaga() {
		success, err = core.SAGACore.doGlobalRollback(globalSession, retrying)
	} else {
		for _, bs := range globalSession.GetSortedBranches() {
			if bs.Status == meta.BranchStatusPhaseoneFailed {
				removeBranchSession(globalSession, bs)
				continue
			}
			branchStatus, err1 := core.branchRollback(globalSession, bs)
			if err1 != nil {
				logging.Logger.Errorf("Exception rollbacking branch xid=%d branchId=%d", globalSession.Xid, bs.BranchId)
				if !retrying {
					queueToRetryRollback(globalSession)
				}
				return false, err1
			}
			switch branchStatus {
			case meta.BranchStatusPhasetwoRollbacked:
				removeBranchSession(globalSession, bs)
				logging.Logger.Infof("Successfully rollback branch xid=%d branchId=%d", globalSession.Xid, bs.BranchId)
				continue
			case meta.BranchStatusPhasetwoRollbackFailedUnretryable:
				endRollBackFailed(globalSession)
				logging.Logger.Infof("ResultCodeFailed to rollback branch and stop retry xid=%d branchId=%d", globalSession.Xid, bs.BranchId)
				return false, nil
			default:
				logging.Logger.Infof("ResultCodeFailed to rollback branch xid=%d branchId=%d", globalSession.Xid, bs.BranchId)
				if !retrying {
					queueToRetryRollback(globalSession)
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
		gs := holder.GetSessionHolder().RootSessionManager.FindGlobalSession(globalSession.Xid)
		if gs != nil && gs.HasBranch() {
			logging.Logger.Infof("Global[%d] rollbacking is NOT done.", globalSession.Xid)
			return false, nil
		}
	}
	if success {
		endRollbacked(globalSession)

		go func() {
			evt := event.NewGlobalTransactionEvent(globalSession.TransactionId, event.RoleTC, globalSession.TransactionName, globalSession.BeginTime,
				int64(time.CurrentTimeMillis()), globalSession.Status)
			event.EventBus.GlobalTransactionEventChannel <- evt
		}()

		logging.Logger.Infof("Successfully rollback global, xid = %d", globalSession.Xid)
	}
	return success, err
}

func (core *DefaultCore) GetStatus(xid string) (meta.GlobalStatus, error) {
	gs := holder.GetSessionHolder().RootSessionManager.FindGlobalSession(xid)
	if gs == nil {
		return meta.GlobalStatusFinished, nil
	} else {
		return gs.Status, nil
	}
}

func (core *DefaultCore) GlobalReport(xid string, globalStatus meta.GlobalStatus) (meta.GlobalStatus, error) {
	gs := holder.GetSessionHolder().RootSessionManager.FindGlobalSession(xid)
	if gs == nil {
		return globalStatus, nil
	}
	core.doGlobalReport(gs, xid, globalStatus)
	return gs.Status, nil
}

func (core *DefaultCore) doGlobalReport(globalSession *session.GlobalSession, xid string, globalStatus meta.GlobalStatus) error {
	if globalSession.IsSaga() {
		return core.SAGACore.doGlobalReport(globalSession, xid, globalStatus)
	}
	return nil
}

func endRollbacked(globalSession *session.GlobalSession) {
	if isTimeoutGlobalStatus(globalSession.Status) {
		changeGlobalSessionStatus(globalSession, meta.GlobalStatusTimeoutRollbacked)
	} else {
		changeGlobalSessionStatus(globalSession, meta.GlobalStatusRollbacked)
	}
	lock.GetLockManager().ReleaseGlobalSessionLock(globalSession)
	holder.GetSessionHolder().RootSessionManager.RemoveGlobalSession(globalSession)
}

func endRollBackFailed(globalSession *session.GlobalSession) {
	if isTimeoutGlobalStatus(globalSession.Status) {
		changeGlobalSessionStatus(globalSession, meta.GlobalStatusTimeoutRollbackFailed)
	} else {
		changeGlobalSessionStatus(globalSession, meta.GlobalStatusRollbackFailed)
	}
	lock.GetLockManager().ReleaseGlobalSessionLock(globalSession)
	holder.GetSessionHolder().RootSessionManager.RemoveGlobalSession(globalSession)
}

func queueToRetryRollback(globalSession *session.GlobalSession) {
	holder.GetSessionHolder().RetryRollbackingSessionManager.AddGlobalSession(globalSession)
	if isTimeoutGlobalStatus(globalSession.Status) {
		changeGlobalSessionStatus(globalSession, meta.GlobalStatusTimeoutRollbackRetrying)
	} else {
		changeGlobalSessionStatus(globalSession, meta.GlobalStatusRollbackRetrying)
	}
}

func isTimeoutGlobalStatus(status meta.GlobalStatus) bool {
	return status == meta.GlobalStatusTimeoutRollbacked ||
		status == meta.GlobalStatusTimeoutRollbackFailed ||
		status == meta.GlobalStatusTimeoutRollbacking ||
		status == meta.GlobalStatusTimeoutRollbackRetrying
}

func endCommitted(globalSession *session.GlobalSession) {
	changeGlobalSessionStatus(globalSession, meta.GlobalStatusCommitted)
	lock.GetLockManager().ReleaseGlobalSessionLock(globalSession)
	holder.GetSessionHolder().RootSessionManager.RemoveGlobalSession(globalSession)
}

func queueToRetryCommit(globalSession *session.GlobalSession) {
	holder.GetSessionHolder().RetryCommittingSessionManager.AddGlobalSession(globalSession)
	changeGlobalSessionStatus(globalSession, meta.GlobalStatusCommitRetrying)
}

func endCommitFailed(globalSession *session.GlobalSession) {
	changeGlobalSessionStatus(globalSession, meta.GlobalStatusCommitFailed)
	lock.GetLockManager().ReleaseGlobalSessionLock(globalSession)
	holder.GetSessionHolder().RootSessionManager.RemoveGlobalSession(globalSession)
}

func asyncCommit(globalSession *session.GlobalSession) {
	holder.GetSessionHolder().AsyncCommittingSessionManager.AddGlobalSession(globalSession)
	changeGlobalSessionStatus(globalSession, meta.GlobalStatusAsyncCommitting)
}

func changeGlobalSessionStatus(globalSession *session.GlobalSession, status meta.GlobalStatus) {
	globalSession.Status = status
	holder.GetSessionHolder().RootSessionManager.UpdateGlobalSessionStatus(globalSession, status)
}

func removeBranchSession(globalSession *session.GlobalSession, branchSession *session.BranchSession) {
	lock.GetLockManager().ReleaseLock(branchSession)
	globalSession.Remove(branchSession)
	holder.GetSessionHolder().RootSessionManager.RemoveBranchSession(globalSession, branchSession)
}
