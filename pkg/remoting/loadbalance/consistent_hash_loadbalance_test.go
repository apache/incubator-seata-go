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
	"sync/atomic"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/connection"
	"seata.apache.org/seata-go/v2/pkg/remoting/mock"
)

func resetConsistentHashForTest() {
	consistentInstance = nil
	virtualNodeNumber = 10
	once = sync.Once{}
}

func TestConsistentHashLoadBalance(t *testing.T) {
	resetConsistentHashForTest()
	defer resetConsistentHashForTest()

	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	for i := 0; i < 3; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(false).AnyTimes()
		session.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
			return "127.0.0.1:8000"
		})
		sessions.Store(session, fmt.Sprintf("session-%d", i))
	}

	result := ConsistentHashLoadBalance(sessions, "test_xid")
	assert.NotNil(t, result)
	assert.False(t, result.IsClosed())

	sessions.Range(func(key, value interface{}) bool {
		t.Logf("key: %v, value: %v", key, value)
		return true
	})
}

func TestConsistentPick_RefreshesClosedSession(t *testing.T) {
	resetConsistentHashForTest()
	defer resetConsistentHashForTest()

	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	closedSession := mock.NewMockTestSession(ctrl)
	closedSession.EXPECT().IsClosed().AnyTimes().Return(true)

	openSession := mock.NewMockTestSession(ctrl)
	openSession.EXPECT().IsClosed().AnyTimes().Return(false)
	openSession.EXPECT().RemoteAddr().AnyTimes().Return("127.0.0.1:8001")

	sessions.Store(closedSession, "closed")
	sessions.Store(openSession, "open")

	c := &Consistent{
		virtualNodeCount: virtualNodeNumber,
		hashCircle: map[int64]connection.Connection{
			1: closedSession,
		},
		sortedHashNodes: []int64{1},
	}

	result := c.pick(sessions, "test_xid")
	assert.NotNil(t, result)
	assert.Equal(t, openSession, result)

	_, stillExists := sessions.Load(closedSession)
	assert.False(t, stillExists)
}

func TestConsistentPick_ConcurrentPickAndRefresh(t *testing.T) {
	resetConsistentHashForTest()
	defer resetConsistentHashForTest()

	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	stableSession := mock.NewMockTestSession(ctrl)
	stableSession.EXPECT().IsClosed().AnyTimes().Return(false)
	stableSession.EXPECT().RemoteAddr().AnyTimes().Return("127.0.0.1:9000")
	sessions.Store(stableSession, "stable")

	type flappingSession struct {
		session connection.Connection
		addr    string
		closed  atomic.Bool
	}

	flapping := make([]*flappingSession, 0, 2)
	for i := 0; i < 2; i++ {
		addr := fmt.Sprintf("127.0.0.1:900%d", i+1)
		state := &flappingSession{addr: addr}

		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().AnyTimes().DoAndReturn(func() bool {
			return state.closed.Load()
		})
		session.EXPECT().RemoteAddr().AnyTimes().Return(addr)

		state.session = session
		flapping = append(flapping, state)
		sessions.Store(session, addr)
	}

	c := &Consistent{
		virtualNodeCount: virtualNodeNumber,
		hashCircle:       make(map[int64]connection.Connection),
	}
	c.refreshHashCircle(sessions)

	const (
		pickers    = 8
		iterations = 200
	)

	start := make(chan struct{})
	var wg sync.WaitGroup

	for pickerID := 0; pickerID < pickers; pickerID++ {
		pickerID := pickerID
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < iterations; i++ {
				session := c.pick(sessions, fmt.Sprintf("xid-%d-%d", pickerID, i))
				if session == nil {
					continue
				}
				_ = session.IsClosed()
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			state := flapping[i%len(flapping)]
			if i%2 == 0 {
				state.closed.Store(true)
			} else {
				state.closed.Store(false)
				sessions.Store(state.session, state.addr)
			}

			c.refreshHashCircle(sessions)
		}
	}()

	close(start)
	wg.Wait()

	result := c.pick(sessions, "final-xid")
	assert.NotNil(t, result)
	assert.False(t, result.IsClosed())
}
