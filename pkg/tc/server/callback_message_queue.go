package server

import (
	"sync"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
)

type CallbackMessageQueue struct {
	lock *sync.Mutex

	queue []*apis.BranchMessage

	notify chan struct{}
}

func NewCallbackMessageQueue() *CallbackMessageQueue {
	return &CallbackMessageQueue{
		queue:  make([]*apis.BranchMessage, 0),
		lock:   &sync.Mutex{},
		notify: make(chan struct{}, 3),
	}
}

func (p *CallbackMessageQueue) Enqueue(msg *apis.BranchMessage) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.queue = append(p.queue, msg)

	p.notify <- struct{}{}
}

func (p *CallbackMessageQueue) Dequeue() *apis.BranchMessage {
	p.lock.Lock()
	defer p.lock.Unlock()

	<-p.notify
	if len(p.queue) == 0 {
		return nil
	}

	var msg *apis.BranchMessage
	msg, p.queue = p.queue[0], p.queue[1:]
	return msg
}
