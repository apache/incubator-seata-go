package server

import (
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
)

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

func (coordinator *DefaultCoordinator) SendSyncRequest(resourceID string, clientID string, message interface{}) (interface{}, error) {
	return coordinator.SendSyncRequestWithTimeout(resourceID, clientID, message, RpcRequestTimeout)
}

func (coordinator *DefaultCoordinator) SendSyncRequestWithTimeout(resourceID string, clientID string, message interface{}, timeout time.Duration) (interface{}, error) {
	session, err := SessionManager.GetGettySession(resourceID, clientID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return coordinator.sendAsyncRequestWithResponse(session, message, timeout)
}

func (coordinator *DefaultCoordinator) SendSyncRequestByGetty(session getty.Session, message interface{}) (interface{}, error) {
	return coordinator.SendSyncRequestByGettyWithTimeout(session, message, RpcRequestTimeout)
}

func (coordinator *DefaultCoordinator) SendSyncRequestByGettyWithTimeout(session getty.Session, message interface{}, timeout time.Duration) (interface{}, error) {
	if session == nil {
		return nil, errors.New("rm rpc_client is not connected")
	}
	return coordinator.sendAsyncRequestWithResponse(session, message, timeout)
}

func (coordinator *DefaultCoordinator) SendASyncRequest(session getty.Session, message interface{}) error {
	return coordinator.sendAsyncRequestWithoutResponse(session, message)
}
