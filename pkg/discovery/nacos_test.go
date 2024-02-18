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

package discovery

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/discovery/mock"
)

func TestNacosRegistryServiceLookup(t *testing.T) {
	cases := []struct {
		name              string
		instances         []model.Instance
		expectedInstances []*ServiceInstance
	}{
		{
			name: "case 1",
			instances: []model.Instance{
				{
					Ip:      "127.0.0.1",
					Port:    8091,
					Healthy: true,
					Enable:  true,
				},
				{
					Ip:      "127.0.0.1",
					Port:    8092,
					Healthy: true,
					Enable:  false,
				},
			},
			expectedInstances: []*ServiceInstance{
				{
					Addr: "127.0.0.1",
					Port: 8091,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	mockClient := mock.NewMockNacosClient(ctrl)
	na := &NacosRegistryService{
		client:   mockClient,
		registry: &NacosRegistry{},
		cluster:  "",
	}
	for _, ca := range cases {
		t.Run(ca.name, func(t *testing.T) {
			mockClient.EXPECT().SelectInstances(gomock.Any()).Return(ca.instances, nil)

			ins, err := na.Lookup("")
			assert.NoError(t, err)

			assert.True(t, reflect.DeepEqual(ins, ca.expectedInstances))

			// reset instances
			na.serverInstances = nil
		})
	}
}
