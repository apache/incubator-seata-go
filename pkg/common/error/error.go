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
	Error_TooManySessions  = errors.New("too many seeessions")
	Error_HeartBeatTimeOut = errors.New("heart beat time out")
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

	/**
	 * Branch rollback failed retriable transaction exception code.
	 */
	TransactionExceptionCodeBranchRollbackFailedRetriable = 3

	/**
	 * Branch rollback failed unretriable transaction exception code.
	 */
	TransactionExceptionCodeBranchRollbackFailedUnretriable = 4

	/**
	 * Branch register failed transaction exception code.
	 */
	TransactionExceptionCodeBranchRegisterFailed = 5

	/**
	 * Branch report failed transaction exception code.
	 */
	TransactionExceptionCodeBranchReportFailed = 6

	/**
	 * Lockable check failed transaction exception code.
	 */
	TransactionExceptionCodeLockableCheckFailed = 7

	/**
	 * Branch transaction not exist transaction exception code.
	 */
	TransactionExceptionCodeBranchTransactionNotExist = 8

	/**
	 * Global transaction not exist transaction exception code.
	 */
	TransactionExceptionCodeGlobalTransactionNotExist = 9

	/**
	 * Global transaction not active transaction exception code.
	 */
	TransactionExceptionCodeGlobalTransactionNotActive = 10

	/**
	 * Global transaction status invalid transaction exception code.
	 */
	TransactionExceptionCodeGlobalTransactionStatusInvalid = 11

	/**
	 * Failed to send branch commit request transaction exception code.
	 */
	TransactionExceptionCodeFailedToSendBranchCommitRequest = 12

	/**
	 * Failed to send branch rollback request transaction exception code.
	 */
	TransactionExceptionCodeFailedToSendBranchRollbackRequest = 13

	/**
	 * Failed to add branch transaction exception code.
	 */
	TransactionExceptionCodeFailedToAddBranch = 14

	/**
	 * Failed to lock global transaction exception code.
	 */
	TransactionExceptionCodeFailedLockGlobalTranscation = 15

	/**
	 * FailedWriteSession
	 */
	TransactionExceptionCodeFailedWriteSession = 16

	/**
	 * Failed to holder exception code
	 */
	FailedStore = 17
)

type TransactionException struct {
	Code    TransactionExceptionCode
	Message string
}

func (e TransactionException) Error() string {
	return "TransactionException: " + e.Message
}
