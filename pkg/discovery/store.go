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
	"strconv"
	"sync"
)

type AddressStoreSubscriber func(cluster string, instances []*ServiceInstance)

type AddressStore struct {
	mu          sync.RWMutex
	clusters    map[string][]*ServiceInstance
	subscribers map[uint64]AddressStoreSubscriber
	nextID      uint64
}

func NewAddressStore() *AddressStore {
	return &AddressStore{
		clusters:    make(map[string][]*ServiceInstance),
		subscribers: make(map[uint64]AddressStoreSubscriber),
	}
}

func (s *AddressStore) Snapshot(cluster string) []*ServiceInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneServiceInstances(s.clusters[cluster])
}

func (s *AddressStore) Update(cluster string, instances []*ServiceInstance) {
	s.saveAndNotify(cluster, instances)
}

func (s *AddressStore) Subscribe(subscriber AddressStoreSubscriber) func() {
	_, unsubscribe := s.subscribeWithSnapshot("", subscriber)
	return unsubscribe
}

func (s *AddressStore) subscribeWithSnapshot(cluster string, subscriber AddressStoreSubscriber) ([]*ServiceInstance, func()) {
	if subscriber == nil {
		return s.Snapshot(cluster), func() {}
	}

	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.subscribers[id] = subscriber
	snapshot := cloneServiceInstances(s.clusters[cluster])
	s.mu.Unlock()

	unsubscribe := func() {
		s.mu.Lock()
		delete(s.subscribers, id)
		s.mu.Unlock()
	}
	return snapshot, unsubscribe
}

func (s *AddressStore) upsert(cluster string, instance *ServiceInstance) {
	if instance == nil {
		return
	}

	s.mu.Lock()
	next := cloneServiceInstances(s.clusters[cluster])
	key := serviceInstanceKey(instance)
	for i := range next {
		if serviceInstanceKey(next[i]) == key {
			s.mu.Unlock()
			return
		}
	}
	next = append(next, cloneServiceInstance(instance))
	snapshot := s.saveLocked(cluster, next)
	subscribers := s.subscribersLocked()
	s.mu.Unlock()
	notifyAddressStoreSubscribers(subscribers, cluster, snapshot)
}

func (s *AddressStore) remove(cluster, addr string, port int) bool {
	s.mu.Lock()
	current := s.clusters[cluster]
	next := make([]*ServiceInstance, 0, len(current))
	removeKey := serviceInstanceKey(&ServiceInstance{Addr: addr, Port: port})
	removed := false
	for _, instance := range current {
		if serviceInstanceKey(instance) == removeKey {
			removed = true
			continue
		}
		next = append(next, cloneServiceInstance(instance))
	}
	if !removed {
		s.mu.Unlock()
		return false
	}
	snapshot := s.saveLocked(cluster, next)
	subscribers := s.subscribersLocked()
	s.mu.Unlock()
	notifyAddressStoreSubscribers(subscribers, cluster, snapshot)
	return true
}

func (s *AddressStore) saveAndNotify(cluster string, instances []*ServiceInstance) {
	s.mu.Lock()
	snapshot := s.saveLocked(cluster, instances)
	subscribers := s.subscribersLocked()
	s.mu.Unlock()
	notifyAddressStoreSubscribers(subscribers, cluster, snapshot)
}

func (s *AddressStore) saveLocked(cluster string, instances []*ServiceInstance) []*ServiceInstance {
	next := cloneServiceInstances(instances)
	if len(next) == 0 {
		delete(s.clusters, cluster)
	} else {
		s.clusters[cluster] = next
	}
	return next
}

func (s *AddressStore) subscribersLocked() []AddressStoreSubscriber {
	subscribers := make([]AddressStoreSubscriber, 0, len(s.subscribers))
	for _, subscriber := range s.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	return subscribers
}

func notifyAddressStoreSubscribers(subscribers []AddressStoreSubscriber, cluster string, instances []*ServiceInstance) {
	// Callbacks run without the store lock so subscribers can call Snapshot.
	for _, subscriber := range subscribers {
		subscriber(cluster, cloneServiceInstances(instances))
	}
}

func cloneServiceInstances(instances []*ServiceInstance) []*ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	result := make([]*ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		if instance == nil {
			continue
		}
		result = append(result, cloneServiceInstance(instance))
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func cloneServiceInstance(instance *ServiceInstance) *ServiceInstance {
	if instance == nil {
		return nil
	}
	clone := *instance
	return &clone
}

func serviceInstanceKey(instance *ServiceInstance) string {
	if instance == nil {
		return ""
	}
	return instance.Addr + ":" + strconv.Itoa(instance.Port)
}
