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

type IServerMessageSender interface {
	/**
	 * Send response.
	 *
	 * @param request the request
	 * @param channel the channel
	 * @param msg     the msg
	 */

	SendResponse(request protocal.RpcMessage, session getty.Session, msg interface{})

	/**
	 * Sync call to RM
	 *
	 * @param resourceId Resource ID
	 * @param clientId   Client ID
	 * @param message    Request message
	 * @return Response message
	 * @throws IOException .
	 * @throws TimeoutException the timeout exception
	 */
	SendSyncRequest(resourceId string, clientId string, message interface{}) (interface{},error)

	/**
	 * Sync call to RM with timeout.
	 *
	 * @param resourceId Resource ID
	 * @param clientId   Client ID
	 * @param message    Request message
	 * @param timeout    timeout of the call
	 * @return Response message
	 * @throws IOException .
	 * @throws TimeoutException the timeout exception
	 */
	SendSyncRequestWithTimeout(resourceId string, clientId string, message interface{}, timeout time.Duration) (interface{},error)

	/**
	 * Send request with response object.
	 * send syn request for rm
	 *
	 * @param clientChannel the client channel
	 * @param message       the message
	 * @return the object
	 * @throws TimeoutException the timeout exception
	 */
	SendSyncRequestByGettySession(session getty.Session, message interface{}) (interface{},error)

	/**
	 * Send request with response object.
	 * send syn request for rm
	 *
	 * @param clientChannel the client channel
	 * @param message       the message
	 * @param timeout       the timeout
	 * @return the object
	 * @throws TimeoutException the timeout exception
	 */
	SendSyncRequestByGettySessionWithTimeout(session getty.Session, message interface{}, timeout time.Duration) (interface{},error)

	/**
	 * ASync call to RM
	 *
	 * @param channel   channel
	 * @param message    Request message
	 * @return Response message
	 * @throws IOException .
	 * @throws TimeoutException the timeout exception
	 */
	SendASyncRequest(session getty.Session, message interface{}) error
}
