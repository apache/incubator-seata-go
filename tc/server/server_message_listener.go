package server

import (
	"github.com/dk-lockdown/seata-golang/base/protocal"
	"github.com/dubbogo/getty"
)

type IServerMessageListener interface {
	OnTrxMessage(rpcMessage protocal.RpcMessage, session getty.Session)

	OnRegRmMessage(request protocal.RpcMessage, session getty.Session)

	OnRegTmMessage(request protocal.RpcMessage, session getty.Session)

	OnCheckMessage(request protocal.RpcMessage, session getty.Session)
}