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

func TestRegisterRMResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeRegRmResult, RegisterRMResponse{}.GetTypeCode())
}

func TestRegisterTMResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeRegCltResult, RegisterTMResponse{}.GetTypeCode())
}

func TestGlobalReportResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalReportResult, GlobalReportResponse{}.GetTypeCode())
}

func TestGlobalLockQueryResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalLockQueryResult, GlobalLockQueryResponse{}.GetTypeCode())
}

func TestGlobalRollbackResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalRollbackResult, GlobalRollbackResponse{}.GetTypeCode())
}

func TestGlobalCommitResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalCommitResult, GlobalCommitResponse{}.GetTypeCode())
}

func TestGlobalBeginResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalBeginResult, GlobalBeginResponse{}.GetTypeCode())
}

func TestBranchRollbackResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeBranchRollbackResult, BranchRollbackResponse{}.GetTypeCode())
}

func TestBranchCommitResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeBranchCommitResult, BranchCommitResponse{}.GetTypeCode())
}

func TestBranchRegisterResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeBranchRegisterResult, BranchRegisterResponse{}.GetTypeCode())
}

func TestBranchReportResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeBranchStatusReportResult, BranchReportResponse{}.GetTypeCode())
}

func TestGlobalStatusResponse_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageTypeGlobalStatusResult, GlobalStatusResponse{}.GetTypeCode())
}
