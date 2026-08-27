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

const (
	FILE         string = "file"
	NACOS        string = "nacos"
	ETCD         string = "etcd"
	ETCD3        string = "etcd3"
	EUREKA       string = "eureka"
	REDIS        string = "redis"
	ZK           string = "zk"
	CONSUL       string = "consul"
	SOFA         string = "sofa"
	NAMINGSERVER string = "namingserver"
	RAFT         string = "raft"
)

type ServiceInstance struct {
	Addr string
	Port int
}

type RegistryService interface {
	Lookup(key string) ([]*ServiceInstance, error)
	Close()
}

type RegisterableRegistryService interface {
	Register(instance *ServiceInstance) error
	Unregister(instance *ServiceInstance) error
}

// RegistryChangeEvent is a backend-neutral snapshot of service instances for a
// transaction service group.
type RegistryChangeEvent struct {
	Key       string
	Instances []*ServiceInstance
}

// RegistryChangeListener receives backend-neutral registry change snapshots.
type RegistryChangeListener func(event RegistryChangeEvent)

// RegistrySubscription cancels a registry change subscription. Unsubscribe is
// safe to call multiple times. A callback already in progress may still run.
type RegistrySubscription interface {
	Unsubscribe()
}

// RegistrySubscriber is an optional capability for registries that can report
// address changes. Listeners receive the current snapshot first, followed by
// backend-neutral snapshots. Calls for one subscription are serialized. Slow
// listeners may skip intermediate snapshots.
type RegistrySubscriber interface {
	Subscribe(key string, listener RegistryChangeListener) (RegistrySubscription, error)
}
