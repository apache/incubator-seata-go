package rpc_client

import "github.com/transaction-wg/seata-golang/pkg/base/protocal"

type RpcRMMessage struct {
	RpcMessage    protocal.RpcMessage
	ServerAddress string
}
