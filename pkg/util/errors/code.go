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

type ErrorCode int32

const (
	_ ErrorCode = iota
	// ErrorCodeUnknown Unknown transaction errors code.
	ErrorCodeUnknown

	// ErrorCodeBeginFailed BeginFailed
	ErrorCodeBeginFailed

	// ErrorCodeLockKeyConflict Lock key conflict transaction errors code.
	ErrorCodeLockKeyConflict

	// ErrorCodeIO transaction errors code.
	ErrorCodeIO

	// ErrorCodeBranchRollbackFailedRetriable Branch rollback failed retriable transaction errors code.
	ErrorCodeBranchRollbackFailedRetriable

	// ErrorCodeBranchRollbackFailedUnretriable Branch rollback failed unretriable transaction errors code.
	ErrorCodeBranchRollbackFailedUnretriable

	// ErrorCodeBranchRegisterFailed Branch register failed transaction errors code.
	ErrorCodeBranchRegisterFailed

	// ErrorCodeBranchReportFailed Branch report failed transaction errors code.
	ErrorCodeBranchReportFailed

	// ErrorCodeLockableCheckFailed Lockable check failed transaction errors code.
	ErrorCodeLockableCheckFailed

	// ErrorCodeBranchTransactionNotExist Branch transaction not exist transaction errors code.
	ErrorCodeBranchTransactionNotExist

	// ErrorCodeGlobalTransactionNotExist Global transaction not exist transaction errors code.
	ErrorCodeGlobalTransactionNotExist

	// ErrorCodeGlobalTransactionNotActive Global transaction not active transaction errors code.
	ErrorCodeGlobalTransactionNotActive

	// ErrorCodeGlobalTransactionStatusInvalid Global transaction status invalid transaction errors code.
	ErrorCodeGlobalTransactionStatusInvalid

	// ErrorCodeFailedToSendBranchCommitRequest Failed to send branch commit request transaction errors code.
	ErrorCodeFailedToSendBranchCommitRequest

	// ErrorCodeFailedToSendBranchRollbackRequest Failed to send branch rollback request transaction errors code.
	ErrorCodeFailedToSendBranchRollbackRequest

	// ErrorCodeFailedToAddBranch Failed to add branch transaction errors code.
	ErrorCodeFailedToAddBranch

	// ErrorCodeFailedLockGlobalTranscation Failed to lock global transaction errors code.
	ErrorCodeFailedLockGlobalTranscation

	// ErrorCodeFailedWriteSession FailedWriteSession
	ErrorCodeFailedWriteSession

	// ErrorCodeFailedStore Failed to holder errors code
	ErrorCodeFailedStore

	// ErrorCodeLockKeyConflictFailFast Lock key conflict fail fast transaction exception code.
	ErrorCodeLockKeyConflictFailFast

	// ErrorCodeTccFenceDbDuplicateKey Insert tcc fence record duplicate key errors
	ErrorCodeTccFenceDbDuplicateKey

	// ErrorCodeRollbackFence rollback tcc fence error
	ErrorCodeRollbackFence

	// ErrorCodeCommitFence commit tcc fence  error
	ErrorCodeCommitFence

	// ErrorCodeTccFenceDb query tcc fence prepare sql failed
	ErrorCodeTccFenceDb

	// ErrorCodePrepareFence prepare tcc fence error
	ErrorCodePrepareFence

	// ErrorCodeFenceBusiness callback business method maybe return this error type
	ErrorCodeFenceBusiness

	// ErrorCodeFencePhase have fence phase but is not illegal value
	ErrorCodeFencePhase
)
