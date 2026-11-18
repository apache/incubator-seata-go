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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	etcd3 "go.etcd.io/etcd/client/v3"

	"seata.apache.org/seata-go/v2/pkg/util/log"
	"seata.apache.org/seata-go/v2/pkg/util/net"
)

const (
	clusterNameSplitChar = "-"
	addressSplitChar     = ":"
	etcdClusterPrefix    = "registry-seata"
)

type EtcdRegistryService struct {
	client        *etcd3.Client
	cfg           etcd3.Config
	vgroupMapping map[string]string // copied during construction; read-only afterwards
	store         *AddressStore

	stopCh    chan struct{}
	closeOnce sync.Once

	subscriptionsMu sync.Mutex
	subscriptions   map[*registryChangeSubscription]struct{}
	closed          bool
}

var _ RegistrySubscriber = (*EtcdRegistryService)(nil)

func newEtcdRegistryService(config *ServiceConfig, etcd3Config *Etcd3Config) (RegistryService, error) {
	if config == nil {
		return nil, fmt.Errorf("service config is nil")
	}
	if etcd3Config == nil {
		return nil, fmt.Errorf("etcd config is nil")
	}

	cfg := etcd3.Config{
		Endpoints: []string{etcd3Config.ServerAddr},
	}
	cli, err := etcd3.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd3 client: %w", err)
	}

	vgroupMapping := make(map[string]string, len(config.VgroupMapping))
	for key, cluster := range config.VgroupMapping {
		vgroupMapping[key] = cluster
	}

	etcdRegistryService := &EtcdRegistryService{
		client:        cli,
		cfg:           cfg,
		vgroupMapping: vgroupMapping,
		store:         NewAddressStore(),
		stopCh:        make(chan struct{}),
	}
	go etcdRegistryService.watch(etcdClusterPrefix)

	return etcdRegistryService, nil
}

func (s *EtcdRegistryService) watch(key string) {
	// The watch goroutine is the only writer of the address store: exiting
	// it froze service discovery until process restart. The loop is
	// therefore self-healing — malformed events are skipped, and a closed
	// watch channel is re-established with backoff. Only stopCh ends it.
	for {
		if !s.watchOnce(key) {
			return
		}
		select {
		case <-s.stopCh:
			log.Warn("stop etcd watch")
			return
		case <-time.After(time.Second):
			// brief backoff before re-establishing the watch
		}
	}
}

// watchOnce establishes one watch and processes events until the channel
// closes or the registry stops. It returns false when the goroutine must
// exit (stopCh closed).
func (s *EtcdRegistryService) watchOnce(key string) bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := s.client.Get(ctx, key, etcd3.WithPrefix())
	if err != nil {
		log.Infof("cant get server instances from etcd")
	}

	if resp != nil {
		// The prefix Get is a full snapshot: collect it and reconcile the
		// whole store against it, so instances deleted while a watch was
		// being rebuilt do not linger forever. Malformed
		// or foreign entries are skipped — a cluster whose only entry is
		// malformed is treated as having no instances.
		snapshot := make(map[string][]*ServiceInstance)
		for _, kv := range resp.Kvs {
			k := kv.Key
			v := kv.Value
			clusterName, err := getClusterName(k)
			if err != nil {
				log.Warnf("etcd key has an incorrect format, skipping: %v", err)
				continue
			}
			serverInstance, err := getServerInstance(v)
			if err != nil {
				log.Warnf("etcd value has an incorrect format, skipping: %v", err)
				continue
			}
			snapshot[clusterName] = append(snapshot[clusterName], serverInstance)
		}
		s.store.ReplaceAll(snapshot)
	}
	watchOptions := []etcd3.OpOption{etcd3.WithPrefix()}
	if resp != nil && resp.Header != nil {
		watchOptions = append(watchOptions, etcd3.WithRev(resp.Header.Revision+1))
	}
	watchCh := s.client.Watch(ctx, key, watchOptions...)

	for {
		select {
		case watchResp, ok := <-watchCh:
			if !ok {
				log.Warnf("Watch channel closed, re-establishing")
				return true
			}
			for _, event := range watchResp.Events {
				switch event.Type {
				case etcd3.EventTypePut:
					log.Infof("Key %s updated. New value: %s\n", event.Kv.Key, event.Kv.Value)

					k := event.Kv.Key
					v := event.Kv.Value
					// Foreign or malformed entries are skipped: returning here
					// permanently killed the watch goroutine and froze the
					// address list until process restart.
					clusterName, err := getClusterName(k)
					if err != nil {
						log.Warnf("etcd key err: %v, event skipped", err)
						continue
					}
					serverInstance, err := getServerInstance(v)
					if err != nil {
						log.Warnf("etcd value err: %v, event skipped", err)
						continue
					}

					s.store.upsert(clusterName, serverInstance)

				case etcd3.EventTypeDelete:
					log.Infof("Key %s deleted.\n", event.Kv.Key)

					cluster, ip, port, err := getClusterAndAddress(event.Kv.Key)
					if err != nil {
						log.Warnf("etcd key err: %v, event skipped", err)
						continue
					}

					if !s.store.remove(cluster, ip, port) {
						log.Warnf("etcd instance not found. cluster: %s addr: %s:%d", cluster, ip, port)
					}
				}
			}
		case <-s.stopCh:
			log.Warn("stop etcd watch")
			return false
		}
	}
}

func getClusterName(key []byte) (string, error) {
	stringKey := string(key)
	keySplit := strings.Split(stringKey, clusterNameSplitChar)
	// The key is registry-seata-{cluster}-{ip}:{port}. The cluster name may
	// itself contain "-", so everything between the fixed prefix segments and
	// the trailing ip:port segment is the cluster (previously a hyphenated
	// cluster name made len!=4 and the key was permanently ignored — the
	// initial snapshot abort scenario for hyphenated cluster names).
	// The trailing segment must be a parseable ip:port: a key without one
	// could be upserted but never deregistered (Put/Delete asymmetry, P3).
	tail := keySplit[len(keySplit)-1]
	if _, _, err := net.SplitIPPortStr(tail); err != nil {
		return "", fmt.Errorf("etcd key has an incorrect tail segment. key: %s", stringKey)
	}
	if len(keySplit) < 4 {
		return "", fmt.Errorf("etcd key has an incorrect format. key: %s", stringKey)
	}

	cluster := strings.Join(keySplit[2:len(keySplit)-1], clusterNameSplitChar)
	return cluster, nil
}

func getServerInstance(value []byte) (*ServiceInstance, error) {
	stringValue := string(value)
	ip, port, err := net.SplitIPPortStr(stringValue)
	if err != nil {
		return nil, fmt.Errorf("etcd port has an incorrect format. err: %w", err)
	}
	serverInstance := &ServiceInstance{
		Addr: ip,
		Port: port,
	}

	return serverInstance, nil
}

func getClusterAndAddress(key []byte) (string, string, int, error) {
	stringKey := string(key)
	keySplit := strings.Split(stringKey, clusterNameSplitChar)
	// Same hyphenated-cluster rule as getClusterName: everything between the
	// fixed prefix segments and the trailing ip:port is the cluster name, so
	// deregistration works for cluster names containing "-" too.
	if len(keySplit) < 4 {
		return "", "", 0, fmt.Errorf("etcd key has an incorrect format. key: %s", stringKey)
	}
	cluster := strings.Join(keySplit[2:len(keySplit)-1], clusterNameSplitChar)
	ip, port, err := net.SplitIPPortStr(keySplit[len(keySplit)-1])
	if err != nil {
		return "", "", 0, fmt.Errorf("etcd port has an incorrect format. err: %w", err)
	}

	return cluster, ip, port, nil
}

func (s *EtcdRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	cluster := s.vgroupMapping[key]
	if cluster == "" {
		return nil, fmt.Errorf("cluster doesn't exist")
	}

	return s.store.Snapshot(cluster), nil
}

func (s *EtcdRegistryService) Subscribe(key string, listener RegistryChangeListener) (RegistrySubscription, error) {
	if listener == nil {
		return nil, fmt.Errorf("registry change listener is nil")
	}

	cluster := s.vgroupMapping[key]
	if cluster == "" {
		return nil, fmt.Errorf("cluster doesn't exist")
	}

	subscription := newRegistryChangeSubscription(listener)
	// Register before checking closed so taking the initial snapshot cannot miss
	// an update. If Close wins the race, the check below removes the callback.
	initial, unsubscribe := s.store.subscribeWithSnapshot(cluster, func(changedCluster string, instances []*ServiceInstance) {
		if changedCluster == cluster {
			subscription.publish(RegistryChangeEvent{Key: key, Instances: instances})
		}
	})
	subscription.initialize(RegistryChangeEvent{Key: key, Instances: initial}, func() {
		unsubscribe()
		s.removeSubscription(subscription)
	})

	s.subscriptionsMu.Lock()
	if s.closed {
		s.subscriptionsMu.Unlock()
		subscription.Unsubscribe()
		return nil, fmt.Errorf("registry service is closed")
	}
	if s.subscriptions == nil {
		s.subscriptions = make(map[*registryChangeSubscription]struct{})
	}
	s.subscriptions[subscription] = struct{}{}
	s.subscriptionsMu.Unlock()

	subscription.start()
	return subscription, nil
}

func (s *EtcdRegistryService) removeSubscription(subscription *registryChangeSubscription) {
	s.subscriptionsMu.Lock()
	delete(s.subscriptions, subscription)
	s.subscriptionsMu.Unlock()
}

func (s *EtcdRegistryService) Close() {
	s.closeOnce.Do(func() {
		s.subscriptionsMu.Lock()
		s.closed = true
		subscriptions := make([]*registryChangeSubscription, 0, len(s.subscriptions))
		for subscription := range s.subscriptions {
			subscriptions = append(subscriptions, subscription)
		}
		s.subscriptions = nil
		s.subscriptionsMu.Unlock()

		for _, subscription := range subscriptions {
			subscription.Unsubscribe()
		}
		if s.stopCh != nil {
			close(s.stopCh)
		}
		if s.client != nil {
			if err := s.client.Close(); err != nil && !errors.Is(err, context.Canceled) {
				log.Warnf("close etcd client failed: %v", err)
			}
		}
	})
}
