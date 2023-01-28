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
	assert.Equal(t, int(MessageTypeGlobalBegin), 1)
	assert.Equal(t, int(MessageTypeGlobalBeginResult), 2)
	assert.Equal(t, int(MessageTypeBranchCommit), 3)
	assert.Equal(t, int(MessageTypeBranchCommitResult), 4)
	assert.Equal(t, int(MessageTypeBranchRollback), 5)
	assert.Equal(t, int(MessageTypeBranchRollbackResult), 6)
	assert.Equal(t, int(MessageTypeGlobalCommit), 7)
	assert.Equal(t, int(MessageTypeGlobalCommitResult), 8)
	assert.Equal(t, int(MessageTypeGlobalRollback), 9)
	assert.Equal(t, int(MessageTypeGlobalRollbackResult), 10)
	assert.Equal(t, int(MessageTypeBranchRegister), 11)
	assert.Equal(t, int(MessageTypeBranchRegisterResult), 12)
	assert.Equal(t, int(MessageTypeBranchStatusReport), 13)
	assert.Equal(t, int(MessageTypeBranchStatusReportResult), 14)
	assert.Equal(t, int(MessageTypeGlobalStatus), 15)
	assert.Equal(t, int(MessageTypeGlobalStatusResult), 16)
	assert.Equal(t, int(MessageTypeGlobalReport), 17)
	assert.Equal(t, int(MessageTypeGlobalReportResult), 18)
	assert.Equal(t, int(MessageTypeGlobalLockQuery), 21)
	assert.Equal(t, int(MessageTypeGlobalLockQueryResult), 22)
	assert.Equal(t, int(MessageTypeSeataMerge), 59)
	assert.Equal(t, int(MessageTypeSeataMergeResult), 60)
	assert.Equal(t, int(MessageTypeRegClt), 101)
	assert.Equal(t, int(MessageTypeRegCltResult), 102)
	assert.Equal(t, int(MessageTypeRegRm), 103)
	assert.Equal(t, int(MessageTypeRegRmResult), 104)
	assert.Equal(t, int(MessageTypeRmDeleteUndolog), 111)
	assert.Equal(t, int(MessageTypeHeartbeatMsg), 120)
	assert.Equal(t, int(MessageTypeBatchResultMsg), 121)
}
