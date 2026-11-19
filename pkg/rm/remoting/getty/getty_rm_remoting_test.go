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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	dubbogetty "github.com/apache/dubbo-getty"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	remotinggetty "seata.apache.org/seata-go/v2/pkg/remoting/getty"
	remotingmock "seata.apache.org/seata-go/v2/pkg/remoting/mock"
	"seata.apache.org/seata-go/v2/pkg/rm"
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

type registerRMTestResource struct{ id string }

func (r registerRMTestResource) GetResourceGroupId() string { return "group" }
func (r registerRMTestResource) GetResourceId() string      { return r.id }
func (r registerRMTestResource) GetBranchType() branch.BranchType {
	return branch.BranchTypeTCC
}

func registerRMResponse(identified bool, msg string) message.RegisterRMResponse {
	return message.RegisterRMResponse{AbstractIdentifyResponse: message.AbstractIdentifyResponse{
		AbstractResultMessage: message.AbstractResultMessage{Msg: msg},
		Identified:            identified,
	}}
}

func TestGettyRMRemotingRegisterResourceOnConnection(t *testing.T) {
	ctl := gomock.NewController(t)
	target := remotingmock.NewMockTestSession(ctl)
	target.EXPECT().IsClosed().Return(false)
	var gotTarget interface{}
	var gotRequest message.RegisterRMRequest
	patches := gomonkey.ApplyMethod(reflect.TypeOf(remotinggetty.GetGettyRemotingClient()), "SendSyncRequestOnSession",
		func(_ *remotinggetty.GettyRemotingClient, msg interface{}, session dubbogetty.Session) (interface{}, error) {
			gotTarget = session
			gotRequest = msg.(message.RegisterRMRequest)
			return registerRMResponse(true, ""), nil
		})
	defer patches.Reset()

	err := (&GettyRMRemoting{}).RegisterResourceOnConnection(
		registerRMTestResource{id: "rm-resource"}, target)
	assert.NoError(t, err)
	assert.Same(t, target, gotTarget)
	assert.Equal(t, "rm-resource", gotRequest.ResourceIds)
}

func TestGettyRMRemotingRegisterResourceOnConnectionRejectsClosedSession(t *testing.T) {
	ctl := gomock.NewController(t)
	session := remotingmock.NewMockTestSession(ctl)
	session.EXPECT().IsClosed().Return(true)
	err := (&GettyRMRemoting{}).RegisterResourceOnConnection(
		registerRMTestResource{id: "rm-resource"}, session)
	assert.ErrorContains(t, err, "session for resource registration is closed")
}

type wrongConnection struct{}

func (wrongConnection) RemoteAddr() string { return "127.0.0.1:8091" }
func (wrongConnection) IsClosed() bool     { return false }
