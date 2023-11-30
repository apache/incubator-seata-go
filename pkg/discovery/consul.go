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
	"sync"
	"time"

	"github.com/hashicorp/consul/api"

	"github.com/seata/seata-go/pkg/util/log"
)

type ConsulRegistryService struct {
	// the consul server client
	client *api.Client

	// serverMap the map of discovery server
	// key: server name value: server address
	serverMap *sync.Map

	// each time to update serverMap
	gapTime time.Duration

	// stopCh a chan to stop discovery
	stopCh chan struct{}
}

// newConsulRegistryService new a consul registry to discovery services
func newConsulRegistryService(config *ConsulConfig) RegistryService {
	if config == nil {
		log.Fatalf("consul service config is nil")
		panic("consul service config is nil")
	}

	cfg := api.DefaultConfig()
	cfg.Address = config.ServerAddr
	cli, err := api.NewClient(cfg)
	if err != nil {
		log.Fatalf("consul client init fail")
		panic(err)
	}

	consulService := &ConsulRegistryService{
		client:    cli,
		serverMap: new(sync.Map),
		gapTime:   3 * time.Second,
		stopCh:    make(chan struct{}),
	}

	go consulService.findAllServices()

	return consulService
}

func (s *ConsulRegistryService) Lookup(key string) (serviceIns []*ServiceInstance, err error) {
	serviceIns = make([]*ServiceInstance, 0)
	si, ok := s.serverMap.Load(key)
	if !ok {
		sis, errx := s.client.Agent().ServicesWithFilter(key)
		if errx != nil {
			panic(errx)
		}
		si = sis
	}

	serviceIns = append(serviceIns, &ServiceInstance{
		Addr: si.(*api.AgentService).Address,
		Port: si.(*api.AgentService).Port,
	})

	return
}

// findAllServices find all services in consul
func (s *ConsulRegistryService) findAllServices() {
	for {
		select {
		case <-time.After(s.gapTime):
			serviceList, err := s.client.Agent().Services()
			if err != nil {
				panic(err)
			}
			for key, v := range serviceList {
				s.serverMap.Store(key, v)
			}
		case <-s.stopCh:
			return
		default:
			return
		}
	}
}

func (s *ConsulRegistryService) Close() {
	s.stopCh <- struct{}{}
}
