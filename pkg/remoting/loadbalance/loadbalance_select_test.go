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
	assert.NotNil(t, consistentInstance,
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
