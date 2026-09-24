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
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"seata.apache.org/seata-go/v2/pkg/discovery/mock"
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
			name: "host is ipv6",
			getResp: &clientv3.GetResponse{
				Kvs: []*mvccpb.KeyValue{
					{
						Key:   []byte("registry-seata-default-2000:0000:0000:0000:0001:2345:6789:abcd:8091"),
						Value: []byte("2000:0000:0000:0000:0001:2345:6789:abcd:8091"),
					},
				},
			},
			watchResp: nil,
			want: []*ServiceInstance{
				{
					Addr: "2000:0000:0000:0000:0001:2345:6789:abcd",
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
			client: newTestEtcdClient(mockEtcdClient),
			vgroupMapping: map[string]string{
				"default_tx_group": "default",
			},
			store:  NewAddressStore(),
			stopCh: make(chan struct{}),
		}

		mockEtcdClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.getResp, nil)
		ch := make(chan clientv3.WatchResponse)
		mockEtcdClient.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any()).Return(ch)
		mockEtcdClient.EXPECT().Close().Return(nil)

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

func TestEtcd3RegistryService_CloseIsRepeatable(t *testing.T) {
	client := clientv3.NewCtxClient(context.Background())
	etcdRegistryService := &EtcdRegistryService{
		client: client,
		stopCh: make(chan struct{}),
	}

	etcdRegistryService.Close()
	etcdRegistryService.Close()

	select {
	case <-etcdRegistryService.stopCh:
	case <-time.After(time.Second):
		t.Fatal("stop channel was not closed")
	}

	select {
	case <-client.Ctx().Done():
	case <-time.After(time.Second):
		t.Fatal("etcd client was not closed")
	}
}

func TestEtcd3RegistryService_WatchContinuesAfterSnapshotRevision(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockEtcdClient := mock.NewMockEtcdClient(ctrl)
	service := &EtcdRegistryService{
		client: newTestEtcdClient(mockEtcdClient),
		store:  NewAddressStore(),
		stopCh: make(chan struct{}),
	}

	mockEtcdClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(&clientv3.GetResponse{
		Header: &etcdserverpb.ResponseHeader{Revision: 41},
	}, nil)
	watchStarted := make(chan struct{})
	watchCh := make(chan clientv3.WatchResponse)
	mockEtcdClient.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key string, options ...clientv3.OpOption) clientv3.WatchChan {
			op := clientv3.OpGet(key, options...)
			assert.Equal(t, int64(42), op.Rev())
			close(watchStarted)
			return watchCh
		})
	mockEtcdClient.EXPECT().Close().Return(nil)

	watchDone := make(chan struct{})
	go func() {
		service.watch(etcdClusterPrefix)
		close(watchDone)
	}()
	waitForSignal(t, watchStarted, "etcd watch to start")
	service.Close()
	waitForSignal(t, watchDone, "etcd watch to stop")
}

func newTestEtcdClient(client mock.EtcdClient) *clientv3.Client {
	return clientv3.NewCtxClient(
		context.Background(),
		func(c *clientv3.Client) {
			c.KV = client
			c.Watcher = client
		},
	)
}

func waitForSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}
