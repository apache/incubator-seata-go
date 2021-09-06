package server

import (
	"sync"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	getty2 "github.com/transaction-wg/seata-golang/pkg/base/getty"
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal/codec"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"github.com/transaction-wg/seata-golang/pkg/tc/event"
	"github.com/transaction-wg/seata-golang/pkg/tc/holder"
	"github.com/transaction-wg/seata-golang/pkg/tc/lock"
	"github.com/transaction-wg/seata-golang/pkg/tc/session"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/runtime"
	time2 "github.com/transaction-wg/seata-golang/pkg/util/time"
)

const (
	RpcRequestTimeout   = 30 * time.Second
	AlwaysRetryBoundary = 0
)

type DefaultCoordinator struct {
	conf                   *config.ServerConfig
	core                   TransactionCoordinator
	idGenerator            *atomic.Uint32
	futures                *sync.Map
}

func NewDefaultCoordinator(conf *config.ServerConfig) *DefaultCoordinator {
	coordinator := &DefaultCoordinator{
		conf:                   conf,
		idGenerator:            &atomic.Uint32{},
		futures:                &sync.Map{},
	}
	core := NewCore(coordinator)
	coordinator.core = core

	go coordinator.processTimeoutCheck()
	go coordinator.processRetryRollingBack()
	go coordinator.processRetryCommitting()
	go coordinator.processAsyncCommitting()
	go coordinator.processUndoLogDelete()
	return coordinator
}

func (coordinator *DefaultCoordinator) sendAsyncRequestWithResponse(address string, session getty.Session, msg interface{}, timeout time.Duration) (interface{}, error) {
	if timeout <= time.Duration(0) {
		return nil, errors.New("timeout should more than 0ms")
	}
	return coordinator.sendAsyncRequest(address, session, msg, timeout)
}

func (coordinator *DefaultCoordinator) sendAsyncRequestWithoutResponse(session getty.Session, msg interface{}) error {
	_, err := coordinator.sendAsyncRequest("", session, msg, time.Duration(0))
	return err
}

func (coordinator *DefaultCoordinator) sendAsyncRequest(address string, session getty.Session, msg interface{}, timeout time.Duration) (interface{}, error) {
	var err error
	if session == nil {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
	}
	rpcMessage := protocal.RpcMessage{
		ID:          int32(coordinator.idGenerator.Inc()),
		MessageType: protocal.MSGTypeRequestOneway,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	resp := getty2.NewMessageFuture(rpcMessage)
	coordinator.futures.Store(rpcMessage.ID, resp)
	//config timeout
	pkgLen, sendLen, err := session.WritePkg(rpcMessage, coordinator.conf.GettyConfig.GettySessionParam.TCPWriteTimeout)
	if err != nil || (pkgLen != 0 && pkgLen != sendLen) {
		log.Warnf("start to close the session because %d of %d bytes data is sent success. err:%+v", sendLen, pkgLen, err)
		coordinator.futures.Delete(rpcMessage.ID)
		runtime.GoWithRecover(func() {
			session.Close()
		}, nil)
		return nil, errors.Wrap(err, "pkg not send completely!")
	}

	if timeout > time.Duration(0) {
		select {
		case <-getty.GetTimeWheel().After(timeout):
			coordinator.futures.Delete(rpcMessage.ID)
			return nil, errors.Errorf("wait response timeout,ip:%s,request:%v", address, rpcMessage)
		case <-resp.Done:
			err = resp.Err
		}
		return resp.Response, err
	}
	return nil, err
}

func (coordinator *DefaultCoordinator) defaultSendResponse(request protocal.RpcMessage, session getty.Session, msg interface{}) {
	resp := protocal.RpcMessage{
		ID:         request.ID,
		Codec:      request.Codec,
		Compressor: request.Compressor,
		Body:       msg,
	}
	_, ok := msg.(protocal.HeartBeatMessage)
	if ok {
		resp.MessageType = protocal.MSGTypeHeartbeatResponse
	} else {
		resp.MessageType = protocal.MSGTypeResponse
	}
	pkgLen, sendLen, err := session.WritePkg(resp, time.Duration(0))
	if err != nil || (pkgLen != 0 && pkgLen != sendLen) {
		log.Warnf("start to close the session because %d of %d bytes data is sent success. err:%+v", sendLen, pkgLen, err)
		runtime.GoWithRecover(func() {
			session.Close()
		}, nil)
	}
}

func (coordinator *DefaultCoordinator) processTimeoutCheck() {
	for {
		timer := time.NewTimer(coordinator.conf.TimeoutRetryPeriod)
		select {
		case <-timer.C:
			coordinator.timeoutCheck()
		}
		timer.Stop()
	}
}

func (coordinator *DefaultCoordinator) processRetryRollingBack() {
	for {
		timer := time.NewTimer(coordinator.conf.RollingBackRetryPeriod)
		select {
		case <-timer.C:
			coordinator.handleRetryRollingBack()
		}
		timer.Stop()
	}
}

func (coordinator *DefaultCoordinator) processRetryCommitting() {
	for {
		timer := time.NewTimer(coordinator.conf.CommittingRetryPeriod)
		select {
		case <-timer.C:
			coordinator.handleRetryCommitting()
		}
		timer.Stop()
	}
}

func (coordinator *DefaultCoordinator) processAsyncCommitting() {
	for {
		timer := time.NewTimer(coordinator.conf.AsyncCommittingRetryPeriod)
		select {
		case <-timer.C:
			coordinator.handleAsyncCommitting()
		}
		timer.Stop()
	}
}

func (coordinator *DefaultCoordinator) processUndoLogDelete() {
	for {
		timer := time.NewTimer(coordinator.conf.LogDeletePeriod)
		select {
		case <-timer.C:
			coordinator.undoLogDelete()
		}
		timer.Stop()
	}
}

func (coordinator *DefaultCoordinator) timeoutCheck() {
	allSessions := holder.GetSessionHolder().RootSessionManager.AllSessions()
	if allSessions == nil && len(allSessions) <= 0 {
		return
	}
	log.Debugf("Transaction Timeout Check Begin: %d", len(allSessions))
	for _, globalSession := range allSessions {
		log.Debugf("%s %s %d %d", globalSession.XID, globalSession.Status.String(), globalSession.BeginTime, globalSession.Timeout)
		shouldTimout := func(gs *session.GlobalSession) bool {
			globalSession.Lock()
			defer globalSession.Unlock()
			if globalSession.Status != meta.GlobalStatusBegin || !globalSession.IsTimeout() {
				return false
			}

			if globalSession.Active {
				globalSession.Active = false
			}
			changeGlobalSessionStatus(globalSession, meta.GlobalStatusTimeoutRollingBack)
			evt := event.NewGlobalTransactionEvent(globalSession.TransactionID, event.RoleTC, globalSession.TransactionName, globalSession.BeginTime, 0, globalSession.Status)
			event.EventBus.GlobalTransactionEventChannel <- evt
			return true
		}(globalSession)
		if !shouldTimout {
			continue
		}
		log.Infof("Global transaction[%s] is timeout and will be rolled back.", globalSession.Status)
		holder.GetSessionHolder().RetryRollbackingSessionManager.AddGlobalSession(globalSession)
	}
	log.Debug("Transaction Timeout Check End.")
}

func (coordinator *DefaultCoordinator) handleRetryRollingBack() {
	rollingBackSessions := holder.GetSessionHolder().RetryRollbackingSessionManager.AllSessions()
	if rollingBackSessions == nil && len(rollingBackSessions) <= 0 {
		return
	}
	now := time2.CurrentTimeMillis()
	for _, rollingBackSession := range rollingBackSessions {
		if rollingBackSession.Status == meta.GlobalStatusRollingBack && !rollingBackSession.IsRollbackingDead() {
			continue
		}
		if isRetryTimeout(int64(now), coordinator.conf.MaxRollbackRetryTimeout, rollingBackSession.BeginTime) {
			if coordinator.conf.RollbackRetryTimeoutUnlockEnable {
				lock.GetLockManager().ReleaseGlobalSessionLock(rollingBackSession)
			}
			holder.GetSessionHolder().RetryRollbackingSessionManager.RemoveGlobalSession(rollingBackSession)
			log.Errorf("GlobalSession rollback retry timeout and removed [%s]", rollingBackSession.XID)
			continue
		}
		_, err := coordinator.core.doGlobalRollback(rollingBackSession, true)
		if err != nil {
			log.Infof("Failed to retry rolling back [%s]", rollingBackSession.XID)
		}
	}
}

func isRetryTimeout(now int64, timeout int64, beginTime int64) bool {
	if timeout >= AlwaysRetryBoundary && now-beginTime > timeout {
		return true
	}
	return false
}

func (coordinator *DefaultCoordinator) handleRetryCommitting() {
	ssMgr := holder.GetSessionHolder().RetryCommittingSessionManager
	committingSessions := ssMgr.AllSessions()
	if committingSessions == nil && len(committingSessions) <= 0 {
		return
	}
	now := time2.CurrentTimeMillis()
	for _, committingSession := range committingSessions {
		if isRetryTimeout(int64(now), coordinator.conf.MaxCommitRetryTimeout, committingSession.BeginTime) {
			ssMgr.RemoveGlobalSession(committingSession)
			log.Errorf("GlobalSession commit retry timeout and removed [%s]", committingSession.XID)
			continue
		}
		_, err := coordinator.core.doGlobalCommit(committingSession, true)
		if err != nil {
			log.Infof("Failed to retry committing [%s]", committingSession.XID)
		}
	}
}

func (coordinator *DefaultCoordinator) handleAsyncCommitting() {
	asyncCommittingSessions := holder.GetSessionHolder().AsyncCommittingSessionManager.AllSessions()
	if asyncCommittingSessions == nil && len(asyncCommittingSessions) <= 0 {
		return
	}
	for _, asyncCommittingSession := range asyncCommittingSessions {
		if asyncCommittingSession.Status != meta.GlobalStatusAsyncCommitting {
			continue
		}
		_, err := coordinator.core.doGlobalCommit(asyncCommittingSession, true)
		if err != nil {
			log.Infof("Failed to async committing [%s]", asyncCommittingSession.XID)
		}
	}
}

func (coordinator *DefaultCoordinator) undoLogDelete() {
	saveDays := coordinator.conf.UndoConfig.LogSaveDays
	for key, session := range SessionManager.GetRmSessions() {
		resourceID := key
		deleteRequest := protocal.UndoLogDeleteRequest{
			ResourceID: resourceID,
			SaveDays:   saveDays,
		}
		err := coordinator.SendASyncRequest(session, deleteRequest)
		if err != nil {
			log.Errorf("Failed to async delete undo log resourceID = %s", resourceID)
		}
	}
}

func (coordinator *DefaultCoordinator) Stop() {
}
