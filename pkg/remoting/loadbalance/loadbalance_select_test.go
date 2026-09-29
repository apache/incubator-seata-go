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
	"fmt"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/protocol/connection"
	"seata.apache.org/seata-go/v2/pkg/remoting/mock"
	"seata.apache.org/seata-go/v2/pkg/remoting/rpc"
)

// newSelectableSessions builds count live sessions with distinct remote
// addresses, so a specific strategy can be told apart from the random fallback.
func newSelectableSessions(ctrl *gomock.Controller, count int) (*sync.Map, []connection.Connection) {
	sessions := &sync.Map{}
	all := make([]connection.Connection, 0, count)
	for i := 0; i < count; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", 8000+i)
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(false).AnyTimes()
		session.EXPECT().RemoteAddr().Return(addr).AnyTimes()
		sessions.Store(session, addr)
		all = append(all, session)
	}
	return sessions, all
}

func containsSession(all []connection.Connection, target connection.Connection) bool {
	for _, session := range all {
		if session == target {
			return true
		}
	}
	return false
}

// TestSelect_LeastActiveLoadBalance checks that Select really routes to the
// LeastActiveLoadBalance implementation. Every session gets a distinct
// in-flight count, so exactly one session is the correct answer and the random
// fallback cannot satisfy the assertion.
func TestSelect_LeastActiveLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	const sessionCount = 8

	sessions, all := newSelectableSessions(ctrl, sessionCount)
	for i := 0; i < sessionCount; i++ {
		rpc.GetStatus(all[i].RemoteAddr()).Active = int32(i)
	}

	for i := 0; i < 50; i++ {
		// Compared by identity: the mocks are structurally identical, so
		// assert.Equal would happily accept any of them.
		got := Select(leastActiveLoadBalance, sessions, "test_xid")
		assert.True(t, got == all[0],
			"Select(%q) must dispatch to LeastActiveLoadBalance, got %v", leastActiveLoadBalance, got)
	}
}

// TestSelect_ConsistentHashLoadBalance checks that Select really routes to the
// ConsistentHashLoadBalance implementation. Building the hash circle is
// observable through the package level cache, which the random fallback never
// populates.
func TestSelect_ConsistentHashLoadBalance(t *testing.T) {
	resetConsistentHashForTest()
	defer resetConsistentHashForTest()

	ctrl := gomock.NewController(t)
	sessions, _ := newSelectableSessions(ctrl, 3)

	got := Select(consistentHashLoadBalance, sessions, "test_xid")
	assert.NotNil(t, got)
	// require, not assert: the ring is dereferenced on the next line.
	require.NotNil(t, consistentInstance,
		"Select(%q) must dispatch to ConsistentHashLoadBalance", consistentHashLoadBalance)
	assert.NotEmpty(t, consistentInstance.sortedHashNodes)

	// The hash circle is deterministic, so the same xid keeps hitting the same session.
	for i := 0; i < 50; i++ {
		assert.True(t, got == Select(consistentHashLoadBalance, sessions, "test_xid"),
			"ConsistentHashLoadBalance must be stable for a given xid")
	}
}

// TestSelect_DeclaredStrategiesAreRoutable guards the dispatch table against
// drifting away from the strategy names declared in this package.
func TestSelect_DeclaredStrategiesAreRoutable(t *testing.T) {
	resetConsistentHashForTest()
	defer resetConsistentHashForTest()

	ctrl := gomock.NewController(t)
	sessions, all := newSelectableSessions(ctrl, 3)

	for _, strategy := range []string{
		randomLoadBalance,
		xidLoadBalance,
		roundRobinLoadBalance,
		consistentHashLoadBalance,
		leastActiveLoadBalance,
	} {
		got := Select(strategy, sessions, "test_xid")
		assert.NotNil(t, got, "Select(%q) returned no session", strategy)
		assert.True(t, containsSession(all, got), "Select(%q) returned a foreign session", strategy)
	}
}

// TestSelect_LeastActiveLoadBalance_ConcurrentWithCounters reproduces the
// production interleaving that LeastActiveLoadBalance now takes part in: one
// goroutine selects while others keep updating the in-flight counters through
// rpc.BeginCount / rpc.EndCount, exactly like SendSync / SendAsync do. Run with
// -race it fails whenever Status.GetActive reads the counter without sync/atomic.
func TestSelect_LeastActiveLoadBalance_ConcurrentWithCounters(t *testing.T) {
	ctrl := gomock.NewController(t)
	const sessionCount = 4

	sessions, all := newSelectableSessions(ctrl, sessionCount)

	const (
		counters = 4
		rounds   = 2000
	)

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i := 0; i < counters; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			addr := all[i].RemoteAddr()
			for j := 0; j < rounds; j++ {
				rpc.BeginCount(addr)
				rpc.EndCount(addr)
			}
		}(i)
	}

	selectorDone := make(chan struct{})
	go func() {
		defer close(selectorDone)
		for {
			select {
			case <-done:
				return
			default:
				Select(leastActiveLoadBalance, sessions, "test_xid")
			}
		}
	}()

	wg.Wait()
	close(done)
	<-selectorDone
}

// TestSelect_ConsistentHashLoadBalance_RebuildsEmptyCircle covers the case where
// the very first selection happens before any session is registered. The circle
// is built through a package level sync.Once, so it used to stay empty forever
// and every later selection degraded to random.
func TestSelect_ConsistentHashLoadBalance_RebuildsEmptyCircle(t *testing.T) {
	resetConsistentHashForTest()
	defer resetConsistentHashForTest()

	ctrl := gomock.NewController(t)
	empty := &sync.Map{}

	assert.Nil(t, Select(consistentHashLoadBalance, empty, "test_xid"))
	require.NotNil(t, consistentInstance)
	assert.True(t, consistentInstance.isEmpty(), "no session means the circle stays empty")

	sessions, all := newSelectableSessions(ctrl, 3)
	got := Select(consistentHashLoadBalance, sessions, "test_xid")

	assert.NotNil(t, got)
	assert.False(t, consistentInstance.isEmpty(),
		"the circle must be rebuilt once sessions appear")
	assert.True(t, containsSession(all, got))
}

// TestSelect_ConsistentHashLoadBalance_WrapsAroundToFirstNode pins the key that
// hashes past the last virtual node: it belongs to the first node of the circle.
// The random fallback would move that key to a different node on every call.
func TestSelect_ConsistentHashLoadBalance_WrapsAroundToFirstNode(t *testing.T) {
	resetConsistentHashForTest()
	defer resetConsistentHashForTest()

	ctrl := gomock.NewController(t)
	sessions, _ := newSelectableSessions(ctrl, 3)

	instance := newConsistenceInstance(sessions)
	require.False(t, instance.isEmpty())

	lastNode := instance.sortedHashNodes[len(instance.sortedHashNodes)-1]
	key := ""
	for i := 0; i < 100000 && key == ""; i++ {
		candidate := fmt.Sprintf("past-last-virtual-node-%d", i)
		if instance.hash(candidate) > lastNode {
			key = candidate
		}
	}
	require.NotEmpty(t, key, "no key hashing past the last virtual node was found")

	first := Select(consistentHashLoadBalance, sessions, key)
	require.NotNil(t, first)
	for i := 0; i < 50; i++ {
		assert.True(t, first == Select(consistentHashLoadBalance, sessions, key),
			"a key past the last virtual node must always wrap to the same node")
	}
}

// TestSelect_ConsistentHashLoadBalance_EmptyKeyIsSpread checks the fallback for
// messages that carry no transaction key at all: they must be spread over the
// circle instead of being pinned to a single node by an empty hash key.
func TestSelect_ConsistentHashLoadBalance_EmptyKeyIsSpread(t *testing.T) {
	resetConsistentHashForTest()
	defer resetConsistentHashForTest()

	ctrl := gomock.NewController(t)
	sessions, all := newSelectableSessions(ctrl, 3)

	distinct := map[connection.Connection]struct{}{}
	for i := 0; i < 200; i++ {
		got := Select(consistentHashLoadBalance, sessions, "")
		require.NotNil(t, got)
		assert.True(t, containsSession(all, got))
		distinct[got] = struct{}{}
	}

	assert.Greater(t, len(distinct), 1,
		"keyless requests must not all be routed to the same node")
}
