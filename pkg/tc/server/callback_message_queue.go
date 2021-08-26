package server

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"sync"
)

type CallbackMessageQueue struct {
	lock *sync.Mutex

	queue []*apis.BranchMessage
}

func NewCallbackMessageQueue() *CallbackMessageQueue {
	return &CallbackMessageQueue{
		queue: make([]*apis.BranchMessage, 0),
		lock:  &sync.Mutex{},
	}
}

func (p *CallbackMessageQueue) Enqueue(msg *apis.BranchMessage) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.queue = append(p.queue, msg)
}

func (p *CallbackMessageQueue) Dequeue() *apis.BranchMessage {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.queue) == 0 {
		return nil
	}
	var msg *apis.BranchMessage
	msg, p.queue = p.queue[0], p.queue[1:]
	return msg
}
