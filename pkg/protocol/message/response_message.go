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
	transaction2 "github.com/seata/seata-go/pkg/common/error"
	model2 "github.com/seata/seata-go/pkg/protocol/branch"
)

type AbstractTransactionResponse struct {
	AbstractResultMessage
	TransactionExceptionCode transaction2.TransactionExceptionCode
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
	return MessageType_BranchRegisterResult
}

type BranchReportResponse struct {
	AbstractTransactionResponse
}

func (resp BranchReportResponse) GetTypeCode() MessageType {
	return MessageType_BranchStatusReportResult
}

type BranchCommitResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchCommitResponse) GetTypeCode() MessageType {
	return MessageType_BranchCommitResult
}

type BranchRollbackResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchRollbackResponse) GetTypeCode() MessageType {
	return MessageType_GlobalRollbackResult
}

type GlobalBeginResponse struct {
	AbstractTransactionResponse

	Xid       string
	ExtraData []byte
}

func (resp GlobalBeginResponse) GetTypeCode() MessageType {
	return MessageType_GlobalBeginResult
}

type GlobalStatusResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalStatusResponse) GetTypeCode() MessageType {
	return MessageType_GlobalStatusResult
}

type GlobalLockQueryResponse struct {
	AbstractTransactionResponse

	Lockable bool
}

func (resp GlobalLockQueryResponse) GetTypeCode() MessageType {
	return MessageType_GlobalLockQueryResult
}

type GlobalReportResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalReportResponse) GetTypeCode() MessageType {
	return MessageType_GlobalStatusResult
}

type GlobalCommitResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalCommitResponse) GetTypeCode() MessageType {
	return MessageType_GlobalCommitResult
}

type GlobalRollbackResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalRollbackResponse) GetTypeCode() MessageType {
	return MessageType_GlobalRollbackResult
}
