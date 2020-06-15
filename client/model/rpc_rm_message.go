package model

import "github.com/xiaobudongzhang/seata-golang/base/protocal"

type RpcRMMessage struct {
	RpcMessage    protocal.RpcMessage
	ServerAddress string
}
