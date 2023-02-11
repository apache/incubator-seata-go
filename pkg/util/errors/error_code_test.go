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
	assert.Equal(t, int(TransactionErrorCodeUnknown), 0)
	assert.Equal(t, int(TransactionErrorCodeBeginFailed), 1)
	assert.Equal(t, int(TransactionErrorCodeCommitFailed), 2)
	assert.Equal(t, int(TransactionErrorCodeRollbackFiled), 3)
	assert.Equal(t, int(TransactionErrorCodeLockKeyConflict), 4)
	assert.Equal(t, int(TransactionErrorCodeIO), 5)
	assert.Equal(t, int(TransactionErrorCodeBranchRollbackFailedRetriable), 6)
	assert.Equal(t, int(TransactionErrorCodeBranchRollbackFailedUnretriable), 7)
	assert.Equal(t, int(TransactionErrorCodeBranchRegisterFailed), 8)
	assert.Equal(t, int(TransactionErrorCodeBranchReportFailed), 9)
	assert.Equal(t, int(TransactionErrorCodeLockableCheckFailed), 10)
	assert.Equal(t, int(TransactionErrorCodeBranchTransactionNotExist), 11)
	assert.Equal(t, int(TransactionErrorCodeGlobalTransactionNotExist), 12)
	assert.Equal(t, int(TransactionErrorCodeGlobalTransactionNotActive), 13)
	assert.Equal(t, int(TransactionErrorCodeGlobalTransactionStatusInvalid), 14)
	assert.Equal(t, int(TransactionErrorCodeFailedToSendBranchCommitRequest), 15)
	assert.Equal(t, int(TransactionErrorCodeFailedToSendBranchRollbackRequest), 16)
	assert.Equal(t, int(TransactionErrorCodeFailedToAddBranch), 17)
	assert.Equal(t, int(TransactionErrorCodeFailedLockGlobalTranscation), 18)
	assert.Equal(t, int(TransactionErrorCodeFailedWriteSession), 19)
	assert.Equal(t, int(FailedStore), 20)
	assert.Equal(t, int(LockKeyConflictFailFast), 21)
	assert.Equal(t, int(TccFenceDbDuplicateKeyError), 22)
	assert.Equal(t, int(RollbackFenceError), 23)
	assert.Equal(t, int(CommitFenceError), 24)
	assert.Equal(t, int(TccFenceDbError), 25)
	assert.Equal(t, int(PrepareFenceError), 26)
	assert.Equal(t, int(FenceBusinessError), 27)
	assert.Equal(t, int(FencePhaseError), 28)
}
