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

	"seata.apache.org/seata-go/pkg/remoting/mock"
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

	result := ConsistentHashLoadBalance(sessions, "test_xid")
	assert.NotNil(t, result)
	assert.False(t, result.IsClosed())

	sessions.Range(func(key, value interface{}) bool {
		t.Logf("key: %v, value: %v", key, value)
		return true
	})
}
