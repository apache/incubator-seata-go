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

func setupSessions(ctrl *gomock.Controller, count int, isClosed bool) (*sync.Map, []string) {
	sessions := &sync.Map{}
	var addrs []string
	for i := 0; i < count; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(isClosed).AnyTimes()
		addr := fmt.Sprintf("127.0.0.1:%d", 8080+i)
		session.EXPECT().RemoteAddr().Return(addr).AnyTimes()
		sessions.Store(session, "session")
		addrs = append(addrs, addr)
	}
	return sessions, addrs
}

func TestSelect_RandomLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Just 1 eligible session so random is deterministic for assertion
	sessions, addrs := setupSessions(ctrl, 1, false)
	conn := Select(randomLoadBalance, sessions, "test_xid")
	assert.NotNil(t, conn)
	assert.Equal(t, addrs[0], conn.RemoteAddr())
}

func TestSelect_XidLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions, addrs := setupSessions(ctrl, 3, false)
	// XID format is ip:port:transactionId
	xid := addrs[1] + ":tx123"
	conn := Select(xidLoadBalance, sessions, xid)
	assert.NotNil(t, conn)
	assert.Equal(t, addrs[1], conn.RemoteAddr())
}

func TestSelect_RoundRobinLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions, _ := setupSessions(ctrl, 3, false)

	conn1 := Select(roundRobinLoadBalance, sessions, "test_xid")
	assert.NotNil(t, conn1)
	conn2 := Select(roundRobinLoadBalance, sessions, "test_xid")
	assert.NotNil(t, conn2)
	assert.NotEqual(t, conn1.RemoteAddr(), conn2.RemoteAddr())
}

func TestSelect_ConsistentHashLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions, _ := setupSessions(ctrl, 3, false)

	resetConsistentHashForTest()
	conn1 := Select(consistentHashLoadBalance, sessions, "test_xid")
	assert.NotNil(t, conn1)

	for i := 0; i < 10; i++ {
		conn2 := Select(consistentHashLoadBalance, sessions, "test_xid")
		assert.NotNil(t, conn2)
		assert.Equal(t, conn1.RemoteAddr(), conn2.RemoteAddr())
	}
}

func TestSelect_LeastActiveLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions, addrs := setupSessions(ctrl, 3, false)

	// Mark all but the first address as busy
	for i := 1; i < len(addrs); i++ {
		rpc.BeginCount(addrs[i])
		defer rpc.EndCount(addrs[i])
	}

	for i := 0; i < 10; i++ {
		conn := Select(leastActiveLoadBalance, sessions, "test_xid")
		assert.NotNil(t, conn)
		assert.Equal(t, addrs[0], conn.RemoteAddr())
	}
}

func TestSelect_FallbackToRandom(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions, addrs := setupSessions(ctrl, 1, false)
	conn := Select("UnknownType", sessions, "test_xid")
	assert.NotNil(t, conn)
	assert.Equal(t, addrs[0], conn.RemoteAddr())
}

func TestSelect_EmptyClosedSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions, _ := setupSessions(ctrl, 3, true)
	conn := Select(randomLoadBalance, sessions, "test_xid")
	assert.Nil(t, conn)

	emptySessions := &sync.Map{}
	assert.Nil(t, Select(randomLoadBalance, emptySessions, "test_xid"))
}

func TestSelect_ConcurrentDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions, addrs := setupSessions(ctrl, 10, false)

	stopCh := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopCh:
				return
			default:
				rpc.BeginCount(addrs[0])
				rpc.EndCount(addrs[0])
			}
		}
	}()

	var wg sync.WaitGroup
	strategies := []string{
		randomLoadBalance,
		xidLoadBalance,
		roundRobinLoadBalance,
		consistentHashLoadBalance,
		leastActiveLoadBalance,
	}

	results := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			strategy := strategies[idx%len(strategies)]
			conn := Select(strategy, sessions, "test_xid")
			results <- (conn != nil)
		}(i)
	}

	wg.Wait()
	close(stopCh)
	close(results)

	for res := range results {
		assert.True(t, res)
	}
}
