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
	"fmt"
	"sync"
	"testing"
)

func TestAddressStoreUpdateAndSnapshotClone(t *testing.T) {
	store := NewAddressStore()
	input := []*ServiceInstance{
		{Addr: "127.0.0.1", Port: 8091},
		{Addr: "127.0.0.2", Port: 8092},
	}

	store.Update("default", input)
	input[0].Addr = "10.0.0.1"

	snapshot := store.Snapshot("default")
	if len(snapshot) != 2 {
		t.Fatalf("snapshot length = %d, want 2", len(snapshot))
	}
	if snapshot[0].Addr != "127.0.0.1" {
		t.Fatalf("snapshot was changed by caller input: %v", snapshot[0])
	}

	snapshot[0].Port = 9999
	next := store.Snapshot("default")
	if next[0].Port != 8091 {
		t.Fatalf("store was changed by snapshot mutation: %v", next[0])
	}
}

func TestAddressStoreSubscribeAndUnsubscribe(t *testing.T) {
	store := NewAddressStore()
	received := make(chan []*ServiceInstance, 1)

	unsubscribe := store.Subscribe(func(cluster string, instances []*ServiceInstance) {
		if cluster != "default" {
			t.Fatalf("cluster = %s, want default", cluster)
		}
		instances[0].Addr = "mutated"
		received <- instances
	})

	store.Update("default", []*ServiceInstance{{Addr: "127.0.0.1", Port: 8091}})
	got := <-received
	if got[0].Addr != "mutated" {
		t.Fatalf("subscriber did not receive a mutable copy: %v", got[0])
	}
	if snapshot := store.Snapshot("default"); snapshot[0].Addr != "127.0.0.1" {
		t.Fatalf("subscriber mutated store snapshot: %v", snapshot[0])
	}

	unsubscribe()
	store.Update("default", []*ServiceInstance{{Addr: "127.0.0.2", Port: 8092}})
	select {
	case got := <-received:
		t.Fatalf("received update after unsubscribe: %v", got)
	default:
	}
}

func TestAddressStoreUpsertRemove(t *testing.T) {
	store := NewAddressStore()
	updates := 0
	store.Subscribe(func(string, []*ServiceInstance) {
		updates++
	})

	store.upsert("default", &ServiceInstance{Addr: "127.0.0.1", Port: 8091})
	store.upsert("default", &ServiceInstance{Addr: "127.0.0.1", Port: 8091})
	store.upsert("default", &ServiceInstance{Addr: "127.0.0.2", Port: 8092})

	snapshot := store.Snapshot("default")
	if len(snapshot) != 2 {
		t.Fatalf("snapshot length = %d, want 2", len(snapshot))
	}
	if updates != 2 {
		t.Fatalf("updates = %d, want 2", updates)
	}

	if !store.remove("default", "127.0.0.1", 8091) {
		t.Fatal("expected remove to delete an instance")
	}
	snapshot = store.Snapshot("default")
	if len(snapshot) != 1 {
		t.Fatalf("snapshot length after remove = %d, want 1", len(snapshot))
	}
	if snapshot[0].Addr != "127.0.0.2" || snapshot[0].Port != 8092 {
		t.Fatalf("unexpected instance after remove: %v", snapshot[0])
	}

	if store.remove("default", "127.0.0.3", 8093) {
		t.Fatal("remove should report false for a missing instance")
	}
	if updates != 3 {
		t.Fatalf("updates after missing remove = %d, want 3", updates)
	}
}

func TestAddressStoreConcurrentUpdateSnapshot(t *testing.T) {
	store := NewAddressStore()
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				store.Update("default", []*ServiceInstance{{
					Addr: fmt.Sprintf("127.0.0.%d", worker),
					Port: 8091 + j,
				}})
				for _, instance := range store.Snapshot("default") {
					if instance == nil {
						t.Error("snapshot contains nil instance")
					}
				}
			}
		}(i)
	}

	wg.Wait()
}
