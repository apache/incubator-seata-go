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

package getty

import (
	"testing"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	"seata.apache.org/seata-go/v2/pkg/rm"

	"github.com/stretchr/testify/assert"
)

func TestGetGettyRMRemotingInstance(t *testing.T) {
	rm.SetRMRemotingInstance(&GettyRMRemoting{})
	tests := []struct {
		name     string
		protocol string
		wantType interface{}
	}{
		{
			name:     "GettyRMRemoting",
			protocol: "seata",
			wantType: &GettyRMRemoting{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			config.InitTransportConfig(&config.TransportConfig{
				Protocol: tt.protocol,
			})

			got := rm.GetRMRemotingInstance()
			assert.NotNil(t, got)
			assert.IsType(t, tt.wantType, got)
		})
	}
}

// TestDecodeBranchRegisterResponse covers the nil / wrong-type / failure /
// success paths; the historic code logged err.Error() with a possibly nil
// err (panic) and bare-asserted the response type.
func TestDecodeBranchRegisterResponse(t *testing.T) {
	success := message.BranchRegisterResponse{
		AbstractTransactionResponse: message.AbstractTransactionResponse{
			AbstractResultMessage: message.AbstractResultMessage{ResultCode: message.ResultCodeSuccess},
		},
		BranchId: 42,
	}
	failed := message.BranchRegisterResponse{AbstractTransactionResponse: message.AbstractTransactionResponse{AbstractResultMessage: message.AbstractResultMessage{ResultCode: message.ResultCodeFailed, Msg: "boom"}}}

	branchId, err := decodeBranchRegisterResponse(success)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), branchId)

	_, err = decodeBranchRegisterResponse(failed)
	assert.Error(t, err)

	_, err = decodeBranchRegisterResponse(nil)
	assert.Error(t, err)

	_, err = decodeBranchRegisterResponse("not-a-response")
	assert.Error(t, err)
}

// TestHandleRegisterRMResponse verifies that a TC-side rejection
// (Identified=false) is surfaced as an error instead of being swallowed with
// only an info log (which made registrations report a healthy startup).
func TestHandleRegisterRMResponse(t *testing.T) {
	rejected := message.RegisterRMResponse{AbstractIdentifyResponse: message.AbstractIdentifyResponse{Identified: false, AbstractResultMessage: message.AbstractResultMessage{Msg: "auth failed"}}}
	accepted := message.RegisterRMResponse{AbstractIdentifyResponse: message.AbstractIdentifyResponse{Identified: true}}

	var successSeen, failureSeen bool
	err := handleRegisterRMResponse(accepted,
		func(message.RegisterRMResponse) { successSeen = true },
		func(message.RegisterRMResponse) { failureSeen = true })
	assert.NoError(t, err)
	assert.True(t, successSeen)

	err = handleRegisterRMResponse(rejected,
		func(message.RegisterRMResponse) { successSeen = true },
		func(message.RegisterRMResponse) { failureSeen = true })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "register RM failed")
	assert.True(t, failureSeen)

	err = handleRegisterRMResponse(nil,
		func(message.RegisterRMResponse) {},
		func(message.RegisterRMResponse) {})
	assert.Error(t, err)

	err = handleRegisterRMResponse("wrong-type",
		func(message.RegisterRMResponse) {},
		func(message.RegisterRMResponse) {})
	assert.Error(t, err)
}
