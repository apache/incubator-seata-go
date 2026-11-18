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
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	etcd3 "go.etcd.io/etcd/client/v3"

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

// TestWatch_SurvivesMalformedEvents verifies the watch goroutine keeps
// consuming events after foreign/malformed keys under the prefix, instead
// of exiting permanently and freezing the address list.
func TestWatch_SurvivesMalformedEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockEtcdClient := mock.NewMockEtcdClient(ctrl)
	service := &EtcdRegistryService{
		client:        newTestEtcdClient(mockEtcdClient),
		store:         NewAddressStore(),
		stopCh:        make(chan struct{}),
		vgroupMapping: map[string]string{"default": "default"},
	}

	// Get/Watch are expected repeatedly: the self-healing watch re-establishes
	// itself after a closed channel.
	mockEtcdClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(&clientv3.GetResponse{Header: &etcdserverpb.ResponseHeader{Revision: 41}}, nil).AnyTimes()
	watchStarted := make(chan struct{})
	var onceClose sync.Once
	watchCh := make(chan clientv3.WatchResponse)
	mockEtcdClient.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key string, options ...clientv3.OpOption) clientv3.WatchChan {
			onceClose.Do(func() { close(watchStarted) })
			return watchCh
		}).AnyTimes()
	mockEtcdClient.EXPECT().Close().Return(nil).AnyTimes()

	goodKey := []byte("registry-seata-default-127.0.0.2:8091")
	// Start the watch goroutine BEFORE sending events: watchCh is
	// unbuffered, so an early send would block forever.
	watchDone := make(chan struct{})
	go func() {
		service.watch(etcdClusterPrefix)
		close(watchDone)
	}()
	waitForSignal(t, watchStarted, "etcd watch to start")

	watchCh <- clientv3.WatchResponse{Events: []*etcd3.Event{
		// foreign key: not parseable as a registry entry
		{Kv: &mvccpb.KeyValue{Key: []byte("foreign-key"), Value: []byte("x")}},
		// malformed value for a well-formed key
		{Kv: &mvccpb.KeyValue{Key: goodKey, Value: []byte("not-an-address")}},
		// a valid instance after the malformed ones
		{Kv: &mvccpb.KeyValue{Key: goodKey, Value: []byte("127.0.0.2:8091")}},
	}}

	// Wait until the valid event has been applied.
	waitFor(t, func() bool {
		instances, err := service.Lookup("default")
		return err == nil && len(instances) == 1 && instances[0].Addr == "127.0.0.2" && instances[0].Port == 8091
	}, "valid instance to be applied after malformed events")

	// The watch goroutine must still be alive: a second valid event is
	// applied too (before the fix the goroutine had already exited).
	watchCh <- clientv3.WatchResponse{Events: []*etcd3.Event{
		{Kv: &mvccpb.KeyValue{Key: []byte("registry-seata-default-127.0.0.3:8091"), Value: []byte("127.0.0.3:8091")}},
	}}
	waitFor(t, func() bool {
		instances, _ := service.Lookup("default")
		return len(instances) == 2
	}, "second instance to be applied (watch still alive)")

	close(service.stopCh)
	select {
	case <-watchDone:
	case <-time.After(3 * time.Second):
		t.Fatal("watch did not stop")
	}
}

func waitFor(t *testing.T, cond func() bool, name string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", name)
}

// TestWatch_RebuildRemovesStaleInstances verifies that an
// instance deleted while the watch was being rebuilt must disappear from
// the store when the rebuild snapshot reconciles it (previously the stale
// instance lingered forever and kept receiving traffic).
func TestWatch_RebuildRemovesStaleInstances(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockEtcdClient := mock.NewMockEtcdClient(ctrl)
	service := &EtcdRegistryService{
		client:        newTestEtcdClient(mockEtcdClient),
		store:         NewAddressStore(),
		stopCh:        make(chan struct{}),
		vgroupMapping: map[string]string{"default": "default"},
	}

	snapshotCalls := 0
	mockEtcdClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
			snapshotCalls++
			resp := &clientv3.GetResponse{Header: &etcdserverpb.ResponseHeader{Revision: int64(100 + snapshotCalls)}}
			if snapshotCalls == 1 {
				// Initial snapshot: two instances.
				resp.Kvs = []*mvccpb.KeyValue{
					{Key: []byte("registry-seata-default-127.0.0.1:8091"), Value: []byte("127.0.0.1:8091")},
					{Key: []byte("registry-seata-default-127.0.0.2:8091"), Value: []byte("127.0.0.2:8091")},
				}
			} else {
				// Rebuild snapshots: .1 has been deleted meanwhile — only .2
				// remains, exercising the reconciliation.
				resp.Kvs = []*mvccpb.KeyValue{
					{Key: []byte("registry-seata-default-127.0.0.2:8091"), Value: []byte("127.0.0.2:8091")},
				}
			}
			return resp, nil
		}).AnyTimes()

	watchStarted := make(chan struct{})
	var onceWatch sync.Once
	watchCh := make(chan clientv3.WatchResponse)
	mockEtcdClient.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key string, options ...clientv3.OpOption) clientv3.WatchChan {
			onceWatch.Do(func() { close(watchStarted) })
			return watchCh
		}).AnyTimes()
	mockEtcdClient.EXPECT().Close().Return(nil).AnyTimes()

	// Seed the store with both instances as if the first snapshot had been
	// applied, then close the channel to force a rebuild.
	service.store.upsert("default", &ServiceInstance{Addr: "127.0.0.1", Port: 8091})
	service.store.upsert("default", &ServiceInstance{Addr: "127.0.0.2", Port: 8091})

	// the ReplaceAll delete branch must notify subscribers —
	// without the notify loop the session manager would keep calling a dead
	// instance while the test itself still looked green.
	var notified []string
	var notifyMu sync.Mutex
	unsubscribe := service.store.Subscribe(func(cluster string, instances []*ServiceInstance) {
		notifyMu.Lock()
		notified = append(notified, cluster)
		notifyMu.Unlock()
	})
	defer unsubscribe()

	watchDone := make(chan struct{})
	go func() {
		service.watch(etcdClusterPrefix)
		close(watchDone)
	}()
	waitForSignal(t, watchStarted, "etcd watch to start")
	close(watchCh)
	waitFor(t, func() bool {
		instances, err := service.Lookup("default")
		return err == nil && len(instances) == 1 && instances[0].Addr == "127.0.0.2"
	}, "stale instance .1 to be removed by the rebuild snapshot")

	notifyMu.Lock()
	gotNotify := len(notified) > 0
	notifyMu.Unlock()
	assert.True(t, gotNotify, "ReplaceAll delete branch must notify subscribers")

	close(service.stopCh)
	select {
	case <-watchDone:
	case <-time.After(3 * time.Second):
		t.Fatal("watch did not stop")
	}
}

// TestWatch_MalformedDeleteEventIsSkipped verifies a Delete event whose key
// cannot be parsed is skipped without killing the watch or panicking
// (the Delete branch previously returned too).
func TestWatch_MalformedDeleteEventIsSkipped(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockEtcdClient := mock.NewMockEtcdClient(ctrl)
	service := &EtcdRegistryService{
		client:        newTestEtcdClient(mockEtcdClient),
		store:         NewAddressStore(),
		stopCh:        make(chan struct{}),
		vgroupMapping: map[string]string{"default": "default"},
	}

	mockEtcdClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(&clientv3.GetResponse{Header: &etcdserverpb.ResponseHeader{Revision: 41}}, nil).AnyTimes()
	watchStarted := make(chan struct{})
	var onceStart sync.Once
	watchCh := make(chan clientv3.WatchResponse)
	mockEtcdClient.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key string, options ...clientv3.OpOption) clientv3.WatchChan {
			onceStart.Do(func() { close(watchStarted) })
			return watchCh
		}).AnyTimes()
	mockEtcdClient.EXPECT().Close().Return(nil).AnyTimes()

	watchDone := make(chan struct{})
	go func() {
		service.watch(etcdClusterPrefix)
		close(watchDone)
	}()
	waitForSignal(t, watchStarted, "etcd watch to start")

	// Malformed Delete first (old code returned here): the watch must skip
	// it and keep the store untouched (no panic, no goroutine death).
	watchCh <- clientv3.WatchResponse{Events: []*etcd3.Event{
		{Type: etcd3.EventTypeDelete, Kv: &mvccpb.KeyValue{Key: []byte("foreign-delete-key")}},
	}}
	waitFor(t, func() bool {
		instances, err := service.Lookup("default")
		return err == nil && len(instances) == 0
	}, "store to remain empty after malformed Delete (goroutine alive)")

	// A valid Put after the malformed event must still be applied.
	watchCh <- clientv3.WatchResponse{Events: []*etcd3.Event{
		{Type: etcd3.EventTypePut, Kv: &mvccpb.KeyValue{
			Key:   []byte("registry-seata-default-127.0.0.9:8091"),
			Value: []byte("127.0.0.9:8091")},
		}},
	}
	waitFor(t, func() bool {
		instances, err := service.Lookup("default")
		return err == nil && len(instances) == 1 && instances[0].Addr == "127.0.0.9"
	}, "valid Put after malformed Delete to be applied")

	close(service.stopCh)
	select {
	case <-watchDone:
	case <-time.After(3 * time.Second):
		t.Fatal("watch did not stop")
	}
}

// TestGetClusterName_HyphenatedCluster verifies a cluster name containing
// "-" is parsed correctly (the old
// 4-segment check permanently ignored such clusters).
func TestGetClusterName_HyphenatedCluster(t *testing.T) {
	cluster, err := getClusterName([]byte("registry-seata-tc-prod-127.0.0.1:8091"))
	assert.NoError(t, err)
	assert.Equal(t, "tc-prod", cluster)

	cluster, err = getClusterName([]byte("registry-seata-us-east-1-127.0.0.1:8091"))
	assert.NoError(t, err)
	assert.Equal(t, "us-east-1", cluster)

	_, err = getClusterName([]byte("registry-seata-garbage"))
	assert.Error(t, err)

	// Delete-branch parse: same hyphenated-cluster rule must apply so
	// deregistration works for hyphenated cluster names (Put/Delete parity).
	cluster, ip, port, err := getClusterAndAddress([]byte("registry-seata-tc-prod-127.0.0.1:8091"))
	assert.NoError(t, err)
	assert.Equal(t, "tc-prod", cluster)
	assert.Equal(t, "127.0.0.1", ip)
	assert.Equal(t, 8091, port)
}
