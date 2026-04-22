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

	getty "github.com/apache/dubbo-getty"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/remoting/mock"
)

func TestConsistentHashLoadBalance(t *testing.T) {
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

	c := NewConsistent(0)
	result := ConsistentHashLoadBalance(c, sessions, "test_xid")
	assert.NotNil(t, result)
	assert.False(t, result.IsClosed())

	sessions.Range(func(key, value interface{}) bool {
		t.Logf("key: %v, value: %v", key, value)
		return true
	})
}

func TestConsistentHashLoadBalance_NilBalancerFallsBackToRandom(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	session := mock.NewMockTestSession(ctrl)
	session.EXPECT().IsClosed().Return(false).AnyTimes()
	sessions.Store(session, "only")

	// A nil *Consistent must not panic and must still yield a live session.
	result := ConsistentHashLoadBalance(nil, sessions, "test_xid")
	assert.Equal(t, getty.Session(session), result)
}

func TestConsistentPick_RefreshesClosedSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	closedSession := mock.NewMockTestSession(ctrl)
	closedSession.EXPECT().IsClosed().AnyTimes().Return(true)

	openSession := mock.NewMockTestSession(ctrl)
	openSession.EXPECT().IsClosed().AnyTimes().Return(false)
	openSession.EXPECT().RemoteAddr().AnyTimes().Return("127.0.0.1:8001")

	sessions.Store(closedSession, "closed")
	sessions.Store(openSession, "open")

	c := NewConsistent(0)
	c.Lock()
	c.hashCircle = map[int64]getty.Session{1: closedSession}
	c.sortedHashNodes = []int64{1}
	c.Unlock()

	result := c.pick(sessions, "test_xid")
	assert.NotNil(t, result)
	assert.Equal(t, getty.Session(openSession), result)

	_, stillExists := sessions.Load(closedSession)
	assert.False(t, stillExists)
}

func TestConsistentPick_ConcurrentPickAndRefresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	stableSession := mock.NewMockTestSession(ctrl)
	stableSession.EXPECT().IsClosed().AnyTimes().Return(false)
	stableSession.EXPECT().RemoteAddr().AnyTimes().Return("127.0.0.1:9000")
	sessions.Store(stableSession, "stable")

	type flappingSession struct {
		session getty.Session
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

	c := NewConsistent(0)
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

// TestConsistentRefresh_SerializedWriters asserts that two goroutines
// rebuilding the ring at the same time still leave it in a consistent
// "every hash-node maps to something in the circle" state.
func TestConsistentRefresh_SerializedWriters(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	for i := 0; i < 4; i++ {
		addr := fmt.Sprintf("127.0.0.1:70%02d", i)
		s := mock.NewMockTestSession(ctrl)
		s.EXPECT().IsClosed().AnyTimes().Return(false)
		s.EXPECT().RemoteAddr().AnyTimes().Return(addr)
		sessions.Store(s, addr)
	}

	c := NewConsistent(0)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.refreshHashCircle(sessions)
		}()
	}
	wg.Wait()

	c.RLock()
	defer c.RUnlock()
	// sortedHashNodes may legitimately contain duplicate positions when two
	// virtual nodes hash to the same slot; hashCircle is a map so it
	// collapses them. Assert every slot in the slice points at a live
	// session in the ring instead of comparing sizes.
	for _, pos := range c.sortedHashNodes {
		session, ok := c.hashCircle[pos]
		assert.True(t, ok)
		assert.NotNil(t, session)
	}
}
