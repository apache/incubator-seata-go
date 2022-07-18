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

package gxqueue

const (
	fastGrowThreshold = 1024
)

// CircularUnboundedQueue is a circular structure and will grow automatically if it exceeds the capacity.
// CircularUnboundedQueue is not thread-safe.
type CircularUnboundedQueue struct {
	data       []interface{}
	head, tail int
	icap       int // initial capacity
	quota      int // specify the maximum size of the queue, setting to 0 denotes unlimited.
}

func NewCircularUnboundedQueue(capacity int) *CircularUnboundedQueue {
	return NewCircularUnboundedQueueWithQuota(capacity, 0)
}

func NewCircularUnboundedQueueWithQuota(capacity, quota int) *CircularUnboundedQueue {
	if capacity < 0 {
		panic("capacity should be greater than zero")
	}
	if quota < 0 {
		panic("quota should be greater or equal to zero")
	}
	if quota != 0 && capacity > quota {
		capacity = quota
	}
	return &CircularUnboundedQueue{
		data:  make([]interface{}, capacity+1),
		icap:  capacity,
		quota: quota,
	}
}

func (q *CircularUnboundedQueue) IsEmpty() bool {
	return q.head == q.tail
}

func (q *CircularUnboundedQueue) Push(t interface{}) bool {
	if nextTail := (q.tail + 1) % len(q.data); nextTail != q.head {
		q.data[q.tail] = t
		q.tail = nextTail
		return true
	}

	if q.grow() {
		// grow succeed
		q.data[q.tail] = t
		q.tail = (q.tail + 1) % len(q.data)
		return true
	}

	return false
}

func (q *CircularUnboundedQueue) Pop() interface{} {
	if q.IsEmpty() {
		panic("queue has no element")
	}

	t := q.data[q.head]
	q.head = (q.head + 1) % len(q.data)

	return t
}

func (q *CircularUnboundedQueue) Peek() interface{} {
	if q.IsEmpty() {
		panic("queue has no element")
	}
	return q.data[q.head]
}

func (q *CircularUnboundedQueue) Cap() int {
	return len(q.data) - 1
}

func (q *CircularUnboundedQueue) Len() int {
	head, tail := q.head, q.tail
	if head > tail {
		tail += len(q.data)
	}
	return tail - head
}

func (q *CircularUnboundedQueue) Reset() {
	q.data = make([]interface{}, q.icap+1)
	q.head, q.tail = 0, 0
}

func (q *CircularUnboundedQueue) InitialCap() int {
	return q.icap
}

func (q *CircularUnboundedQueue) grow() bool {
	oldcap := q.Cap()
	if oldcap == 0 {
		oldcap++
	}
	var newcap int
	if oldcap < fastGrowThreshold {
		newcap = oldcap * 2
	} else {
		newcap = oldcap + oldcap/4
	}
	if q.quota != 0 && newcap > q.quota {
		newcap = q.quota
	}

	if newcap == q.Cap() {
		return false
	}

	newdata := make([]interface{}, newcap+1)
	copy(newdata[0:], q.data[q.head:])
	if q.head > q.tail {
		copy(newdata[len(q.data)-q.head:], q.data[:q.head-1])
	}

	q.head, q.tail = 0, q.Cap()
	q.data = newdata

	return true
}
