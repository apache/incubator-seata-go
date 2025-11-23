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
	"math"
	"sync"
	"testing"

	getty "github.com/apache/dubbo-getty"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/remoting/mock"
)

func TestRoundRobinLoadBalance_Normal(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	for i := 0; i < 10; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(i == 2).AnyTimes()
		session.EXPECT().RemoteAddr().Return(fmt.Sprintf("%d", i)).AnyTimes()
		sessions.Store(session, fmt.Sprintf("session-%d", i+1))
	}

	for i := 0; i < 10; i++ {
		if i == 2 {
			continue
		}
		result := RoundRobinLoadBalance(sessions, "some_xid")
		assert.Equal(t, fmt.Sprintf("%d", i), result.RemoteAddr())
		assert.NotNil(t, result)
		assert.False(t, result.IsClosed())
	}
}

func TestRoundRobinLoadBalance_OverSequence(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}
	sequence = math.MaxInt32

	for i := 0; i < 10; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(false).AnyTimes()
		session.EXPECT().RemoteAddr().Return(fmt.Sprintf("%d", i)).AnyTimes()
		sessions.Store(session, fmt.Sprintf("session-%d", i+1))
	}

	for i := 0; i < 10; i++ {
		// over sequence here
		if i == 0 {
			result := RoundRobinLoadBalance(sessions, "some_xid")
			assert.Equal(t, "7", result.RemoteAddr())
			assert.NotNil(t, result)
			assert.False(t, result.IsClosed())
			continue
		}
		result := RoundRobinLoadBalance(sessions, "some_xid")
		assert.Equal(t, fmt.Sprintf("%d", i-1), result.RemoteAddr())
		assert.NotNil(t, result)
		assert.False(t, result.IsClosed())
	}
}

func TestRoundRobinLoadBalance_All_Closed(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}
	for i := 0; i < 10; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(true).AnyTimes()
		sessions.Store(session, fmt.Sprintf("session-%d", i+1))
	}
	if result := RoundRobinLoadBalance(sessions, "some_xid"); result != nil {
		t.Errorf("Expected nil, actual got %+v", result)
	}
}

func TestRoundRobinLoadBalance_Empty(t *testing.T) {
	sessions := &sync.Map{}
	if result := RoundRobinLoadBalance(sessions, "some_xid"); result != nil {
		t.Errorf("Expected nil, actual got %+v", result)
	}
}

func TestRoundRobinLoadBalance_ConcurrentAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	for i := 0; i < 5; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(false).AnyTimes()
		session.EXPECT().RemoteAddr().Return(fmt.Sprintf("addr-%d", i)).AnyTimes()
		sessions.Store(session, fmt.Sprintf("session-%d", i))
	}

	var wg sync.WaitGroup
	results := make([]getty.Session, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = RoundRobinLoadBalance(sessions, "xid")
		}(i)
	}
	wg.Wait()

	for i, result := range results {
		assert.NotNil(t, result, "Result at index %d should not be nil", i)
		assert.False(t, result.IsClosed())
	}
}

func TestRoundRobinLoadBalance_SelectedSessionClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	callCount := 0
	session1 := mock.NewMockTestSession(ctrl)
	session1.EXPECT().IsClosed().DoAndReturn(func() bool {
		callCount++
		return callCount > 2
	}).AnyTimes()
	session1.EXPECT().RemoteAddr().Return("addr-0").AnyTimes()

	session2 := mock.NewMockTestSession(ctrl)
	session2.EXPECT().IsClosed().Return(false).AnyTimes()
	session2.EXPECT().RemoteAddr().Return("addr-1").AnyTimes()

	sessions.Store(session1, "session-1")
	sessions.Store(session2, "session-2")

	result1 := RoundRobinLoadBalance(sessions, "xid")
	assert.NotNil(t, result1)

	result2 := RoundRobinLoadBalance(sessions, "xid")
	assert.NotNil(t, result2)

	result3 := RoundRobinLoadBalance(sessions, "xid")
	assert.NotNil(t, result3)
}

func TestGetSelector_CacheReuse(t *testing.T) {
	sessions := &sync.Map{}

	selector1 := getSelector(sessions)
	assert.NotNil(t, selector1)

	selector2 := getSelector(sessions)
	assert.Equal(t, selector1, selector2, "Should return cached selector")
}

func TestGetSelector_ConcurrentCreation(t *testing.T) {
	sessions := &sync.Map{}

	selectorCache = sync.Map{}

	var wg sync.WaitGroup
	selectors := make([]*rrSelector, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			selectors[idx] = getSelector(sessions)
		}(i)
	}
	wg.Wait()

	for i := 1; i < 10; i++ {
		assert.Equal(t, selectors[0], selectors[i],
			"All selectors should be the same instance")
	}
}

func TestRRSelector_GetValidSnapshot_Nil(t *testing.T) {
	selector := &rrSelector{sessions: &sync.Map{}}
	selector.snapshot.Store((*rrSnapshot)(nil))

	snap := selector.getValidSnapshot()
	assert.Nil(t, snap, "Should return nil for nil snapshot")
}

func TestRRSelector_GetValidSnapshot_EmptySessions(t *testing.T) {
	selector := &rrSelector{sessions: &sync.Map{}}
	selector.snapshot.Store(&rrSnapshot{sessions: []getty.Session{}})

	snap := selector.getValidSnapshot()
	assert.Nil(t, snap, "Should return nil for empty sessions")
}

func TestRRSelector_RebuildWithSeq_DeleteClosedSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	closedSession := mock.NewMockTestSession(ctrl)
	closedSession.EXPECT().IsClosed().Return(true).AnyTimes()

	openSession := mock.NewMockTestSession(ctrl)
	openSession.EXPECT().IsClosed().Return(false).AnyTimes()
	openSession.EXPECT().RemoteAddr().Return("addr-1").AnyTimes()

	sessions.Store(closedSession, "closed")
	sessions.Store(openSession, "open")

	result := RoundRobinLoadBalance(sessions, "xid")

	count := 0
	sessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, 1, count, "Should only have 1 session left")
	assert.NotNil(t, result)
	assert.Equal(t, "addr-1", result.RemoteAddr())
}
