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

package client

import (
	"errors"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

// branchEndResult is the protocol-independent result of a branch operation.
type branchEndResult struct {
	status     branch.BranchStatus
	resultCode message.ResultCode
	errMsg     string
}

func newBranchEndResult(status branch.BranchStatus, bizErr error) branchEndResult {
	result := branchEndResult{status: status, resultCode: message.ResultCodeSuccess}
	if bizErr != nil {
		result.resultCode = message.ResultCodeFailed
		result.errMsg = bizErr.Error()
	}
	return result
}

func branchEndResultCodeProto(resultCode message.ResultCode) pb.ResultCodeProto {
	if resultCode == message.ResultCodeFailed {
		return pb.ResultCodeProto_Failed
	}
	return pb.ResultCodeProto_Success
}

func branchEndSendResponse(
	sendResponse func(int32, interface{}) error,
	fallback func(int32, interface{}) error,
) func(int32, interface{}) error {
	if sendResponse != nil {
		return sendResponse
	}
	return fallback
}

func branchEndProcessError(bizErr, sendErr error) error {
	if sendErr == nil {
		return nil
	}
	if bizErr == nil {
		return sendErr
	}
	return errors.Join(bizErr, sendErr)
}

func branchEndResourceManager(
	getResourceManager func(branch.BranchType) rm.ResourceManager,
	branchType branch.BranchType,
) rm.ResourceManager {
	if getResourceManager != nil {
		return getResourceManager(branchType)
	}
	return rm.GetRmCacheInstance().GetResourceManager(branchType)
}
