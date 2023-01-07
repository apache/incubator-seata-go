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

var MagicCodeBytes = [2]byte{0xda, 0xda}

type (
	MessageType      int
	GettyRequestType byte
	GlobalStatus     byte
)

const (
	/**
	 * The constant TypeGlobalBegin.
	 */
	MessageTypeGlobalBegin MessageType = iota + 1
	/**
	 * The constant TypeGlobalBeginResult.
	 */
	MessageTypeGlobalBeginResult
	/**
	 * The constant TypeBranchCommit.
	 */
	MessageTypeBranchCommit
	/**
	 * The constant TypeBranchCommitResult.
	 */
	MessageTypeBranchCommitResult
	/**
	 * The constant TypeBranchRollback.
	 */
	MessageTypeBranchRollback
	/**
	 * The constant TypeBranchRollbackResult.
	 */
	MessageTypeBranchRollbackResult
	/**
	 * The constant TypeGlobalCommit.
	 */
	MessageTypeGlobalCommit
	/**
	 * The constant TypeGlobalCommitResult.
	 */
	MessageTypeGlobalCommitResult
	/**
	 * The constant TypeGlobalRollback.
	 */
	MessageTypeGlobalRollback
	/**
	 * The constant TypeGlobalRollbackResult.
	 */
	MessageTypeGlobalRollbackResult
	/**
	 * The constant TypeBranchRegister.
	 */
	MessageTypeBranchRegister
	/**
	 * The constant TypeBranchRegisterResult.
	 */
	MessageTypeBranchRegisterResult
	/**
	 * The constant TypeBranchStatusReport.
	 */
	MessageTypeBranchStatusReport
	/**
	 * The constant TypeBranchStatusReportResult.
	 */
	MessageTypeBranchStatusReportResult
	/**
	 * The constant TypeGlobalStatus.
	 */
	MessageTypeGlobalStatus
	/**
	 * The constant TypeGlobalStatusResult.
	 */
	MessageTypeGlobalStatusResult
	/**
	 * The constant TypeGlobalReport.
	 */
	MessageTypeGlobalReport
	/**
	 * The constant TypeGlobalReportResult.
	 */
	MessageTypeGlobalReportResult
	/**
	 * The constant TypeGlobalLockQuery.
	 */
	MessageTypeGlobalLockQuery MessageType = iota + 3
	/**
	 * The constant TypeGlobalLockQueryResult.
	 */
	MessageTypeGlobalLockQueryResult

	/**
	 * The constant TypeSeataMerge.
	 */
	MessageTypeSeataMerge MessageType = iota + 39
	/**
	 * The constant TypeSeataMergeResult.
	 */
	MessageTypeSeataMergeResult

	/**
	 * The constant TypeRegClt.
	 */
	MessageTypeRegClt MessageType = iota + 79
	/**
	 * The constant TypeRegCltResult.
	 */
	MessageTypeRegCltResult
	/**
	 * The constant TypeRegRm.
	 */
	MessageTypeRegRm
	/**
	 * The constant TypeRegRmResult.
	 */
	MessageTypeRegRmResult
	/**
	 * The constant TypeRmDeleteUndolog.
	 */
	MessageTypeRmDeleteUndolog MessageType = iota + 85
	/**
	 * the constant TypeHeartbeatMsg
	 */
	MessageTypeHeartbeatMsg MessageType = iota + 93

	/**
	 * the constant MessageTypeBatchResultMsg
	 */
	MessageTypeBatchResultMsg
)

const (
	VERSION = 1

	// MaxFrameLength max frame length
	MaxFrameLength = 8 * 1024 * 1024

	// V1HeadLength v1 head length
	V1HeadLength = 16

	// Request message type
	GettyRequestTypeRequestSync GettyRequestType = 0

	// Response message type
	GettyRequestTypeResponse GettyRequestType = 1

	// Request which no need response
	GettyRequestTypeRequestOneway GettyRequestType = 2

	// Heartbeat Request
	GettyRequestTypeHeartbeatRequest GettyRequestType = 3

	// Heartbeat Response
	GettyRequestTypeHeartbeatResponse GettyRequestType = 4
)

const (

	/**
	 * Un known global status.
	 */
	// Unknown
	GlobalStatusUnKnown GlobalStatus = 0

	/**
	 * The GlobalStatusBegin.
	 */
	// PHASE 1: can accept new branch registering.
	GlobalStatusBegin GlobalStatus = 1

	/**
	 * PHASE 2: Running Status: may be changed any time.
	 */
	// Committing.
	GlobalStatusCommitting GlobalStatus = 2

	/**
	 * The Commit retrying.
	 */
	// Retrying commit after a recoverable failure.
	GlobalStatusCommitRetrying GlobalStatus = 3

	/**
	 * Rollbacking global status.
	 */
	// Rollbacking
	GlobalStatusRollbacking GlobalStatus = 4

	/**
	 * The Rollback retrying.
	 */
	// Retrying rollback after a recoverable failure.
	GlobalStatusRollbackRetrying GlobalStatus = 5

	/**
	 * The Timeout rollbacking.
	 */
	// Rollbacking since timeout
	GlobalStatusTimeoutRollbacking GlobalStatus = 6

	/**
	 * The Timeout rollback retrying.
	 */
	// Retrying rollback  GlobalStatus = since timeout) after a recoverable failure.
	GlobalStatusTimeoutRollbackRetrying GlobalStatus = 7

	/**
	 * All branches can be async committed. The committing is NOT done yet, but it can be seen as committed for TM/RM
	 * client.
	 */
	GlobalStatusAsyncCommitting GlobalStatus = 8

	/**
	 * PHASE 2: Final Status: will NOT change any more.
	 */
	// Finally: global transaction is successfully committed.
	GlobalStatusCommitted GlobalStatus = 9

	/**
	 * The Commit failed.
	 */
	// Finally: failed to commit
	GlobalStatusCommitFailed GlobalStatus = 10

	/**
	 * The Rollbacked.
	 */
	// Finally: global transaction is successfully rollbacked.
	GlobalStatusRollbacked GlobalStatus = 11

	/**
	 * The Rollback failed.
	 */
	// Finally: failed to rollback
	GlobalStatusRollbackFailed GlobalStatus = 12

	/**
	 * The Timeout rollbacked.
	 */
	// Finally: global transaction is successfully rollbacked since timeout.
	GlobalStatusTimeoutRollbacked GlobalStatus = 13

	/**
	 * The Timeout rollback failed.
	 */
	// Finally: failed to rollback since timeout
	GlobalStatusTimeoutRollbackFailed GlobalStatus = 14

	/**
	 * The Finished.
	 */
	// Not managed in session MAP any more
	GlobalStatusFinished GlobalStatus = 15
)
