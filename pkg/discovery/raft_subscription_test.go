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
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/discovery/metadata"
)

func TestRaftRegistryServiceSubscribePublishesMetadataChanges(t *testing.T) {
	service := newSubscriptionTestRaftRegistry()
	service.metadata.RefreshGroupMetadata("test-cluster", "", metadata.MetadataResponse{
		Term: 1,
		Nodes: []*metadata.Node{{
			Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 8001},
			Group:       "",
			Role:        metadata.LEADER,
		}},
	})

	events := make(chan RegistryChangeEvent, 2)
	subscription, err := service.Subscribe("default_tx_group", func(event RegistryChangeEvent) {
		events <- event
	})
	require.NoError(t, err)

	initial := nextRaftRegistryEvent(t, events)
	assert.Equal(t, "default_tx_group", initial.Key)
	assert.Equal(t, []*ServiceInstance{{Addr: "127.0.0.1", Port: 8001}}, initial.Instances)

	service.metadata.RefreshGroupMetadata("test-cluster", "", metadata.MetadataResponse{
		Term: 2,
		Nodes: []*metadata.Node{{
			Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 8002},
			Group:       "",
			Role:        metadata.LEADER,
		}},
	})
	service.stateMu.Lock()
	service.publishClusterSnapshotLocked("test-cluster")
	service.stateMu.Unlock()

	changed := nextRaftRegistryEvent(t, events)
	assert.Equal(t, []*ServiceInstance{{Addr: "127.0.0.1", Port: 8002}}, changed.Instances)

	subscription.Unsubscribe()
	subscription.Unsubscribe()
	service.Close()
	service.Close()
}

func TestRaftRegistryServiceCloseStopsSubscription(t *testing.T) {
	service := newSubscriptionTestRaftRegistry()
	service.metadata.RefreshGroupMetadata("test-cluster", "", metadata.MetadataResponse{
		Nodes: []*metadata.Node{{
			Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 8001},
			Group:       "",
			Role:        metadata.LEADER,
		}},
	})

	events := make(chan RegistryChangeEvent, 2)
	_, err := service.Subscribe("default_tx_group", func(event RegistryChangeEvent) {
		events <- event
	})
	require.NoError(t, err)
	_ = nextRaftRegistryEvent(t, events)

	service.Close()
	service.stateMu.Lock()
	service.publishClusterSnapshotLocked("test-cluster")
	service.stateMu.Unlock()

	assert.Equal(t, 0, len(events))
}

func TestRaftRegistryServiceSubscribeAfterCloseDoesNotLookup(t *testing.T) {
	service := newSubscriptionTestRaftRegistry()
	service.Close()

	subscription, err := service.Subscribe("default_tx_group", func(RegistryChangeEvent) {})
	assert.Nil(t, subscription)
	assert.EqualError(t, err, "registry service is closed")

	instances, err := service.Lookup("default_tx_group")
	assert.Nil(t, instances)
	assert.EqualError(t, err, "registry service is closed")
}

func newSubscriptionTestRaftRegistry() *RaftRegistryService {
	return &RaftRegistryService{
		cfg:           &RaftConfig{},
		metadata:      metadata.NewMetadata(),
		vgroupMapping: map[string]string{"default_tx_group": "test-cluster"},
		stopCh:        make(chan struct{}),
		httpClient:    &http.Client{},
		random:        rand.New(rand.NewSource(1)),
	}
}

func nextRaftRegistryEvent(t *testing.T, events <-chan RegistryChangeEvent) RegistryChangeEvent {
	t.Helper()

	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for registry change event")
	}
	return RegistryChangeEvent{}
}
