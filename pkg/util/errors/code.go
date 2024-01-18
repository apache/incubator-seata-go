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
	TransactionErrorCodeUnknown TransactionErrorCode = iota

	// TransactionErrorCodeBeginFailed BeginFailed
	TransactionErrorCodeBeginFailed

	// TransactionErrorCodeLockKeyConflict Lock key conflict transaction errors code.
	TransactionErrorCodeLockKeyConflict

	// TransactionErrorCodeIO transaction errors code.
	TransactionErrorCodeIO

	// TransactionErrorCodeBranchRollbackFailedRetriable Branch rollback failed retriable transaction errors code.
	TransactionErrorCodeBranchRollbackFailedRetriable

	// TransactionErrorCodeBranchRollbackFailedUnretriable Branch rollback failed unretriable transaction errors code.
	TransactionErrorCodeBranchRollbackFailedUnretriable

	// TransactionErrorCodeBranchRegisterFailed Branch register failed transaction errors code.
	TransactionErrorCodeBranchRegisterFailed

	// TransactionErrorCodeBranchReportFailed Branch report failed transaction errors code.
	TransactionErrorCodeBranchReportFailed

	// TransactionErrorCodeLockableCheckFailed Lockable check failed transaction errors code.
	TransactionErrorCodeLockableCheckFailed

	// TransactionErrorCodeBranchTransactionNotExist Branch transaction not exist transaction errors code.
	TransactionErrorCodeBranchTransactionNotExist

	// TransactionErrorCodeGlobalTransactionNotExist Global transaction not exist transaction errors code.
	TransactionErrorCodeGlobalTransactionNotExist

	// TransactionErrorCodeGlobalTransactionNotActive Global transaction not active transaction errors code.
	TransactionErrorCodeGlobalTransactionNotActive

	// TransactionErrorCodeGlobalTransactionStatusInvalid Global transaction status invalid transaction errors code.
	TransactionErrorCodeGlobalTransactionStatusInvalid

	// TransactionErrorCodeFailedToSendBranchCommitRequest Failed to send branch commit request transaction errors code.
	TransactionErrorCodeFailedToSendBranchCommitRequest

	// TransactionErrorCodeFailedToSendBranchRollbackRequest Failed to send branch rollback request transaction errors code.
	TransactionErrorCodeFailedToSendBranchRollbackRequest

	// TransactionErrorCodeFailedToAddBranch Failed to add branch transaction errors code.
	TransactionErrorCodeFailedToAddBranch

	// TransactionErrorCodeFailedLockGlobalTranscation Failed to lock global transaction errors code.
	TransactionErrorCodeFailedLockGlobalTranscation

	// TransactionErrorCodeFailedWriteSession FailedWriteSession
	TransactionErrorCodeFailedWriteSession

	// FailedStore Failed to holder errors code
	FailedStore

	// LockKeyConflictFailFast Lock key conflict fail fast transaction exception code.
	LockKeyConflictFailFast

	// TccFenceDbDuplicateKeyError Insert tcc fence record duplicate key errors
	TccFenceDbDuplicateKeyError

	// RollbackFenceError rollback tcc fence error
	RollbackFenceError

	// CommitFenceError commit tcc fence  error
	CommitFenceError

	// TccFenceDbError query tcc fence prepare sql failed
	TccFenceDbError

	// PrepareFenceError prepare tcc fence error
	PrepareFenceError

	// FenceBusinessError callback business method maybe return this error type
	FenceBusinessError

	// FencePhaseError have fence phase but is not illegal value
	FencePhaseError

	// ObjectNotExists object not exists
	ObjectNotExists
	// StateMachineInstanceNotExists State machine instance not exists
	StateMachineInstanceNotExists
	// ContextVariableReplayFailed Context variable replay failed
	ContextVariableReplayFailed
	// InvalidParameter Context variable replay failed
	InvalidParameter
	// OperationDenied Operation denied
	OperationDenied
)
