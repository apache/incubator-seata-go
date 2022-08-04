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

package error

import "github.com/pkg/errors"

var (
	ErrorTooManySessions  = errors.New("too many seeessions")
	ErrorHeartBeatTimeOut = errors.New("heart beat time out")
)

type TransactionExceptionCode byte

const (
	/**
	 * Unknown transaction exception code.
	 */
	TransactionExceptionCodeUnknown = TransactionExceptionCode(0)

	/**
	 * BeginFailed
	 */
	TransactionExceptionCodeBeginFailed = 1

	/**
	 * Lock key conflict transaction exception code.
	 */
	TransactionExceptionCodeLockKeyConflict = 2

	/**
	 * Io transaction exception code.
	 */
	IO = 3
	/**
	 * Branch rollback failed retriable transaction exception code.
	 */
	TransactionExceptionCodeBranchRollbackFailedRetriable = 4

	/**
	 * Branch rollback failed unretriable transaction exception code.
	 */
	TransactionExceptionCodeBranchRollbackFailedUnretriable = 5

	/**
	 * Branch register failed transaction exception code.
	 */
	TransactionExceptionCodeBranchRegisterFailed = 6

	/**
	 * Branch report failed transaction exception code.
	 */
	TransactionExceptionCodeBranchReportFailed = 7

	/**
	 * Lockable check failed transaction exception code.
	 */
	TransactionExceptionCodeLockableCheckFailed = 8

	/**
	 * Branch transaction not exist transaction exception code.
	 */
	TransactionExceptionCodeBranchTransactionNotExist = 9

	/**
	 * Global transaction not exist transaction exception code.
	 */
	TransactionExceptionCodeGlobalTransactionNotExist = 10

	/**
	 * Global transaction not active transaction exception code.
	 */
	TransactionExceptionCodeGlobalTransactionNotActive = 11

	/**
	 * Global transaction status invalid transaction exception code.
	 */
	TransactionExceptionCodeGlobalTransactionStatusInvalid = 12

	/**
	 * Failed to send branch commit request transaction exception code.
	 */
	TransactionExceptionCodeFailedToSendBranchCommitRequest = 13

	/**
	 * Failed to send branch rollback request transaction exception code.
	 */
	TransactionExceptionCodeFailedToSendBranchRollbackRequest = 14

	/**
	 * Failed to add branch transaction exception code.
	 */
	TransactionExceptionCodeFailedToAddBranch = 15

	/**
	 * Failed to lock global transaction exception code.
	 */
	TransactionExceptionCodeFailedLockGlobalTranscation = 16

	/**
	 * FailedWriteSession
	 */
	TransactionExceptionCodeFailedWriteSession = 17

	/**
	 * Failed to holder exception code
	 */
	FailedStore = 18

	/**
	 * Lock key conflict fail fast transaction exception code.
	 */
	LockKeyConflictFailFast = 19
)

type TransactionException struct {
	Code    TransactionExceptionCode
	Message string
}

func (e TransactionException) Error() string {
	return "TransactionException: " + e.Message
}
