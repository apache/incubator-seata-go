package server

import (
	"fmt"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/getty"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	getty2 "github.com/dk-lockdown/seata-golang/base/getty"
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/base/protocal"
	"github.com/dk-lockdown/seata-golang/base/protocal/codec"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	time2 "github.com/dk-lockdown/seata-golang/pkg/time"
	"github.com/dk-lockdown/seata-golang/tc/config"
	"github.com/dk-lockdown/seata-golang/tc/event"
	"github.com/dk-lockdown/seata-golang/tc/holder"
	"github.com/dk-lockdown/seata-golang/tc/lock"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

const (
	RPC_REQUEST_TIMEOUT   = 30 * time.Second
	ALWAYS_RETRY_BOUNDARY = 0
)

type DefaultCoordinator struct {
	conf                   config.ServerConfig
	core                   TransactionCoordinator
	idGenerator            atomic.Uint32
	futures                *sync.Map
	timeoutCheckTicker     *time.Ticker
	retryRollbackingTicker *time.Ticker
	retryCommittingTicker  *time.Ticker
	asyncCommittingTicker  *time.Ticker
	undoLogDeleteTicker    *time.Ticker
}

func NewDefaultCoordinator(conf config.ServerConfig) *DefaultCoordinator {
	coordinator := &DefaultCoordinator{
		conf:                   conf,
		idGenerator:            atomic.Uint32{},
		futures:                &sync.Map{},
		timeoutCheckTicker:     time.NewTicker(conf.TimeoutRetryPeriod),
		retryRollbackingTicker: time.NewTicker(conf.RollbackingRetryPeriod),
		retryCommittingTicker:  time.NewTicker(conf.CommittingRetryPeriod),
		asyncCommittingTicker:  time.NewTicker(conf.AsynCommittingRetryPeriod),
		undoLogDeleteTicker:    time.NewTicker(conf.LogDeletePeriod),
	}
	core := NewCore(coordinator)
	coordinator.core = core

	go coordinator.processTimeoutCheck()
	go coordinator.processRetryRollbacking()
	go coordinator.processRetryCommitting()
	go coordinator.processAsyncCommitting()
	go coordinator.processUndoLogDelete()
	return coordinator
}

///////////////////////////////////////////////////
// EventListener
///////////////////////////////////////////////////
func (coordinator *DefaultCoordinator) OnOpen(session getty.Session) error {
	logging.Logger.Infof("got getty_session:%s", session.Stat())
	return nil
}

func (coordinator *DefaultCoordinator) OnError(session getty.Session, err error) {
	SessionManager.ReleaseGettySession(session)
	session.Close()
	logging.Logger.Infof("getty_session{%s} got error{%v}, will be closed.", session.Stat(), err)
}

func (coordinator *DefaultCoordinator) OnClose(session getty.Session) {
	logging.Logger.Info("getty_session{%s} is closing......", session.Stat())
}

func (coordinator *DefaultCoordinator) OnMessage(session getty.Session, pkg interface{}) {
	logging.Logger.Info("received message:{%v}", pkg)
	rpcMessage, ok := pkg.(protocal.RpcMessage)
	if ok {
		_, isRegTM := rpcMessage.Body.(protocal.RegisterTMRequest)
		if isRegTM {
			coordinator.OnRegTmMessage(rpcMessage, session)
			return
		}

		heartBeat, isHeartBeat := rpcMessage.Body.(protocal.HeartBeatMessage)
		if isHeartBeat && heartBeat == protocal.HeartBeatMessagePing {
			coordinator.OnCheckMessage(rpcMessage, session)
			return
		}

		if rpcMessage.MessageType == protocal.MSGTYPE_RESQUEST ||
			rpcMessage.MessageType == protocal.MSGTYPE_RESQUEST_ONEWAY {
			logging.Logger.Debugf("msgId:%s, body:%v", rpcMessage.Id, rpcMessage.Body)
			_, isRegRM := rpcMessage.Body.(protocal.RegisterRMRequest)
			if isRegRM {
				coordinator.OnRegRmMessage(rpcMessage, session)
			} else {
				if SessionManager.IsRegistered(session) {
					defer func() {
						if err := recover(); err != nil {
							logging.Logger.Errorf("Catch Exception while do RPC, request: %v,err: %w", rpcMessage, err)
						}
					}()
					coordinator.OnTrxMessage(rpcMessage, session)
				} else {
					session.Close()
					logging.Logger.Infof("close a unhandled connection! [%v]", session)
				}
			}
		} else {
			resp, loaded := coordinator.futures.Load(rpcMessage.Id)
			if loaded {
				response := resp.(*getty2.MessageFuture)
				response.Response = rpcMessage.Body
				response.Done <- true
				coordinator.futures.Delete(rpcMessage.Id)
			}
		}
	}
}

func (coordinator *DefaultCoordinator) OnCron(session getty.Session) {

}

/////////////////////////////////////////////////////////////
// ServerMessageListener
/////////////////////////////////////////////////////////////
func (coordinator *DefaultCoordinator) OnTrxMessage(rpcMessage protocal.RpcMessage, session getty.Session) {
	rpcContext := SessionManager.GetContextFromIdentified(session)
	logging.Logger.Debugf("server received:%v,clientIp:%s,vgroup:%s", rpcMessage.Body, session.RemoteAddr(), rpcContext.TransactionServiceGroup)

	warpMessage, isWarpMessage := rpcMessage.Body.(protocal.MergedWarpMessage)
	if isWarpMessage {
		resultMessage := protocal.MergeResultMessage{Msgs: make([]protocal.MessageTypeAware, 0)}
		for _, msg := range warpMessage.Msgs {
			resp := coordinator.handleTrxMessage(msg, *rpcContext)
			resultMessage.Msgs = append(resultMessage.Msgs, resp)
		}
		coordinator.SendResponse(rpcMessage, rpcContext.Session, resultMessage)
	} else {
		message := rpcMessage.Body.(protocal.MessageTypeAware)
		resp := coordinator.handleTrxMessage(message, *rpcContext)
		coordinator.SendResponse(rpcMessage, rpcContext.Session, resp)
	}
}

func (coordinator *DefaultCoordinator) handleTrxMessage(msg protocal.MessageTypeAware, ctx RpcContext) protocal.MessageTypeAware {
	switch msg.GetTypeCode() {
	case protocal.TypeGlobalBegin:
		req := msg.(protocal.GlobalBeginRequest)
		resp := coordinator.doGlobalBegin(req, ctx)
		return resp
	case protocal.TypeGlobalStatus:
		req := msg.(protocal.GlobalStatusRequest)
		resp := coordinator.doGlobalStatus(req, ctx)
		return resp
	case protocal.TypeGlobalReport:
		req := msg.(protocal.GlobalReportRequest)
		resp := coordinator.doGlobalReport(req, ctx)
		return resp
	case protocal.TypeGlobalCommit:
		req := msg.(protocal.GlobalCommitRequest)
		resp := coordinator.doGlobalCommit(req, ctx)
		return resp
	case protocal.TypeGlobalRollback:
		req := msg.(protocal.GlobalRollbackRequest)
		resp := coordinator.doGlobalRollback(req, ctx)
		return resp
	case protocal.TypeBranchRegister:
		req := msg.(protocal.BranchRegisterRequest)
		resp := coordinator.doBranchRegister(req, ctx)
		return resp
	case protocal.TypeBranchStatusReport:
		req := msg.(protocal.BranchReportRequest)
		resp := coordinator.doBranchReport(req, ctx)
		return resp
	default:
		return nil
	}
}

func (coordinator *DefaultCoordinator) OnRegRmMessage(rpcMessage protocal.RpcMessage, session getty.Session) {
	message := rpcMessage.Body.(protocal.RegisterRMRequest)

	//version things
	SessionManager.RegisterRmGettySession(message, session)
	logging.Logger.Debugf("checkAuth for client:%s,vgroup:%s,applicationId:%s", session.RemoteAddr(), message.TransactionServiceGroup, message.ApplicationId)

	coordinator.SendResponse(rpcMessage, session, protocal.RegisterRMResponse{AbstractIdentifyResponse: protocal.AbstractIdentifyResponse{Identified: true}})
}

func (coordinator *DefaultCoordinator) OnRegTmMessage(rpcMessage protocal.RpcMessage, session getty.Session) {
	message := rpcMessage.Body.(protocal.RegisterTMRequest)

	//version things
	SessionManager.RegisterTmGettySession(message, session)
	logging.Logger.Debugf("checkAuth for client:%s,vgroup:%s,applicationId:%s", session.RemoteAddr(), message.TransactionServiceGroup, message.ApplicationId)

	coordinator.SendResponse(rpcMessage, session, protocal.RegisterTMResponse{AbstractIdentifyResponse: protocal.AbstractIdentifyResponse{Identified: true}})
}

func (coordinator *DefaultCoordinator) OnCheckMessage(rpcMessage protocal.RpcMessage, session getty.Session) {
	coordinator.SendResponse(rpcMessage, session, protocal.HeartBeatMessagePong)
	logging.Logger.Debugf("received PING from %s", session.RemoteAddr())
}

/////////////////////////////////////////////////////////////
// ServerMessageSender
/////////////////////////////////////////////////////////////
func (coordinator *DefaultCoordinator) SendResponse(request protocal.RpcMessage, session getty.Session, msg interface{}) {
	var ss = session
	_, ok := msg.(protocal.HeartBeatMessage)
	if !ok {
		ss = SessionManager.GetSameClientGettySession(session)
	}
	if ss != nil {
		coordinator.defaultSendResponse(request, ss, msg)
	}
}

func (coordinator *DefaultCoordinator) SendSyncRequest(resourceId string, clientId string, message interface{}) (interface{}, error) {
	return coordinator.SendSyncRequestWithTimeout(resourceId, clientId, message, RPC_REQUEST_TIMEOUT)
}

func (coordinator *DefaultCoordinator) SendSyncRequestWithTimeout(resourceId string, clientId string, message interface{}, timeout time.Duration) (interface{}, error) {
	session, err := SessionManager.GetGettySession(resourceId, clientId)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return coordinator.sendAsyncRequestWithResponse("", session, message, timeout)
}

func (coordinator *DefaultCoordinator) SendSyncRequestByGetty(session getty.Session, message interface{}) (interface{}, error) {
	return coordinator.SendSyncRequestByGettyWithTimeout(session, message, RPC_REQUEST_TIMEOUT)
}

func (coordinator *DefaultCoordinator) SendSyncRequestByGettyWithTimeout(session getty.Session, message interface{}, timeout time.Duration) (interface{}, error) {
	if session == nil {
		return nil, errors.New("rm client is not connected")
	}
	return coordinator.sendAsyncRequestWithResponse("", session, message, timeout)
}

func (coordinator *DefaultCoordinator) SendASyncRequest(session getty.Session, message interface{}) error {
	return coordinator.sendAsyncRequestWithoutResponse(session, message)
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
		logging.Logger.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
	}
	rpcMessage := protocal.RpcMessage{
		Id:          int32(coordinator.idGenerator.Inc()),
		MessageType: protocal.MSGTYPE_RESQUEST_ONEWAY,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	resp := getty2.NewMessageFuture(rpcMessage)
	coordinator.futures.Store(rpcMessage.Id, resp)
	//config timeout
	err = session.WritePkg(rpcMessage, coordinator.conf.GettyConfig.GettySessionParam.TcpWriteTimeout)
	if err != nil {
		coordinator.futures.Delete(rpcMessage.Id)
	}

	if timeout > time.Duration(0) {
		select {
		case <-getty.GetTimeWheel().After(timeout):
			coordinator.futures.Delete(rpcMessage.Id)
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
		Id:         request.Id,
		Codec:      request.Codec,
		Compressor: request.Compressor,
		Body:       msg,
	}
	_, ok := msg.(protocal.HeartBeatMessage)
	if ok {
		resp.MessageType = protocal.MSGTYPE_HEARTBEAT_RESPONSE
	} else {
		resp.MessageType = protocal.MSGTYPE_RESPONSE
	}
	session.WritePkg(resp, time.Duration(0))
}

/////////////////////////////////////////////////////////////
// TCInboundHandler
/////////////////////////////////////////////////////////////
func (coordinator *DefaultCoordinator) doGlobalBegin(request protocal.GlobalBeginRequest, ctx RpcContext) protocal.GlobalBeginResponse {
	var resp = protocal.GlobalBeginResponse{}
	xid, err := coordinator.core.Begin(ctx.ApplicationId, ctx.TransactionServiceGroup, request.TransactionName, request.Timeout)
	if err != nil {
		trxException, ok := err.(meta.TransactionException)
		resp.ResultCode = protocal.ResultCodeFailed
		if ok {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			logging.Logger.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		logging.Logger.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.Xid = xid
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doGlobalStatus(request protocal.GlobalStatusRequest, ctx RpcContext) protocal.GlobalStatusResponse {
	var resp = protocal.GlobalStatusResponse{}
	globalStatus, err := coordinator.core.GetStatus(request.Xid)
	if err != nil {
		trxException, ok := err.(meta.TransactionException)
		resp.ResultCode = protocal.ResultCodeFailed
		if ok {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			logging.Logger.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		logging.Logger.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.GlobalStatus = globalStatus
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doGlobalReport(request protocal.GlobalReportRequest, ctx RpcContext) protocal.GlobalReportResponse {
	var resp = protocal.GlobalReportResponse{}
	globalStatus, err := coordinator.core.GlobalReport(request.Xid, request.GlobalStatus)
	if err != nil {
		trxException, ok := err.(meta.TransactionException)
		resp.ResultCode = protocal.ResultCodeFailed
		if ok {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			logging.Logger.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		logging.Logger.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.GlobalStatus = globalStatus
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doGlobalCommit(request protocal.GlobalCommitRequest, ctx RpcContext) protocal.GlobalCommitResponse {
	var resp = protocal.GlobalCommitResponse{}
	globalStatus, err := coordinator.core.Commit(request.Xid)
	if err != nil {
		trxException, ok := err.(meta.TransactionException)
		resp.ResultCode = protocal.ResultCodeFailed
		if ok {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			logging.Logger.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		logging.Logger.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.GlobalStatus = globalStatus
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doGlobalRollback(request protocal.GlobalRollbackRequest, ctx RpcContext) protocal.GlobalRollbackResponse {
	var resp = protocal.GlobalRollbackResponse{}
	globalStatus, err := coordinator.core.Rollback(request.Xid)
	if err != nil {
		trxException, ok := err.(meta.TransactionException)
		resp.ResultCode = protocal.ResultCodeFailed
		globalSession := holder.GetSessionHolder().FindGlobalSessionWithBranchSessions(request.Xid, false)
		if globalSession == nil {
			resp.GlobalStatus = meta.GlobalStatusFinished
		} else {
			resp.GlobalStatus = globalSession.Status
		}
		if ok {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			logging.Logger.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		logging.Logger.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.GlobalStatus = globalStatus
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doBranchRegister(request protocal.BranchRegisterRequest, ctx RpcContext) protocal.BranchRegisterResponse {
	var resp = protocal.BranchRegisterResponse{}
	branchId, err := coordinator.core.BranchRegister(request.BranchType, request.ResourceId, ctx.ClientId, request.Xid, request.ApplicationData, request.LockKey)
	if err != nil {
		trxException, ok := err.(meta.TransactionException)
		resp.ResultCode = protocal.ResultCodeFailed
		if ok {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			logging.Logger.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		logging.Logger.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.BranchId = branchId
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doBranchReport(request protocal.BranchReportRequest, ctx RpcContext) protocal.BranchReportResponse {
	var resp = protocal.BranchReportResponse{}
	err := coordinator.core.BranchReport(request.BranchType, request.Xid, request.BranchId, request.Status, request.ApplicationData)
	if err != nil {
		trxException, ok := err.(meta.TransactionException)
		resp.ResultCode = protocal.ResultCodeFailed
		if ok {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			logging.Logger.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		logging.Logger.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) doLockCheck(request protocal.GlobalLockQueryRequest, ctx RpcContext) protocal.GlobalLockQueryResponse {
	var resp = protocal.GlobalLockQueryResponse{}
	result, err := coordinator.core.LockQuery(request.BranchType, request.ResourceId, request.Xid, request.LockKey)
	if err != nil {
		trxException, ok := err.(meta.TransactionException)
		resp.ResultCode = protocal.ResultCodeFailed
		if ok {
			resp.TransactionExceptionCode = trxException.Code
			resp.Msg = fmt.Sprintf("TransactionException[%s]", err.Error())
			logging.Logger.Errorf("Catch TransactionException while do RPC, request: %v", request)
			return resp
		}
		resp.Msg = fmt.Sprintf("RuntimeException[%s]", err.Error())
		logging.Logger.Errorf("Catch RuntimeException while do RPC, request: %v", request)
		return resp
	}
	resp.Lockable = result
	resp.ResultCode = protocal.ResultCodeSuccess
	return resp
}

func (coordinator *DefaultCoordinator) processTimeoutCheck() {
	for {
		<-coordinator.timeoutCheckTicker.C
		coordinator.timeoutCheck()
	}
}

func (coordinator *DefaultCoordinator) processRetryRollbacking() {
	for {
		<-coordinator.retryRollbackingTicker.C
		coordinator.handleRetryRollbacking()
	}
}

func (coordinator *DefaultCoordinator) processRetryCommitting() {
	for {
		<-coordinator.retryCommittingTicker.C
		coordinator.handleRetryCommitting()
	}
}

func (coordinator *DefaultCoordinator) processAsyncCommitting() {
	for {
		<-coordinator.asyncCommittingTicker.C
		coordinator.handleAsyncCommitting()
	}
}

func (coordinator *DefaultCoordinator) processUndoLogDelete() {
	for {
		<-coordinator.undoLogDeleteTicker.C
		coordinator.undoLogDelete()
	}
}

func (coordinator *DefaultCoordinator) timeoutCheck() {
	allSessions := holder.GetSessionHolder().RootSessionManager.AllSessions()
	if allSessions == nil && len(allSessions) <= 0 {
		return
	}
	logging.Logger.Debugf("Transaction Timeout Check Begin: %d", len(allSessions))
	for _, globalSession := range allSessions {
		logging.Logger.Debugf("%s %s %d %d", globalSession.Xid, globalSession.Status.String(), globalSession.BeginTime, globalSession.Timeout)
		shouldTimout := func(gs *session.GlobalSession) bool {
			globalSession.Lock()
			defer globalSession.Unlock()
			if globalSession.Status != meta.GlobalStatusBegin || !globalSession.IsTimeout() {
				return false
			}

			if globalSession.Active {
				globalSession.Active = false
			}
			changeGlobalSessionStatus(globalSession, meta.GlobalStatusTimeoutRollbacking)
			evt := event.NewGlobalTransactionEvent(globalSession.TransactionId, event.RoleTC, globalSession.TransactionName, globalSession.BeginTime, 0, globalSession.Status)
			event.EventBus.GlobalTransactionEventChannel <- evt
			return true
		}(globalSession)
		if !shouldTimout {
			continue
		}
		logging.Logger.Infof("Global transaction[%s] is timeout and will be rolled back.", globalSession.Status)
		holder.GetSessionHolder().RetryRollbackingSessionManager.AddGlobalSession(globalSession)
	}
	logging.Logger.Debug("Transaction Timeout Check End.")
}

func (coordinator *DefaultCoordinator) handleRetryRollbacking() {
	rollbackingSessions := holder.GetSessionHolder().RetryRollbackingSessionManager.AllSessions()
	if rollbackingSessions == nil && len(rollbackingSessions) <= 0 {
		return
	}
	now := time2.CurrentTimeMillis()
	for _, rollbackingSession := range rollbackingSessions {
		if rollbackingSession.Status == meta.GlobalStatusRollbacking && !rollbackingSession.IsRollbackingDead() {
			continue
		}
		if isRetryTimeout(int64(now), coordinator.conf.MaxRollbackRetryTimeout, rollbackingSession.BeginTime) {
			if coordinator.conf.RollbackRetryTimeoutUnlockEnable {
				lock.GetLockManager().ReleaseGlobalSessionLock(rollbackingSession)
			}
			holder.GetSessionHolder().RetryRollbackingSessionManager.RemoveGlobalSession(rollbackingSession)
			logging.Logger.Errorf("GlobalSession rollback retry timeout and removed [%s]", rollbackingSession.Xid)
			continue
		}
		_, err := coordinator.core.doGlobalRollback(rollbackingSession, true)
		if err != nil {
			logging.Logger.Infof("Failed to retry rollbacking [%s]", rollbackingSession.Xid)
		}
	}
}

func isRetryTimeout(now int64, timeout int64, beginTime int64) bool {
	if timeout >= ALWAYS_RETRY_BOUNDARY && now-beginTime > timeout {
		return true
	}
	return false
}

func (coordinator *DefaultCoordinator) handleRetryCommitting() {
	committingSessions := holder.GetSessionHolder().RetryCommittingSessionManager.AllSessions()
	if committingSessions == nil && len(committingSessions) <= 0 {
		return
	}
	now := time2.CurrentTimeMillis()
	for _, committingSession := range committingSessions {
		if isRetryTimeout(int64(now), coordinator.conf.MaxCommitRetryTimeout, committingSession.BeginTime) {
			holder.GetSessionHolder().RetryCommittingSessionManager.RemoveGlobalSession(committingSession)
			logging.Logger.Errorf("GlobalSession commit retry timeout and removed [%s]", committingSession.Xid)
			continue
		}
		_, err := coordinator.core.doGlobalCommit(committingSession, true)
		if err != nil {
			logging.Logger.Infof("Failed to retry committing [%s]", committingSession.Xid)
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
			logging.Logger.Infof("Failed to async committing [%s]", asyncCommittingSession.Xid)
		}
	}
}

func (coordinator *DefaultCoordinator) undoLogDelete() {
	saveDays := coordinator.conf.UndoConfig.LogSaveDays
	for key, session := range SessionManager.GetRmSessions() {
		resourceId := key
		deleteRequest := protocal.UndoLogDeleteRequest{
			ResourceId: resourceId,
			SaveDays:   saveDays,
		}
		err := coordinator.SendASyncRequest(session, deleteRequest)
		if err != nil {
			logging.Logger.Errorf("Failed to async delete undo log resourceId = %s", resourceId)
		}
	}
}

func (coordinator *DefaultCoordinator) Stop() {
	coordinator.timeoutCheckTicker.Stop()
	coordinator.retryRollbackingTicker.Stop()
	coordinator.retryCommittingTicker.Stop()
	coordinator.asyncCommittingTicker.Stop()
	coordinator.undoLogDeleteTicker.Stop()
}
