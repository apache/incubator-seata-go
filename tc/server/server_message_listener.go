package server

import (
	getty "github.com/apache/dubbo-getty"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
)

type ServerMessageListener interface {
	OnTrxMessage(rpcMessage protocal.RpcMessage, session getty.Session)

	OnRegRmMessage(request protocal.RpcMessage, session getty.Session)

	OnRegTmMessage(request protocal.RpcMessage, session getty.Session)

	OnCheckMessage(request protocal.RpcMessage, session getty.Session)
}
