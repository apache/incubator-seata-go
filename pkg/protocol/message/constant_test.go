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

package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstantMessageType(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalBegin, MessageType(1))
	assert.Equal(t, MessageTypeGlobalBeginResult, MessageType(2))
	assert.Equal(t, MessageTypeBranchCommit, MessageType(3))
	assert.Equal(t, MessageTypeBranchCommitResult, MessageType(4))
	assert.Equal(t, MessageTypeBranchRollback, MessageType(5))
	assert.Equal(t, MessageTypeBranchRollbackResult, MessageType(6))
	assert.Equal(t, MessageTypeGlobalCommit, MessageType(7))
	assert.Equal(t, MessageTypeGlobalCommitResult, MessageType(8))
	assert.Equal(t, MessageTypeGlobalRollback, MessageType(9))
	assert.Equal(t, MessageTypeGlobalRollbackResult, MessageType(10))
	assert.Equal(t, MessageTypeBranchRegister, MessageType(11))
	assert.Equal(t, MessageTypeBranchRegisterResult, MessageType(12))
	assert.Equal(t, MessageTypeBranchStatusReport, MessageType(13))
	assert.Equal(t, MessageTypeBranchStatusReportResult, MessageType(14))
	assert.Equal(t, MessageTypeGlobalStatus, MessageType(15))
	assert.Equal(t, MessageTypeGlobalStatusResult, MessageType(16))
	assert.Equal(t, MessageTypeGlobalReport, MessageType(17))
	assert.Equal(t, MessageTypeGlobalReportResult, MessageType(18))
	assert.Equal(t, MessageTypeGlobalLockQuery, MessageType(21))
	assert.Equal(t, MessageTypeGlobalLockQueryResult, MessageType(22))
	assert.Equal(t, MessageTypeSeataMerge, MessageType(59))
	assert.Equal(t, MessageTypeSeataMergeResult, MessageType(60))
	assert.Equal(t, MessageTypeRegClt, MessageType(101))
	assert.Equal(t, MessageTypeRegCltResult, MessageType(102))
	assert.Equal(t, MessageTypeRegRm, MessageType(103))
	assert.Equal(t, MessageTypeRegRmResult, MessageType(104))
	assert.Equal(t, MessageTypeRmDeleteUndolog, MessageType(111))
	assert.Equal(t, MessageTypeHeartbeatMsg, MessageType(120))
	assert.Equal(t, MessageTypeBatchResultMsg, MessageType(121))
}
