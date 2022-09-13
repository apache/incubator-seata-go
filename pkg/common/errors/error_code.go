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

type TransactionErrorCode int32

const (
	// TransactionErrorCodeUnknown Unknown transaction errors code.
	TransactionErrorCodeUnknown = TransactionErrorCode(0)

	// TransactionErrorCodeBeginFailed BeginFailed
	TransactionErrorCodeBeginFailed = TransactionErrorCode(1)

	// TransactionErrorCodeLockKeyConflict Lock key conflict transaction errors code.
	TransactionErrorCodeLockKeyConflict = TransactionErrorCode(2)

	// Io transaction errors code.
	IO = TransactionErrorCode(3)

	// TransactionErrorCodeBranchRollbackFailedRetriable Branch rollback failed retriable transaction errors code.
	TransactionErrorCodeBranchRollbackFailedRetriable = TransactionErrorCode(4)

	// TransactionErrorCodeBranchRollbackFailedUnretriable Branch rollback failed unretriable transaction errors code.
	TransactionErrorCodeBranchRollbackFailedUnretriable = TransactionErrorCode(5)

	// TransactionErrorCodeBranchRegisterFailed Branch register failed transaction errors code.
	TransactionErrorCodeBranchRegisterFailed = TransactionErrorCode(6)

	// TransactionErrorCodeBranchReportFailed Branch report failed transaction errors code.
	TransactionErrorCodeBranchReportFailed = TransactionErrorCode(7)

	// TransactionErrorCodeLockableCheckFailed Lockable check failed transaction errors code.
	TransactionErrorCodeLockableCheckFailed = TransactionErrorCode(8)

	// TransactionErrorCodeBranchTransactionNotExist Branch transaction not exist transaction errors code.
	TransactionErrorCodeBranchTransactionNotExist = TransactionErrorCode(9)

	// TransactionErrorCodeGlobalTransactionNotExist Global transaction not exist transaction errors code.
	TransactionErrorCodeGlobalTransactionNotExist = TransactionErrorCode(10)

	// TransactionErrorCodeGlobalTransactionNotActive Global transaction not active transaction errors code.
	TransactionErrorCodeGlobalTransactionNotActive = TransactionErrorCode(11)

	// TransactionErrorCodeGlobalTransactionStatusInvalid Global transaction status invalid transaction errors code.
	TransactionErrorCodeGlobalTransactionStatusInvalid = TransactionErrorCode(12)

	// TransactionErrorCodeFailedToSendBranchCommitRequest Failed to send branch commit request transaction errors code.
	TransactionErrorCodeFailedToSendBranchCommitRequest = TransactionErrorCode(13)

	// TransactionErrorCodeFailedToSendBranchRollbackRequest Failed to send branch rollback request transaction errors code.
	TransactionErrorCodeFailedToSendBranchRollbackRequest = TransactionErrorCode(14)

	// TransactionErrorCodeFailedToAddBranch Failed to add branch transaction errors code.
	TransactionErrorCodeFailedToAddBranch = TransactionErrorCode(15)

	// TransactionErrorCodeFailedLockGlobalTranscation Failed to lock global transaction errors code.
	TransactionErrorCodeFailedLockGlobalTranscation = TransactionErrorCode(16)

	// TransactionErrorCodeFailedWriteSession FailedWriteSession
	TransactionErrorCodeFailedWriteSession = TransactionErrorCode(17)

	// FailedStore Failed to holder errors code
	FailedStore = TransactionErrorCode(18)

	// LockKeyConflictFailFast Lock key conflict fail fast transaction exception code.
	LockKeyConflictFailFast = TransactionErrorCode(19)

	// TccFenceDbDuplicateKeyError Insert tcc fence record duplicate key errors
	TccFenceDbDuplicateKeyError = TransactionErrorCode(20)

	// RollbackFenceError rollback tcc fence error
	RollbackFenceError = TransactionErrorCode(21)

	// CommitFenceError commit tcc fence  error
	CommitFenceError = TransactionErrorCode(22)

	// TccFenceDbError query tcc fence prepare sql failed
	TccFenceDbError = TransactionErrorCode(23)

	// PrepareFenceError prepare tcc fence error
	PrepareFenceError = TransactionErrorCode(24)

	// FenceBusinessError callback business method maybe return this error type
	FenceBusinessError = TransactionErrorCode(26)

	// FencePhaseError have fence phase but is not illegal value
	FencePhaseError = TransactionErrorCode(27)
)
