package server

import (
	"github.com/dubbogo/getty"
)

import (
	"github.com/dk-lockdown/seata-golang/base/protocal"
)

type IServerMessageListener interface {
	OnTrxMessage(rpcMessage protocal.RpcMessage, session getty.Session)

	OnRegRmMessage(request protocal.RpcMessage, session getty.Session)

	OnRegTmMessage(request protocal.RpcMessage, session getty.Session)

	OnCheckMessage(request protocal.RpcMessage, session getty.Session)
}