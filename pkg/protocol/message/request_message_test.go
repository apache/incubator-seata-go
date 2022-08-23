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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchRegisterRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeBranchRegister, BranchRegisterRequest{}.GetTypeCode())
}

func TestBranchReportRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeBranchStatusReport, BranchReportRequest{}.GetTypeCode())
}

func TestBranchCommitRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeBranchCommit, BranchCommitRequest{}.GetTypeCode())
}

func TestBranchRollbackRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeBranchRollback, BranchRollbackRequest{}.GetTypeCode())
}

func TestGlobalBeginRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalBegin, GlobalBeginRequest{}.GetTypeCode())
}

func TestGlobalCommitRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalCommit, GlobalCommitRequest{}.GetTypeCode())
}

func TestGlobalRollbackRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalRollback, GlobalRollbackRequest{}.GetTypeCode())
}

func TestGlobalLockQueryRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalLockQuery, GlobalLockQueryRequest{}.GetTypeCode())
}

func TestGlobalReportRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalReport, GlobalReportRequest{}.GetTypeCode())
}

func TestUndoLogDeleteRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeRmDeleteUndolog, UndoLogDeleteRequest{}.GetTypeCode())
}

func TestRegisterTMRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeRegClt, RegisterTMRequest{}.GetTypeCode())
}

func TestRegisterRMRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeRegRm, RegisterRMRequest{}.GetTypeCode())
}

func TestGlobalStatusRequest_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalStatusResult, GlobalStatusResponse{}.GetTypeCode())
}
