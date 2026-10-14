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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	getty "github.com/apache/dubbo-getty"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/discovery"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	"seata.apache.org/seata-go/v2/pkg/remoting/loadbalance"
	"seata.apache.org/seata-go/v2/pkg/remoting/mock"
)

func TestSessionManagerRefreshesServerListFromRegistrySubscription(t *testing.T) {
	registry := &fakeSubscriberRegistry{
		fakeLookupRegistry: fakeLookupRegistry{
			instances: []*discovery.ServiceInstance{{Addr: "127.0.0.1", Port: 8091}},
		},
	}
	started := make(chan string, 2)
	closed := make(chan string, 1)
	manager := newTestSessionManager(func(instance *discovery.ServiceInstance) closeableClient {
		address := serverAddress(instance)
		started <- address
		return &fakeServerClient{address: address, closed: closed}
	})

	manager.initWithRegistry(registry)
	assert.Equal(t, "default_tx_group", registry.subscribeKey)
	assert.Equal(t, "127.0.0.1:8091", nextStartedServer(t, started))

	registry.publish([]*discovery.ServiceInstance{{Addr: "127.0.0.2", Port: 8092}})
	assert.Equal(t, "127.0.0.1:8091", nextClosedServer(t, closed))
	assert.Equal(t, "127.0.0.2:8092", nextStartedServer(t, started))
	assert.False(t, manager.isServerAddressAvailable("127.0.0.1:8091"))
	assert.True(t, manager.isServerAddressAvailable("127.0.0.2:8092"))
}

func TestSessionManagerFallsBackToLookupWhenRegistryDoesNotSubscribe(t *testing.T) {
	registry := &fakeLookupRegistry{
		instances: []*discovery.ServiceInstance{{Addr: "127.0.0.1", Port: 8091}},
	}
	started := make(chan string, 1)
	manager := newTestSessionManager(func(instance *discovery.ServiceInstance) closeableClient {
		address := serverAddress(instance)
		started <- address
		return &fakeServerClient{address: address}
	})

	manager.initWithRegistry(registry)
	assert.Equal(t, "default_tx_group", registry.lookupKey)
	assert.Equal(t, "127.0.0.1:8091", nextStartedServer(t, started))
	assert.True(t, manager.isServerAddressAvailable("127.0.0.1:8091"))
}

func TestSessionManagerSelectSessionSkipsRemovedServerAddress(t *testing.T) {
	previousLoadBalance := loadbalance.GetLoadBalanceConfig()
	loadbalance.InitLoadBalanceConfig(loadbalance.Config{Type: "RoundRobinLoadBalance"})
	t.Cleanup(func() {
		loadbalance.InitLoadBalanceConfig(previousLoadBalance)
	})

	ctrl := gomock.NewController(t)
	manager := newTestSessionManager(nil)
	manager.refreshServerList([]*discovery.ServiceInstance{
		{Addr: "127.0.0.1", Port: 8091},
		{Addr: "127.0.0.2", Port: 8092},
	})

	removed := mock.NewMockTestSession(ctrl)
	removed.EXPECT().IsClosed().Return(false).AnyTimes()
	removed.EXPECT().RemoteAddr().Return("127.0.0.1:8091").AnyTimes()
	removed.EXPECT().Close().Times(1)
	kept := mock.NewMockTestSession(ctrl)
	kept.EXPECT().IsClosed().Return(false).AnyTimes()
	kept.EXPECT().RemoteAddr().Return("127.0.0.2:8092").AnyTimes()

	manager.registerSession(removed)
	manager.registerSession(kept)
	manager.refreshServerList([]*discovery.ServiceInstance{{Addr: "127.0.0.2", Port: 8092}})

	selected := manager.selectSession(message.GlobalBeginRequest{TransactionName: "tx"})
	assert.Equal(t, getty.Session(kept), selected)
	assert.Equal(t, int32(1), atomic.LoadInt32(&manager.sessionSize))
	if _, ok := manager.serverClients.Load("127.0.0.1:8091"); ok {
		t.Fatal("removed server client is still tracked")
	}
}

func TestSessionManagerClosesStaleClientWhenAddressRemovedDuringStart(t *testing.T) {
	started := make(chan string, 1)
	resumeStart := make(chan struct{})
	closed := make(chan string, 1)
	manager := newTestSessionManager(func(instance *discovery.ServiceInstance) closeableClient {
		address := serverAddress(instance)
		started <- address
		<-resumeStart
		return &fakeServerClient{address: address, closed: closed}
	})

	refreshDone := make(chan struct{})
	go func() {
		manager.refreshServerList([]*discovery.ServiceInstance{{Addr: "127.0.0.1", Port: 8091}})
		close(refreshDone)
	}()
	assert.Equal(t, "127.0.0.1:8091", nextStartedServer(t, started))

	manager.refreshServerList(nil)
	close(resumeStart)

	select {
	case <-refreshDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stale server client start to finish")
	}
	assert.Equal(t, "127.0.0.1:8091", nextClosedServer(t, closed))
	if _, ok := manager.serverClients.Load("127.0.0.1:8091"); ok {
		t.Fatal("stale server client is still tracked")
	}
}

func newTestSessionManager(startClient func(*discovery.ServiceInstance) closeableClient) *SessionManager {
	if startClient == nil {
		startClient = func(*discovery.ServiceInstance) closeableClient {
			return &fakeServerClient{}
		}
	}
	return &SessionManager{
		gettyConf:   &config.Config{},
		seataConfig: &config.SeataConfig{TxServiceGroup: "default_tx_group"},
		startClient: startClient,
	}
}

func nextStartedServer(t *testing.T, started <-chan string) string {
	t.Helper()

	select {
	case address := <-started:
		return address
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for server client start")
	}
	return ""
}

func nextClosedServer(t *testing.T, closed <-chan string) string {
	t.Helper()

	select {
	case address := <-closed:
		return address
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for server client close")
	}
	return ""
}

type fakeLookupRegistry struct {
	lookupKey string
	instances []*discovery.ServiceInstance
}

func (r *fakeLookupRegistry) Lookup(key string) ([]*discovery.ServiceInstance, error) {
	r.lookupKey = key
	return cloneTestServiceInstances(r.instances), nil
}

func (r *fakeLookupRegistry) Close() {}

type fakeSubscriberRegistry struct {
	fakeLookupRegistry
	subscribeKey string
	listener     discovery.RegistryChangeListener
}

func (r *fakeSubscriberRegistry) Subscribe(key string, listener discovery.RegistryChangeListener) (discovery.RegistrySubscription, error) {
	r.subscribeKey = key
	r.listener = listener
	listener(discovery.RegistryChangeEvent{Key: key, Instances: cloneTestServiceInstances(r.instances)})
	return fakeRegistrySubscription{}, nil
}

func (r *fakeSubscriberRegistry) publish(instances []*discovery.ServiceInstance) {
	r.listener(discovery.RegistryChangeEvent{Key: r.subscribeKey, Instances: cloneTestServiceInstances(instances)})
}

type fakeRegistrySubscription struct{}

func (fakeRegistrySubscription) Unsubscribe() {}

func cloneTestServiceInstances(instances []*discovery.ServiceInstance) []*discovery.ServiceInstance {
	clones := make([]*discovery.ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		clone := *instance
		clones = append(clones, &clone)
	}
	return clones
}

type fakeServerClient struct {
	address string
	closed  chan<- string
	once    sync.Once
}

func (c *fakeServerClient) Close() {
	c.once.Do(func() {
		if c.closed != nil {
			c.closed <- c.address
		}
	})
}

// TestGetXidUnwrapsRpcMessage covers the key ConsistentHashLoadBalance hashes on.
// SendSync / SendAsync hand over the whole message.RpcMessage, so getXid has to
// unwrap the body: reflecting over RpcMessage itself used to yield the literal
// "<invalid Value>" for every request, which pinned them all to one TC.
func TestGetXidUnwrapsRpcMessage(t *testing.T) {
	manager := &SessionManager{}

	assert.Equal(t, "tx-1", manager.getXid(message.RpcMessage{
		ID:   1,
		Body: message.GlobalBeginRequest{TransactionName: "tx-1"},
	}))
	assert.Equal(t, "tx-2", manager.getXid(message.RpcMessage{
		ID:   2,
		Body: message.BranchRegisterRequest{Xid: "tx-2"},
	}))
	// Passing the body directly keeps working, which is what selectSession did
	// in the existing tests.
	assert.Equal(t, "tx-3", manager.getXid(message.GlobalBeginRequest{TransactionName: "tx-3"}))

	// No transaction key at all: an empty key, never the placeholder string.
	assert.Equal(t, "", manager.getXid(message.RpcMessage{ID: 4}))
	assert.Equal(t, "", manager.getXid(nil))
	assert.Equal(t, "", manager.getXid(message.HeartBeatMessage{Ping: true}))
}
