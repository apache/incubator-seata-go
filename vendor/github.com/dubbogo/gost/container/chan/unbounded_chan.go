/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gxchan

import (
	"go.uber.org/atomic"
)

import (
	"github.com/dubbogo/gost/container/queue"
)

// UnboundedChan is a chan that could grow if the number of elements exceeds the capacity.
type UnboundedChan struct {
	in       chan interface{}
	out      chan interface{}
	queue    *gxqueue.CircularUnboundedQueue
	queueLen *atomic.Int32
	queueCap *atomic.Int32
}

// NewUnboundedChan creates an instance of UnboundedChan.
func NewUnboundedChan(capacity int) *UnboundedChan {
	return NewUnboundedChanWithQuota(capacity, 0)
}

func NewUnboundedChanWithQuota(capacity, quota int) *UnboundedChan {
	if capacity <= 0 {
		panic("capacity should be greater than 0")
	}
	if quota < 0 {
		panic("quota should be greater or equal to 0")
	}
	if quota != 0 && capacity > quota {
		capacity = quota
	}

	var (
		incap  = capacity / 3
		outcap = capacity / 3
		qcap   = capacity - 2*(capacity/3)
		qquota = quota - 2*(capacity/3)
	)

	if capacity/3 > 0 {
		incap--
	} else {
		qcap--
		qquota--
	}

	// address quota if the value is not valid
	if quota == 0 { // quota == 0 means no limits for queue
		qquota = 0
	} else { // quota != 0 means chan couldn't grow unlimitedly
		if qquota == 0 {
			// qquota == 0 means queue could grow unlimitedly
			// in this case, the total quota will be set to quota+1
			qquota = 1
		}
	}

	ch := &UnboundedChan{
		in:       make(chan interface{}, incap),
		out:      make(chan interface{}, outcap),
		queue:    gxqueue.NewCircularUnboundedQueueWithQuota(qcap, qquota),
		queueLen: &atomic.Int32{},
		queueCap: &atomic.Int32{},
	}
	ch.queueCap.Store(int32(ch.queue.Cap()))

	go ch.run()

	return ch
}

// In returns write-only chan
func (ch *UnboundedChan) In() chan<- interface{} {
	return ch.in
}

// Out returns read-only chan
func (ch *UnboundedChan) Out() <-chan interface{} {
	return ch.out
}

// Len returns the total length of chan
func (ch *UnboundedChan) Len() int {
	return len(ch.in) + len(ch.out) + int(ch.queueLen.Load())
}

// Cap returns the total capacity of chan.
func (ch *UnboundedChan) Cap() int {
	return cap(ch.in) + cap(ch.out) + int(ch.queueCap.Load()) + 1
}

func (ch *UnboundedChan) run() {
	defer func() {
		close(ch.out)
	}()

	for {
		val, ok := <-ch.in
		if !ok { // `ch.in` was closed and queue has no elements
			return
		}

		select {
		case ch.out <- val: // data was written to `ch.out`
			continue
		default: // `ch.out` is full, move the data to `ch.queue`
			if ok := ch.queuePush(val); !ok {
				ch.block(val)
			}
		}

		for !ch.queue.IsEmpty() {
			select {
			case val, ok := <-ch.in: // `ch.in` was closed
				if !ok {
					ch.closeWait()
					return
				}
				if ok = ch.queuePush(val); !ok { // try to push the value into queue
					ch.block(val)
				}
			case ch.out <- ch.queue.Peek():
				ch.queuePop()
			}
		}

		if ch.queue.Cap() > ch.queue.InitialCap() {
			ch.queueReset()
		}
	}
}

// closeWait waits for being empty of `ch.queue`
func (ch *UnboundedChan) closeWait() {
	for !ch.queue.IsEmpty() {
		ch.out <- ch.queuePop()
	}
}

// block waits for having an idle space on `ch.out`
func (ch *UnboundedChan) block(val interface{}) {
	// `val` is not in `ch.queue` and `ch.in`, but it is stored into `UnboundedChan`
	defer func() {
		ch.queueLen.Add(-1)
	}()
	ch.queueLen.Add(1)

	if !ch.queue.IsEmpty() {
		ch.out <- ch.queue.Peek()
		ch.queue.Pop()
		ch.queue.Push(val)
		return
	}

	ch.out <- val
}

func (ch *UnboundedChan) queuePush(val interface{}) (ok bool) {
	ok = ch.queue.Push(val)
	if ok {
		ch.queueLen.Add(1)
		ch.queueCap.Store(int32(ch.queue.Cap()))
	}
	return
}

func (ch *UnboundedChan) queueReset() {
	ch.queue.Reset()
	ch.queueCap.Store(int32(ch.queue.Cap()))
}

func (ch *UnboundedChan) queuePop() (t interface{}) {
	t = ch.queue.Pop()
	ch.queueLen.Add(-1)
	return
}
