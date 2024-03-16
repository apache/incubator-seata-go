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

package tcc

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"

	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/rm"

	"github.com/stretchr/testify/assert"
)

func TestActionContext(t *testing.T) {
	applicationData := `{"actionContext":{"zhangsan":"lisi"}}`
	businessActionContext := GetTCCResourceManagerInstance().
		getBusinessActionContext("1111111111", 2645276141, "TestActionContext", []byte(applicationData))

	assert.NotEmpty(t, businessActionContext)
	bytes, err := json.Marshal(businessActionContext.ActionContext)
	assert.Nil(t, err)
	assert.Equal(t, `{"zhangsan":"lisi"}`, string(bytes))
}

// TestBranchReport
func TestBranchReport(t *testing.T) {
	patches := gomonkey.ApplyMethod(reflect.TypeOf(getty.GetGettyRemotingClient()), "SendSyncRequest", func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
		return message.BranchReportResponse{
			AbstractTransactionResponse: message.AbstractTransactionResponse{
				AbstractResultMessage: message.AbstractResultMessage{
					ResultCode: message.ResultCodeSuccess,
				},
			},
		}, nil
	})

	defer patches.Reset()

	err := GetTCCResourceManagerInstance().BranchReport(
		context.Background(), rm.BranchReportParam{
			BranchType:      branch.BranchTypeTCC,
			Xid:             "1111111111",
			BranchId:        2645276141,
			Status:          branch.BranchStatusPhaseoneDone,
			ApplicationData: `{"actionContext":{"zhangsan":"lisi"}}`,
		})

	assert.Nil(t, err)
}
