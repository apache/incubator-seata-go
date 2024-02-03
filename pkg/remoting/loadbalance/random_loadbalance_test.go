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

	"github.com/seata/seata-go/pkg/remoting/mock"
)

func TestRandomLoadBalance_Normal(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}

	for i := 0; i < 10; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(i == 2).AnyTimes()
		sessions.Store(session, fmt.Sprintf("session-%d", (i+1)))
	}
	result := RandomLoadBalance(sessions, "some_xid")

	assert.NotNil(t, result)
	//assert random load balance return session not closed
	assert.False(t, result.IsClosed())
}

func TestRandomLoadBalance_All_Closed(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}
	//mock  closed  sessions
	for i := 0; i < 10; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(true).AnyTimes()
		sessions.Store(session, fmt.Sprintf("session-%d", (i+1)))
	}
	if result := RandomLoadBalance(sessions, "some_xid"); result != nil {
		t.Errorf("Expected nil, actual got %+v", result)
	}
}

func TestRandomLoadBalance_Empty(t *testing.T) {
	sessions := &sync.Map{}
	if result := RandomLoadBalance(sessions, "some_xid"); result != nil {
		t.Errorf("Expected nil, actual got %+v", result)
	}
}

func TestRandomLoadBalance_All_Opening(t *testing.T) {

	ctrl := gomock.NewController(t)
	sessions := &sync.Map{}
	for i := 0; i < 10; i++ {
		session := mock.NewMockTestSession(ctrl)
		session.EXPECT().IsClosed().Return(false).AnyTimes()
		sessions.Store(session, fmt.Sprintf("session-%d", (i+1)))
	}
	result := RandomLoadBalance(sessions, "some_xid")
	//assert return session is not closed
	assert.False(t, result.IsClosed())
}
