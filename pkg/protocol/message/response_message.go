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

import (
	model2 "seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/util/errors"
)

type AbstractTransactionResponse struct {
	AbstractResultMessage
	TransactionErrorCode errors.TransactionErrorCode
}

type AbstractBranchEndResponse struct {
	AbstractTransactionResponse
	Xid          string
	BranchId     int64
	BranchStatus model2.BranchStatus
}

type AbstractGlobalEndResponse struct {
	AbstractTransactionResponse
	GlobalStatus GlobalStatus
}

type BranchRegisterResponse struct {
	AbstractTransactionResponse
	BranchId int64
}

func (resp BranchRegisterResponse) GetTypeCode() MessageType {
	return MessageTypeBranchRegisterResult
}

type RegisterTMResponse struct {
	AbstractIdentifyResponse
}

func (resp RegisterTMResponse) GetTypeCode() MessageType {
	return MessageTypeRegCltResult
}

type BranchReportResponse struct {
	AbstractTransactionResponse
}

func (resp BranchReportResponse) GetTypeCode() MessageType {
	return MessageTypeBranchStatusReportResult
}

type BranchCommitResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchCommitResponse) GetTypeCode() MessageType {
	return MessageTypeBranchCommitResult
}

type BranchRollbackResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchRollbackResponse) GetTypeCode() MessageType {
	return MessageTypeBranchRollbackResult
}

type GlobalBeginResponse struct {
	AbstractTransactionResponse

	Xid       string
	ExtraData []byte
}

func (resp GlobalBeginResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalBeginResult
}

type GlobalStatusResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalStatusResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalStatusResult
}

type GlobalLockQueryResponse struct {
	AbstractTransactionResponse

	Lockable bool
}

func (resp GlobalLockQueryResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalLockQueryResult
}

type GlobalReportResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalReportResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalReportResult
}

type GlobalCommitResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalCommitResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalCommitResult
}

type GlobalRollbackResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalRollbackResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalRollbackResult
}

type RegisterRMResponse struct {
	AbstractIdentifyResponse
}

func (resp RegisterRMResponse) GetTypeCode() MessageType {
	return MessageTypeRegRmResult
}
