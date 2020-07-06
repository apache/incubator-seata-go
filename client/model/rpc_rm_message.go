package model

import "github.com/dk-lockdown/seata-golang/base/protocal"

type RpcRMMessage struct {
	RpcMessage    protocal.RpcMessage
	ServerAddress string
}
