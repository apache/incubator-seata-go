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
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/remoting/mock"
)

func TestConsistentHashLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	now := time.Now()

	session := mock.NewMockTestSession(ctrl)
	session.EXPECT().GetActive().Return(now.Add(-time.Second * 7)).AnyTimes()
	session.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return "127.0.0.1:8000"
	})
	session.EXPECT().IsClosed().Return(false).AnyTimes()
	sessions.Store(session, fmt.Sprintf("session-%d", 1))

	session = mock.NewMockTestSession(ctrl)
	session.EXPECT().GetActive().Return(now.Add(-time.Second * 5)).AnyTimes()
	session.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return "127.0.0.1:8001"
	})
	session.EXPECT().IsClosed().Return(false).AnyTimes()
	sessions.Store(session, fmt.Sprintf("session-%d", 2))

	session = mock.NewMockTestSession(ctrl)
	session.EXPECT().GetActive().Return(now.Add(-time.Second * 10)).AnyTimes()
	session.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return "127.0.0.1:8002"
	})
	session.EXPECT().IsClosed().Return(true).AnyTimes()
	sessions.Store(session, fmt.Sprintf("session-%d", 3))

	result := LeastActiveLoadBalance(sessions, "test_xid")
	assert.True(t, result.RemoteAddr() == "127.0.0.1:8000")
	assert.False(t, result.IsClosed())
}
