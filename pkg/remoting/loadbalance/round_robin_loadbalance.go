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

// RoundRobinConfig holds configuration for round-robin load balancing
type RoundRobinConfig struct {
	MaxRetries int
}

// DefaultRoundRobinConfig provides sensible defaults
var DefaultRoundRobinConfig = RoundRobinConfig{
	MaxRetries: 3, // Balanced default: enough attempts without excessive overhead
}

type rrSnapshot struct {
	sessions []getty.Session
}

type rrSelector struct {
	sessions   *sync.Map
	snapshot   atomic.Value // stores *rrSnapshot
	mu         sync.Mutex
	maxRetries int
}

// RoundRobinLoadBalance selects a session using round-robin algorithm with default config
func RoundRobinLoadBalance(sessions *sync.Map, s string) getty.Session {
	return RoundRobinLoadBalanceWithConfig(sessions, s, DefaultRoundRobinConfig)
}

func RoundRobinLoadBalanceWithConfig(sessions *sync.Map, s string, config RoundRobinConfig) getty.Session {
	maxRetries := config.MaxRetries

	// Validate and normalize
	// Note: 0 is valid (means no retry, directly rebuild on first failure)
	if maxRetries < 0 {
		maxRetries = DefaultRoundRobinConfig.MaxRetries
	}
	if maxRetries > 10 {
		maxRetries = 10
	}

	selector := &rrSelector{
		sessions:   sessions,
		maxRetries: maxRetries,
	}
	selector.snapshot.Store((*rrSnapshot)(nil))

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
	// if maxRetries is 0, skip retry loop entirely
	if r.maxRetries == 0 {
		snap := r.getValidSnapshot()
		if snap != nil && len(snap.sessions) > 0 {
			idx := seq % len(snap.sessions)
			session := snap.sessions[idx]
			if !session.IsClosed() {
				return session
			}
		}
		// first attempt failed, rebuild immediately
		return r.rebuildWithSeq(seq)
	}

	for retry := 0; retry < r.maxRetries; retry++ {
		snap := r.getValidSnapshot()
		if snap != nil {
			n := len(snap.sessions)
			if n > 0 {
				idx := (seq + retry) % n
				session := snap.sessions[idx]
				if !session.IsClosed() {
					return session
				}
			}
		}

		if retry < r.maxRetries-1 && snap != nil {
			continue
		}

		break
	}

	return r.rebuildWithSeq(seq)
}

func (r *rrSelector) rebuildWithSeq(seq int) getty.Session {
	r.mu.Lock()
	defer r.mu.Unlock()

	snap := r.getValidSnapshot()
	if snap != nil {
		n := len(snap.sessions)
		if n > 0 {
			// try to find an open session starting from the calculated index
			for i := 0; i < n; i++ {
				idx := (seq + i) % n
				session := snap.sessions[idx]
				if !session.IsClosed() {
					return session
				}
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
	newSnap := &rrSnapshot{sessions: sessions}
	r.snapshot.Store(newSnap)

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
