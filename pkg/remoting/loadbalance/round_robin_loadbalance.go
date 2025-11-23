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

package loadbalance

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"

	getty "github.com/apache/dubbo-getty"
)

var sequence int32

type rrSnapshot struct {
	sessions []getty.Session
}

type rrSelector struct {
	sessions *sync.Map
	snapshot atomic.Value
	mu       sync.Mutex
}

var selectorCache sync.Map

// getSelector gets or creates the rrSelector for the given sessions.
func getSelector(sessions *sync.Map) *rrSelector {
	if v, ok := selectorCache.Load(sessions); ok {
		return v.(*rrSelector)
	}
	selector := &rrSelector{sessions: sessions}
	selector.snapshot.Store((*rrSnapshot)(nil))
	actual, _ := selectorCache.LoadOrStore(sessions, selector)
	return actual.(*rrSelector)
}

func RoundRobinLoadBalance(sessions *sync.Map, s string) getty.Session {
	selector := getSelector(sessions)
	seq := getPositiveSequence()
	return selector.selectWithSeq(seq)
}

func (r *rrSelector) getValidSnapshot() *rrSnapshot {
	v := r.snapshot.Load()
	if v == nil {
		return nil
	}
	snap := v.(*rrSnapshot)
	if snap == nil || len(snap.sessions) == 0 {
		return nil
	}
	return snap
}

func (r *rrSelector) selectWithSeq(seq int) getty.Session {
	// fast path: use cached snapshot
	if snap := r.getValidSnapshot(); snap != nil {
		n := len(snap.sessions)
		if n > 0 {
			idx := seq % n
			session := snap.sessions[idx]
			if !session.IsClosed() {
				return session
			}
			// selected session is closed, trigger rebuild
		}
	}

	// slow path: rebuild snapshot
	return r.rebuildWithSeq(seq)
}

func (r *rrSelector) rebuildWithSeq(seq int) getty.Session {
	r.mu.Lock()
	defer r.mu.Unlock()

	// double check: another goroutine may have rebuilt during lock acquisition
	if snap := r.getValidSnapshot(); snap != nil {
		n := len(snap.sessions)
		if n > 0 {
			idx := seq % n
			session := snap.sessions[idx]
			if !session.IsClosed() {
				return session
			}
		}
	}

	addrToSession := make(map[string]getty.Session)
	toDelete := make([]interface{}, 0)

	r.sessions.Range(func(key, value interface{}) bool {
		session := key.(getty.Session)
		if session.IsClosed() {
			toDelete = append(toDelete, key)
		} else {
			addr := session.RemoteAddr()
			addrToSession[addr] = session
		}
		return true
	})

	// delete closed sessions synchronously
	for _, k := range toDelete {
		r.sessions.Delete(k)
	}

	if len(addrToSession) == 0 {
		r.snapshot.Store((*rrSnapshot)(nil))
		return nil
	}

	// sort by address to ensure consistent order
	addrs := make([]string, 0, len(addrToSession))
	for addr := range addrToSession {
		addrs = append(addrs, addr)
	}
	sort.Strings(addrs)

	// build session list from sorted addresses
	sessions := make([]getty.Session, len(addrs))
	for i, addr := range addrs {
		sessions[i] = addrToSession[addr]
	}

	// store new snapshot
	snap := &rrSnapshot{sessions: sessions}
	r.snapshot.Store(snap)

	// select session using the same seq
	n := len(sessions)
	idx := seq % n
	return sessions[idx]
}

func getPositiveSequence() int {
	for {
		current := atomic.LoadInt32(&sequence)
		var next int32
		if current == math.MaxInt32 {
			next = 0
		} else {
			next = current + 1
		}
		if atomic.CompareAndSwapInt32(&sequence, current, next) {
			return int(current)
		}
	}
}
