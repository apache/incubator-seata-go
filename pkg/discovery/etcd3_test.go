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
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"reflect"
	"seata.apache.org/seata-go/pkg/discovery/mock"
	"testing"
	"time"
)

func TestEtcd3RegistryService_Lookup(t *testing.T) {

	tests := []struct {
		name      string
		getResp   *clientv3.GetResponse
		watchResp *clientv3.WatchResponse
		want      []*ServiceInstance
	}{
		{
			name: "normal",
			getResp: &clientv3.GetResponse{
				Kvs: []*mvccpb.KeyValue{
					{
						Key:   []byte("registry-seata-default-172.0.0.1:8091"),
						Value: []byte("172.0.0.1:8091"),
					},
				},
			},
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
			getResp: nil,
			watchResp: &clientv3.WatchResponse{
				Events: []*clientv3.Event{
					{
						Type: clientv3.EventTypePut,
						Kv: &mvccpb.KeyValue{
							Key:   []byte("registry-seata-default-172.0.0.1:8091"),
							Value: []byte("172.0.0.1:8091"),
						},
					},
				},
			},
			want: []*ServiceInstance{
				{
					Addr: "172.0.0.1",
					Port: 8091,
				},
			},
		},
		{
			name: "use watch del ServiceInstances",
			getResp: &clientv3.GetResponse{
				Kvs: []*mvccpb.KeyValue{
					{
						Key:   []byte("registry-seata-default-172.0.0.1:8091"),
						Value: []byte("172.0.0.1:8091"),
					},
					{
						Key:   []byte("registry-seata-default-172.0.0.1:8092"),
						Value: []byte("172.0.0.1:8092"),
					},
				},
			},
			watchResp: &clientv3.WatchResponse{
				Events: []*clientv3.Event{
					{
						Type: clientv3.EventTypeDelete,
						Kv: &mvccpb.KeyValue{
							Key:   []byte("registry-seata-default-172.0.0.1:8091"),
							Value: []byte("172.0.0.1:8091"),
						},
					},
				},
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
		mockEtcdClient := mock.NewMockEtcdClient(ctrl)
		etcdRegistryService := &EtcdRegistryService{
			client: &clientv3.Client{
				KV:      mockEtcdClient,
				Watcher: mockEtcdClient,
			},
			vgroupMapping: map[string]string{
				"default_tx_group": "default",
			},
			grouplist: make(map[string][]*ServiceInstance, 0),
			stopCh:    make(chan struct{}),
		}

		mockEtcdClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.getResp, nil)
		ch := make(chan clientv3.WatchResponse)
		mockEtcdClient.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any()).Return(ch)

		go func() {
			etcdRegistryService.watch("registry-seata")
		}()
		// wait a second for watch
		time.Sleep(1 * time.Second)

		if tt.watchResp != nil {
			go func() {
				ch <- *tt.watchResp
			}()
		}

		// wait one more second for update
		time.Sleep(1 * time.Second)
		serviceInstances, err := etcdRegistryService.Lookup("default_tx_group")
		if err != nil {
			t.Errorf("error happen when look up . err = %e", err)
		}
		t.Logf(tt.name)
		for i := range serviceInstances {
			t.Log(serviceInstances[i].Addr)
			t.Log(serviceInstances[i].Port)
		}
		assert.True(t, reflect.DeepEqual(serviceInstances, tt.want))

		etcdRegistryService.Close()
	}
}
