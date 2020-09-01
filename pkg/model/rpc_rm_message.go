package model

import "github.com/transaction-wg/seata-golang/base/protocal"

type RpcRMMessage struct {
	RpcMessage    protocal.RpcMessage
	ServerAddress string
}
