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

package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionErrorCode(t *testing.T) {
	assert.Equal(t, TransactionErrorCodeUnknown, TransactionErrorCode(0))
	assert.Equal(t, TransactionErrorCodeBeginFailed, TransactionErrorCode(1))
	assert.Equal(t, TransactionErrorCodeLockKeyConflict, TransactionErrorCode(2))
	assert.Equal(t, IO, TransactionErrorCode(3))
	assert.Equal(t, TransactionErrorCodeBranchRollbackFailedRetriable, TransactionErrorCode(4))
	assert.Equal(t, TransactionErrorCodeBranchRollbackFailedUnretriable, TransactionErrorCode(5))
	assert.Equal(t, TransactionErrorCodeBranchRegisterFailed, TransactionErrorCode(6))
	assert.Equal(t, TransactionErrorCodeBranchReportFailed, TransactionErrorCode(7))
	assert.Equal(t, TransactionErrorCodeLockableCheckFailed, TransactionErrorCode(8))
	assert.Equal(t, TransactionErrorCodeBranchTransactionNotExist, TransactionErrorCode(9))
	assert.Equal(t, TransactionErrorCodeGlobalTransactionNotExist, TransactionErrorCode(10))
	assert.Equal(t, TransactionErrorCodeGlobalTransactionNotActive, TransactionErrorCode(11))
	assert.Equal(t, TransactionErrorCodeGlobalTransactionStatusInvalid, TransactionErrorCode(12))
	assert.Equal(t, TransactionErrorCodeFailedToSendBranchCommitRequest, TransactionErrorCode(13))
	assert.Equal(t, TransactionErrorCodeFailedToSendBranchRollbackRequest, TransactionErrorCode(14))
	assert.Equal(t, TransactionErrorCodeFailedToAddBranch, TransactionErrorCode(15))
	assert.Equal(t, TransactionErrorCodeFailedLockGlobalTranscation, TransactionErrorCode(16))
	assert.Equal(t, TransactionErrorCodeFailedWriteSession, TransactionErrorCode(17))
	assert.Equal(t, FailedStore, TransactionErrorCode(18))
	assert.Equal(t, LockKeyConflictFailFast, TransactionErrorCode(19))
	assert.Equal(t, TccFenceDbDuplicateKeyError, TransactionErrorCode(20))
	assert.Equal(t, RollbackFenceError, TransactionErrorCode(21))
	assert.Equal(t, CommitFenceError, TransactionErrorCode(22))
	assert.Equal(t, TccFenceDbError, TransactionErrorCode(23))
	assert.Equal(t, PrepareFenceError, TransactionErrorCode(24))
	assert.Equal(t, FenceBusinessError, TransactionErrorCode(26))
	assert.Equal(t, FencePhaseError, TransactionErrorCode(27))
}
