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
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/v2/pkg/remoting/mock"
)

func setupSessions(ctrl *gomock.Controller, count int, isClosed bool) *sync.Map {
	sessions := &sync.Map{}
	for i := 0; i < count; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(isClosed).AnyTimes()
		session.EXPECT().RemoteAddr().Return("127.0.0.1:8080").AnyTimes()
		sessions.Store(session, "session")
	}
	return sessions
}

func TestSelect_RandomLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := setupSessions(ctrl, 3, false)
	conn := Select(randomLoadBalance, sessions, "test_xid")
	assert.NotNil(t, conn)
}

func TestSelect_XidLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := setupSessions(ctrl, 3, false)
	conn := Select(xidLoadBalance, sessions, "test_xid")
	assert.NotNil(t, conn)
}

func TestSelect_RoundRobinLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := setupSessions(ctrl, 3, false)
	conn := Select(roundRobinLoadBalance, sessions, "test_xid")
	assert.NotNil(t, conn)
}

func TestSelect_ConsistentHashLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := setupSessions(ctrl, 3, false)
	conn := Select(consistentHashLoadBalance, sessions, "test_xid")
	assert.NotNil(t, conn)
}

func TestSelect_LeastActiveLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := setupSessions(ctrl, 3, false)
	conn := Select(leastActiveLoadBalance, sessions, "test_xid")
	assert.NotNil(t, conn)
}

func TestSelect_FallbackToRandom(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := setupSessions(ctrl, 3, false)
	conn := Select("UnknownType", sessions, "test_xid")
	assert.NotNil(t, conn)
}

func TestSelect_EmptyClosedSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := setupSessions(ctrl, 3, true)
	conn := Select(randomLoadBalance, sessions, "test_xid")
	assert.Nil(t, conn)

	emptySessions := &sync.Map{}
	assert.Nil(t, Select(randomLoadBalance, emptySessions, "test_xid"))
}

func TestSelect_ConcurrentDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := setupSessions(ctrl, 10, false)
	
	var wg sync.WaitGroup
	strategies := []string{
		randomLoadBalance,
		xidLoadBalance,
		roundRobinLoadBalance,
		consistentHashLoadBalance,
		leastActiveLoadBalance,
	}
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			strategy := strategies[idx%len(strategies)]
			conn := Select(strategy, sessions, "test_xid")
			assert.NotNil(t, conn)
		}(i)
	}
	
	wg.Wait()
}
