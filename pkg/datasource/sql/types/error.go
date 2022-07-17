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

package types

type ErrorCode int32

const (
	_ ErrorCode = iota
	ErrorCode_Unknown
	ErrorCode_BeginFailed
	ErrorCode_LockKeyConflict
	ErrorCode_IO
	ErrorCode_BranchRollbackFailed_Retriable
	ErrorCode_BranchRollbackFailed_Unretriable
	ErrorCode_BranchRegisterFailed
	ErrorCode_BranchReportFailed
	ErrorCode_LockableCheckFailed
	ErrorCode_BranchTransactionNotExist
	ErrorCode_GlobalTransactionNotExist
	ErrorCode_GlobalTransactionNotActive
	ErrorCode_GlobalTransactionStatusInvalid
	ErrorCode_FailedToSendBranchCommitRequest
	ErrorCode_FailedToSendBranchRollbackRequest
	ErrorCode_FailedToAddBranch
	ErrorCode_FailedWriteSession
	ErrorCode_FailedLockGlobalTranscation
	ErrorCode_FailedStore
	ErrorCode_LockKeyConflictFailFast
)

type TransactionError struct {
	code ErrorCode
	msg  string
}

func (e *TransactionError) Error() string {
	return e.msg
}

func (e *TransactionError) Code() ErrorCode {
	return e.code
}
