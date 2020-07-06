package server

import (
	"time"
)

import (
	"github.com/dubbogo/getty"
)

import (
	"github.com/dk-lockdown/seata-golang/base/protocal"
)

type ServerMessageSender interface {

	// Send response.
	SendResponse(request protocal.RpcMessage, session getty.Session, msg interface{})

	// Sync call to RM
	SendSyncRequest(resourceId string, clientId string, message interface{}) (interface{}, error)

	// Sync call to RM with timeout.
	SendSyncRequestWithTimeout(resourceId string, clientId string, message interface{}, timeout time.Duration) (interface{}, error)

	// Send request with response object.
	SendSyncRequestByGetty(session getty.Session, message interface{}) (interface{}, error)

	// Send request with response object.
	SendSyncRequestByGettyWithTimeout(session getty.Session, message interface{}, timeout time.Duration) (interface{}, error)

	// ASync call to RM
	SendASyncRequest(session getty.Session, message interface{}) error
}
