package common

import "github.com/opentrx/seata-golang/v2/pkg/apis"

// MessageFuture ...
type MessageFuture struct {
	ID       int64
	Err      error
	Response interface{}
	Done     chan bool
}

// NewMessageFuture ...
func NewMessageFuture(message *apis.BranchMessage) *MessageFuture {
	return &MessageFuture{
		ID:   message.ID,
		Done: make(chan bool),
	}
}
