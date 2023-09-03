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
	"fmt"
	"github.com/seata/seata-go/pkg/util/log"
	etcd3 "go.etcd.io/etcd/client/v3"
	"strconv"
	"strings"
	"sync"
)

const (
	clusterNameSplitChar = "-"
	addressSplitChar     = ":"
	etcdClusterPrefix    = "registry-seata"
)

type EtcdRegistryService struct {
	client        *etcd3.Client
	cfg           etcd3.Config
	vgroupMapping map[string]string
	grouplist     map[string][]*ServiceInstance
	rwLock        sync.RWMutex

	stopCh chan struct{}
}

func newEtcdRegistryService(config *ServiceConfig, etcd3Config *Etcd3Config) RegistryService {

	if etcd3Config == nil {
		log.Fatalf("etcd config is nil")
		panic("etcd config is nil")
	}

	cfg := etcd3.Config{
		Endpoints: []string{etcd3Config.ServerAddr},
	}
	cli, err := etcd3.New(cfg)
	if err != nil {
		log.Fatalf("failed to create etcd3 client")
		panic("failed to create etcd3 client")
	}

	vgroupMapping := config.VgroupMapping
	grouplist := make(map[string][]*ServiceInstance, 0)

	etcdRegistryService := &EtcdRegistryService{
		client:        cli,
		cfg:           cfg,
		vgroupMapping: vgroupMapping,
		grouplist:     grouplist,
		stopCh:        make(chan struct{}),
	}
	go etcdRegistryService.watch(etcdClusterPrefix)

	return etcdRegistryService
}

func (s *EtcdRegistryService) watch(key string) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := s.client.Get(ctx, key, etcd3.WithPrefix())
	if err != nil {
		log.Infof("cant get server instances from etcd")
	}

	for _, kv := range resp.Kvs {
		k := kv.Key
		v := kv.Value
		clusterName, err := getClusterName(k)
		if err != nil {
			log.Errorf("etcd key has an incorrect format: ", err)
			return
		}
		serverInstance, err := getServerInstance(v)
		if err != nil {
			log.Errorf("etcd value has an incorrect format: ", err)
			return
		}
		s.rwLock.Lock()
		if s.grouplist[clusterName] == nil {
			s.grouplist[clusterName] = []*ServiceInstance{serverInstance}
		} else {
			s.grouplist[clusterName] = append(s.grouplist[clusterName], serverInstance)
		}
		s.rwLock.Unlock()
	}

	// watch the changes of endpoints
	watchCh := s.client.Watch(ctx, key, etcd3.WithPrefix())

	for {
		select {
		case watchResp, ok := <-watchCh:
			if !ok {
				log.Warnf("Watch channel closed")
				return
			}
			for _, event := range watchResp.Events {
				switch event.Type {
				case etcd3.EventTypePut:
					log.Infof("Key %s updated. New value: %s\n", event.Kv.Key, event.Kv.Value)

					k := event.Kv.Key
					v := event.Kv.Value
					clusterName, err := getClusterName(k)
					if err != nil {
						log.Errorf("etcd key err: ", err)
						return
					}
					serverInstance, err := getServerInstance(v)
					if err != nil {
						log.Errorf("etcd value err: ", err)
						return
					}

					s.rwLock.Lock()
					if s.grouplist[clusterName] == nil {
						s.grouplist[clusterName] = []*ServiceInstance{serverInstance}
						s.rwLock.Unlock()
						continue
					}
					if ifHaveSameServiceInstances(s.grouplist[clusterName], serverInstance) {
						s.rwLock.Unlock()
						continue
					}
					s.grouplist[clusterName] = append(s.grouplist[clusterName], serverInstance)
					s.rwLock.Unlock()

				case etcd3.EventTypeDelete:
					log.Infof("Key %s deleted.\n", event.Kv.Key)

					cluster, ip, port, err := getClusterAndAddress(event.Kv.Key)
					if err != nil {
						log.Errorf("etcd key err: ", err)
						return
					}

					s.rwLock.Lock()
					serviceInstances := s.grouplist[cluster]
					if serviceInstances == nil {
						log.Warnf("etcd doesnt exit cluster: ", cluster)
						s.rwLock.Unlock()
						continue
					}
					s.grouplist[cluster] = removeValueFromList(serviceInstances, ip, port)
					s.rwLock.Unlock()
				}
			}
		case <-s.stopCh:
			log.Warn("stop etcd watch")
			return
		}
	}
}

func getClusterName(key []byte) (string, error) {
	stringKey := string(key)
	keySplit := strings.Split(stringKey, clusterNameSplitChar)
	if len(keySplit) != 4 {
		return "", fmt.Errorf("etcd key has an incorrect format. key: %s", stringKey)
	}

	cluster := keySplit[2]
	return cluster, nil
}

func getServerInstance(value []byte) (*ServiceInstance, error) {
	stringValue := string(value)
	valueSplit := strings.Split(stringValue, addressSplitChar)
	if len(valueSplit) != 2 {
		return nil, fmt.Errorf("etcd value has an incorrect format. value: %s", stringValue)
	}
	ip := valueSplit[0]
	port, err := strconv.Atoi(valueSplit[1])
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
	if len(keySplit) != 4 {
		return "", "", 0, fmt.Errorf("etcd key has an incorrect format. key: %s", stringKey)
	}
	cluster := keySplit[2]
	address := strings.Split(keySplit[3], addressSplitChar)
	ip := address[0]
	port, err := strconv.Atoi(address[1])
	if err != nil {
		return "", "", 0, fmt.Errorf("etcd port has an incorrect format. err: %w", err)
	}

	return cluster, ip, port, nil
}

func ifHaveSameServiceInstances(list []*ServiceInstance, value *ServiceInstance) bool {
	for _, v := range list {
		if v.Addr == value.Addr && v.Port == value.Port {
			return true
		}
	}
	return false
}

func removeValueFromList(list []*ServiceInstance, ip string, port int) []*ServiceInstance {
	for k, v := range list {
		if v.Addr == ip && v.Port == port {
			result := list[:k]
			if k < len(list)-1 {
				result = append(result, list[k+1:]...)
			}
			return result
		}
	}

	return list
}

func (s *EtcdRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	s.rwLock.RLock()
	defer s.rwLock.RUnlock()
	cluster := s.vgroupMapping[key]
	if cluster == "" {
		return nil, fmt.Errorf("cluster doesnt exit")
	}

	list := s.grouplist[cluster]
	return list, nil
}

func (s *EtcdRegistryService) Close() {
	s.stopCh <- struct{}{}
}
