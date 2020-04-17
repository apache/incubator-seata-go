package getty

import (
	"github.com/dk-lockdown/seata-golang/base/protocal"
	"time"
)

type IClientMessageSender interface {

	/**
	 * Send msg with response object.
	 *
	 * @param msg the msg
	 * @return the object
	 * @throws TimeoutException the timeout exception
	 */
	SendMsgWithResponse(msg interface{}) (interface{},error)

	/**
	 * Send msg with response object.
	 *
	 * @param msg     the msg
	 * @param timeout the timeout
	 * @return the object
	 * @throws TimeoutException the timeout exception
	 */
	SendMsgWithResponseAndTimeout(msg interface{}, timeout time.Duration) (interface{},error)

	/**
	 * Send msg with response object.
	 *
	 * @param serverAddress the server address
	 * @param msg           the msg
	 * @param timeout       the timeout
	 * @return the object
	 * @throws TimeoutException the timeout exception
	 */
	SendMsgByServerAddressWithResponseAndTimeout(serverAddress string, msg interface{}, timeout time.Duration) (interface{},error)

	/**
	 * Send response.
	 *
	 * @param request       the msg id
	 * @param serverAddress the server address
	 * @param msg           the msg
	 */
	SendResponse(request protocal.RpcMessage, serverAddress string, msg interface{})
}
