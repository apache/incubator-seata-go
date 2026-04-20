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

	"seata.apache.org/seata-go/v2/pkg/remoting/mock"
	"seata.apache.org/seata-go/v2/pkg/remoting/rpc"
)

// TestSelect_RoutesConsistentHash pins down the P7 regression: configuring
// "ConsistentHashLoadBalance" must actually dispatch to the consistent-hash
// strategy, not silently fall through to random.
func TestSelect_RoutesConsistentHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	for i := 0; i < 3; i++ {
		addr := fmt.Sprintf("127.0.0.1:80%02d", i)
		s := mock.NewMockTestSession(ctrl)
		s.EXPECT().IsClosed().AnyTimes().Return(false)
		s.EXPECT().RemoteAddr().AnyTimes().Return(addr)
		sessions.Store(s, addr)
	}

	c := NewConsistent(0)

	first := Select(consistentHashLoadBalance, sessions, "xid-A", c)
	assert.NotNil(t, first)

	// Consistent hashing must be deterministic for the same xid+ring.
	for i := 0; i < 5; i++ {
		again := Select(consistentHashLoadBalance, sessions, "xid-A", c)
		assert.Equal(t, first.RemoteAddr(), again.RemoteAddr())
	}
}

// TestSelect_RoutesLeastActive pins the other half of the P7 fix.
func TestSelect_RoutesLeastActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	busyAddr := "127.0.0.1:8100"
	idleAddr := "127.0.0.1:8101"

	busy := mock.NewMockTestSession(ctrl)
	busy.EXPECT().IsClosed().AnyTimes().Return(false)
	busy.EXPECT().RemoteAddr().AnyTimes().Return(busyAddr)
	sessions.Store(busy, busyAddr)

	idle := mock.NewMockTestSession(ctrl)
	idle.EXPECT().IsClosed().AnyTimes().Return(false)
	idle.EXPECT().RemoteAddr().AnyTimes().Return(idleAddr)
	sessions.Store(idle, idleAddr)

	// Drive busy's active count higher than idle's.
	rpc.BeginCount(busyAddr)
	rpc.BeginCount(busyAddr)
	defer func() {
		rpc.RemoveStatus(busyAddr)
		rpc.RemoveStatus(idleAddr)
	}()

	result := Select(leastActiveLoadBalance, sessions, "any-xid", nil)
	assert.Equal(t, idleAddr, result.RemoteAddr())
}

// TestSelect_UnknownFallsBackToRandom documents that an unrecognized
// strategy keeps the historical behaviour of random selection.
func TestSelect_UnknownFallsBackToRandom(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	s := mock.NewMockTestSession(ctrl)
	s.EXPECT().IsClosed().AnyTimes().Return(false)
	sessions.Store(s, "only")

	result := Select("NotARealBalancer", sessions, "xid", nil)
	assert.NotNil(t, result)
}
