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

const (
	// TransactionErrorCodeUnknown Unknown transaction errors code.
	TransactionErrorCodeUnknown = 0

	// TransactionErrorCodeBeginFailed BeginFailed
	TransactionErrorCodeBeginFailed = 1

	// TransactionErrorCodeLockKeyConflict Lock key conflict transaction errors code.
	TransactionErrorCodeLockKeyConflict = 2

	// Io transaction errors code.

	// TransactionErrorCodeBranchRollbackFailedRetriable Branch rollback failed retriable transaction errors code.
	TransactionErrorCodeBranchRollbackFailedRetriable = 3

	// TransactionErrorCodeBranchRollbackFailedUnretriable Branch rollback failed unretriable transaction errors code.
	TransactionErrorCodeBranchRollbackFailedUnretriable = 4

	// TransactionErrorCodeBranchRegisterFailed Branch register failed transaction errors code.
	TransactionErrorCodeBranchRegisterFailed = 5

	// TransactionErrorCodeBranchReportFailed Branch report failed transaction errors code.
	TransactionErrorCodeBranchReportFailed = 6

	// TransactionErrorCodeLockableCheckFailed Lockable check failed transaction errors code.
	TransactionErrorCodeLockableCheckFailed = 7

	// TransactionErrorCodeBranchTransactionNotExist Branch transaction not exist transaction errors code.
	TransactionErrorCodeBranchTransactionNotExist = 8

	// TransactionErrorCodeGlobalTransactionNotExist Global transaction not exist transaction errors code.
	TransactionErrorCodeGlobalTransactionNotExist = 9

	// TransactionErrorCodeGlobalTransactionNotActive Global transaction not active transaction errors code.
	TransactionErrorCodeGlobalTransactionNotActive = 10

	// TransactionErrorCodeGlobalTransactionStatusInvalid Global transaction status invalid transaction errors code.
	TransactionErrorCodeGlobalTransactionStatusInvalid = 11

	// TransactionErrorCodeFailedToSendBranchCommitRequest Failed to send branch commit request transaction errors code.
	TransactionErrorCodeFailedToSendBranchCommitRequest = 12

	// TransactionErrorCodeFailedToSendBranchRollbackRequest Failed to send branch rollback request transaction errors code.
	TransactionErrorCodeFailedToSendBranchRollbackRequest = 13

	// TransactionErrorCodeFailedToAddBranch Failed to add branch transaction errors code.
	TransactionErrorCodeFailedToAddBranch = 14

	// TransactionErrorCodeFailedLockGlobalTranscation Failed to lock global transaction errors code.
	TransactionErrorCodeFailedLockGlobalTranscation = 15

	// TransactionErrorCodeFailedWriteSession FailedWriteSession
	TransactionErrorCodeFailedWriteSession = 16

	// FailedStore Failed to holder errors code
	FailedStore = 17

	// FenceErrorCodeDuplicateKey Insert tcc fence record duplicate key errors
	FenceErrorCodeDuplicateKey = 18

	// InsertRecordError Insert tcc fence record error
	InsertRecordError = 19

	// UpdateRecordError Update tcc fence record error
	UpdateRecordError = 20

	// RecordNotExists TCC fence record already exists
	RecordNotExists = 21
)
