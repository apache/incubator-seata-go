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

	"seata.apache.org/seata-go/pkg/remoting/mock"
)

func TestXidLoadBalance(t *testing.T) {
	sessions := &sync.Map{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockTestSession(ctrl)
	m.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return "127.0.0.1:8000"
	})
	m.EXPECT().IsClosed().AnyTimes().DoAndReturn(func() bool {
		return false
	})
	sessions.Store(m, 8000)

	m = mock.NewMockTestSession(ctrl)
	m.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return "127.0.0.1:8001"
	})
	m.EXPECT().IsClosed().AnyTimes().DoAndReturn(func() bool {
		return true
	})
	sessions.Store(m, 8001)

	m = mock.NewMockTestSession(ctrl)
	m.EXPECT().RemoteAddr().AnyTimes().DoAndReturn(func() string {
		return "127.0.0.1:8002"
	})
	m.EXPECT().IsClosed().AnyTimes().DoAndReturn(func() bool {
		return false
	})
	sessions.Store(m, 8002)

	// test
	testCases := []struct {
		name        string
		sessions    *sync.Map
		xid         string
		returnAddrs []string
	}{
		{
			name:        "normal",
			sessions:    sessions,
			xid:         "127.0.0.1:8000:111",
			returnAddrs: []string{"127.0.0.1:8000"},
		},
		{
			name:        "session is closed",
			sessions:    sessions,
			xid:         "127.0.0.1:8001:111",
			returnAddrs: []string{"127.0.0.1:8000", "127.0.0.1:8002"},
		},
		{
			name:        "xid is not exist",
			sessions:    sessions,
			xid:         "127.0.0.1:9000:111",
			returnAddrs: []string{"127.0.0.1:8000", "127.0.0.1:8002"},
		},
	}
	for _, test := range testCases {
		session := XidLoadBalance(test.sessions, test.xid)
		assert.Contains(t, test.returnAddrs, session.RemoteAddr())
	}

}
