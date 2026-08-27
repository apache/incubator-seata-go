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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEtcd3RegistryService_Subscribe(t *testing.T) {
	store := NewAddressStore()
	store.Update("default", []*ServiceInstance{{Addr: "127.0.0.1", Port: 8091}})
	service := newTestEtcdRegistryService(store)

	events := make(chan RegistryChangeEvent, 4)
	subscription, err := service.Subscribe("default_tx_group", func(event RegistryChangeEvent) {
		events <- event
	})
	require.NoError(t, err)
	defer subscription.Unsubscribe()

	assert.Equal(t, RegistryChangeEvent{
		Key:       "default_tx_group",
		Instances: []*ServiceInstance{{Addr: "127.0.0.1", Port: 8091}},
	}, nextRegistryChangeEvent(t, events))

	store.Update("other", []*ServiceInstance{{Addr: "127.0.0.2", Port: 8092}})
	store.Update("default", []*ServiceInstance{
		{Addr: "127.0.0.1", Port: 8091},
		{Addr: "127.0.0.3", Port: 8093},
	})
	assert.Equal(t, []*ServiceInstance{
		{Addr: "127.0.0.1", Port: 8091},
		{Addr: "127.0.0.3", Port: 8093},
	}, nextRegistryChangeEvent(t, events).Instances)

	subscription.Unsubscribe()
	subscription.Unsubscribe()
	store.mu.RLock()
	assert.Empty(t, store.subscribers)
	store.mu.RUnlock()
}

func TestEtcd3RegistryService_SubscribeValidatesInputs(t *testing.T) {
	service := newTestEtcdRegistryService(NewAddressStore())

	subscription, err := service.Subscribe("default_tx_group", nil)
	assert.Nil(t, subscription)
	assert.EqualError(t, err, "registry change listener is nil")

	subscription, err = service.Subscribe("missing_tx_group", func(RegistryChangeEvent) {})
	assert.Nil(t, subscription)
	assert.EqualError(t, err, "cluster doesn't exist")
}

func TestEtcd3RegistryService_SubscribeCoalescesSlowListener(t *testing.T) {
	store := NewAddressStore()
	service := newTestEtcdRegistryService(store)

	block := make(chan struct{})
	events := make(chan RegistryChangeEvent, 4)
	subscription, err := service.Subscribe("default_tx_group", func(event RegistryChangeEvent) {
		events <- event
		<-block
	})
	require.NoError(t, err)
	defer subscription.Unsubscribe()

	nextRegistryChangeEvent(t, events)
	store.Update("default", []*ServiceInstance{{Addr: "127.0.0.1", Port: 8091}})
	store.Update("default", []*ServiceInstance{{Addr: "127.0.0.2", Port: 8092}})
	store.Update("default", []*ServiceInstance{{Addr: "127.0.0.3", Port: 8093}})
	close(block)

	assert.Equal(t, []*ServiceInstance{{Addr: "127.0.0.3", Port: 8093}}, nextRegistryChangeEvent(t, events).Instances)
}

func TestEtcd3RegistryService_CloseUnsubscribesListeners(t *testing.T) {
	store := NewAddressStore()
	service := newTestEtcdRegistryService(store)

	events := make(chan RegistryChangeEvent, 1)
	subscription, err := service.Subscribe("default_tx_group", func(event RegistryChangeEvent) {
		events <- event
	})
	require.NoError(t, err)
	nextRegistryChangeEvent(t, events)

	service.Close()
	concreteSubscription := subscription.(*registryChangeSubscription)
	waitForSignal(t, concreteSubscription.doneCh, "registry subscription to close")
	store.mu.RLock()
	assert.Empty(t, store.subscribers)
	store.mu.RUnlock()

	subscription, err = service.Subscribe("default_tx_group", func(RegistryChangeEvent) {})
	assert.Nil(t, subscription)
	assert.EqualError(t, err, "registry service is closed")
}

func newTestEtcdRegistryService(store *AddressStore) *EtcdRegistryService {
	return &EtcdRegistryService{
		vgroupMapping: map[string]string{"default_tx_group": "default"},
		store:         store,
	}
}

func nextRegistryChangeEvent(t *testing.T, events <-chan RegistryChangeEvent) RegistryChangeEvent {
	t.Helper()

	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for registry change event")
	}
	return RegistryChangeEvent{}
}
