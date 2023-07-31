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
	"github.com/seata/seata-go/pkg/remoting/mock"
	"github.com/stretchr/testify/assert"
)

func TestRandomLoadBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	// defer ctrl.Finish()
	sessions := &sync.Map{}

	//mock two opening session
	openSession := mock.NewMockTestSession(ctrl)
	openSession.EXPECT().IsClosed().Return(false).AnyTimes()

	//mock two opening session
	openSession2 := mock.NewMockTestSession(ctrl)
	openSession2.EXPECT().IsClosed().Return(false).AnyTimes()

	//mock one closed session
	closeSession := mock.NewMockTestSession(ctrl)
	closeSession.EXPECT().IsClosed().Return(true).AnyTimes()

	sessions.Store(openSession, "session1")
	sessions.Store(closeSession, "session2")
	sessions.Store(openSession2, "session3")

	result := RandomLoadBalance(sessions, "some_xid")

	assert.NotNil(t, result)
	//assert random load balance return session not closed
	assert.False(t, result.IsClosed())
}
