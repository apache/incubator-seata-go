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
	"seata.apache.org/seata-go/v2/pkg/remoting/rpc"
)

func TestSelect_ConsistentHashLoadBalance(t *testing.T) {
	resetConsistentHashState()
	t.Cleanup(resetConsistentHashState)

	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	session := mock.NewMockTestSession(ctrl)
	session.EXPECT().IsClosed().Return(false).AnyTimes()
	session.EXPECT().RemoteAddr().Return("127.0.0.1:8000").MinTimes(1)
	sessions.Store(session, "session-1")

	result := Select(consistentHashLoadBalance, sessions, "test_xid")
	assert.NotNil(t, result)
}

func TestSelect_LeastActiveLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	busyAddr := "127.0.0.1:8000"
	idleAddr := "127.0.0.2:8000"

	busySession := mock.NewMockTestSession(ctrl)
	busySession.EXPECT().IsClosed().Return(false).AnyTimes()
	busySession.EXPECT().RemoteAddr().Return(busyAddr).MinTimes(1)

	idleSession := mock.NewMockTestSession(ctrl)
	idleSession.EXPECT().IsClosed().Return(false).AnyTimes()
	idleSession.EXPECT().RemoteAddr().Return(idleAddr).MinTimes(1)

	sessions.Store(busySession, "busy")
	sessions.Store(idleSession, "idle")

	rpc.BeginCount(busyAddr)
	t.Cleanup(func() {
		rpc.RemoveStatus(busyAddr)
		rpc.RemoveStatus(idleAddr)
	})

	result := Select(leastActiveLoadBalance, sessions, "test_xid")
	assert.NotNil(t, result)
	assert.Equal(t, idleAddr, result.RemoteAddr())
}

func resetConsistentHashState() {
	consistentInstance = nil
	once = sync.Once{}
}
