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
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/discovery/mock"
)

// ZookeeperWatchEvent is used to simulate ZooKeeper watch events in tests
type ZookeeperWatchEvent struct {
	EventType string
	Path      string
	Data      []byte
}

// TestZookeeperRegistryService_Lookup tests the Lookup method in the ZooKeeper service discovery
func TestZookeeperRegistryService_Lookup(t *testing.T) {

	tests := []struct {
		name      string
		getData   []byte
		watchResp *ZookeeperWatchEvent
		want      []*ServiceInstance
	}{
		{
			name:      "normal",
			getData:   []byte("172.0.0.1:8091"),
			watchResp: nil,
			want: []*ServiceInstance{
				{
					Addr: "172.0.0.1",
					Port: 8091,
				},
			},
		},
		{
			name:    "use watch update ServiceInstances",
			getData: []byte("172.0.0.1:8091"), // Newly added node data
			watchResp: &ZookeeperWatchEvent{
				EventType: "put",
				Path:      "/registry-seata/default/172.0.0.1:8091",
				Data:      []byte("172.0.0.1:8091"),
			},
			want: []*ServiceInstance{
				{
					Addr: "172.0.0.1",
					Port: 8091,
				},
			},
		},
		{
			name:    "use watch del ServiceInstances",
			getData: []byte("172.0.0.1:8092"), // Retained node data
			watchResp: &ZookeeperWatchEvent{
				EventType: "delete",
				Path:      "/registry-seata/default/172.0.0.1:8091",
				Data:      []byte("172.0.0.1:8091"),
			},
			want: []*ServiceInstance{
				{
					Addr: "172.0.0.1",
					Port: 8092,
				},
			},
		},
	}

	for _, tt := range tests {
		ctrl := gomock.NewController(t)
		mockZkClient := mock.NewMockZkConnInterface(ctrl)

		zkRegistryService := &ZookeeperRegistryService{
			conn: mockZkClient,
			vgroupMapping: map[string]string{
				"default_tx_group": "default",
			},
			grouplist: make(map[string][]*ServiceInstance),
			stopCh:    make(chan struct{}),
		}

		mockZkClient.EXPECT().Get(gomock.Any()).Return(tt.getData, nil, nil).AnyTimes()
		eventCh := make(chan zk.Event)
		mockZkClient.EXPECT().GetW(gomock.Any()).Return(tt.getData, nil, eventCh, nil).AnyTimes()
		mockZkClient.EXPECT().Close().Return(nil).AnyTimes()

		// Start watch listening
		go func() {
			zkRegistryService.watch("/registry-seata/default")
		}()

		time.Sleep(1 * time.Second)

		if tt.watchResp != nil {
			var zkEvent zk.Event
			switch tt.watchResp.EventType {
			case "put":
				zkEvent = zk.Event{
					Type:  zk.EventNodeDataChanged, // Simulate data update or creation event
					Path:  tt.watchResp.Path,
					State: zk.StateConnected,
				}
			case "delete":
				zkEvent = zk.Event{
					Type:  zk.EventNodeDeleted,
					Path:  tt.watchResp.Path,
					State: zk.StateConnected,
				}
			}
			go func() {
				eventCh <- zkEvent
			}()
		}

		time.Sleep(1 * time.Second)
		serviceInstances, err := zkRegistryService.Lookup("default_tx_group")
		if err != nil {
			t.Errorf("Error occurred during lookup: %v", err)
		}
		t.Logf("Test case: %s", tt.name)
		for _, si := range serviceInstances {
			t.Log(si.Addr, si.Port)
		}
		assert.True(t, reflect.DeepEqual(serviceInstances, tt.want))

		zkRegistryService.Close()
	}
}
