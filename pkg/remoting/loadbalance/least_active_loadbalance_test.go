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
	"strconv"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/remoting/mock"
	"seata.apache.org/seata-go/pkg/remoting/rpc"
)

func TestLeastActiveLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	for i := 1; i <= 3; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(false).AnyTimes()
		addr := "127.0.0." + strconv.Itoa(i) + ":8000"
		session.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
			return addr
		})
		sessions.Store(session, fmt.Sprintf("session-%d", i))
		rpc.BeginCount(addr)
	}

	session := mock.NewMockTestSession(ctrl)
	session.EXPECT().IsClosed().Return(true).AnyTimes()
	addr := "127.0.0.5:8000"
	session.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return addr
	})
	sessions.Store(session, "session-5")
	rpc.BeginCount(addr)

	countTwo := "127.0.0.1:8000"
	rpc.BeginCount(countTwo)

	result := LeastActiveLoadBalance(sessions, "test_xid")
	assert.False(t, result.RemoteAddr() == countTwo)
	assert.False(t, result.RemoteAddr() == addr)
	assert.False(t, result.IsClosed())
}
