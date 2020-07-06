package getty

import "github.com/dk-lockdown/seata-golang/base/protocal"

// MessageFuture ...
type MessageFuture struct {
	Id       int32
	Err      error
	Response interface{}
	Done     chan bool
}

// NewMessageFuture ...
func NewMessageFuture(message protocal.RpcMessage) *MessageFuture {
	return &MessageFuture{
		Id:   message.Id,
		Done: make(chan bool),
	}
}
