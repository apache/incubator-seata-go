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

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"

	"github.com/seata/seata-go/pkg/util/log"
)

type ConsulRegistryService struct {
	// the config about consul
	config *ConsulConfig

	// the consul server client
	client *api.Client

	// serverMap the map of discovery server
	// key: server name value: server address
	serverMap *sync.Map

	// stopCh a chan to stop discovery
	stopCh chan struct{}

	watchers map[string]*watch.Plan // store plans

	RWMutex *sync.RWMutex

	// watch plan type
	watchType string
}

// newConsulRegistryService new a consul registry to discovery services
func newConsulRegistryService(config *ConsulConfig, opt ...map[string]interface{}) RegistryService {
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
		stopCh:    make(chan struct{}),
		config:    config,
		watchType: "service",
	}

	consulService.findServiceAddress()
	go func() {
		_, err = consulService.NewWatchPlan(opt...)
		if err != nil {
			return
		}
	}()

	return consulService
}

func (s *ConsulRegistryService) Lookup(key string) (serviceIns []*ServiceInstance, err error) {
	insList, ok := s.serverMap.Load(key)
	if !ok {
		// try again
		var r []*ServiceInstance
		var svcMap = make(map[string]*api.AgentService)
		svcMap, err = s.client.Agent().ServicesWithFilter(fmt.Sprintf("Service == \"%s\"", key))
		if err != nil {
			return
		}

		for _, v := range svcMap {
			r = append(r, &ServiceInstance{
				Addr: v.Address,
				Port: v.Port,
			})
		}

		serviceIns = r
		return
	}

	serviceIns, ok = insList.([]*ServiceInstance)
	if !ok {
		return
	}
	return
}

// findServiceAddress find all service address which register in consul
func (s *ConsulRegistryService) findServiceAddress() {
	svcMap, err := s.client.Agent().Services()
	if err != nil {
		return
	}

	for k, v := range svcMap {
		serviceIns, ok := s.serverMap.Load(k)
		if !ok {
			// first time to store, inits service instance
			insList := make([]*ServiceInstance, 0, 1)
			insList = append(insList, &ServiceInstance{
				Addr: v.Address,
				Port: v.Port,
			})
			s.serverMap.Store(k, insList)
			continue
		}

		insList := serviceIns.([]*ServiceInstance)
		insList = append(insList, &ServiceInstance{
			Addr: v.Address,
			Port: v.Port,
		})
		s.serverMap.Store(k, insList)
	}

	return
}

func (s *ConsulRegistryService) Close() {
	s.stopCh <- struct{}{}
}

// RegisterService 将gRPC服务注册到consul
func RegisterService(serviceName string, ip string, port int) error {
	cfg := api.DefaultConfig()
	cfg.Address = "localhost:8500"
	c, _ := api.NewClient(cfg)
	srv := &api.AgentServiceRegistration{
		Name:    serviceName,                     // service name
		Tags:    []string{"fanone", "tags_test"}, // service tags
		Address: ip,
		Port:    port,
	}
	return c.Agent().ServiceRegister(srv)
}

// NewWatchPlan new watch plan
func (s *ConsulRegistryService) NewWatchPlan(opts ...map[string]interface{}) (*watch.Plan, error) {
	var options = map[string]interface{}{
		"type": s.watchType,
	}

	// combine params
	for _, opt := range opts {
		for k, v := range opt {
			options[k] = v
		}
	}

	pl, err := watch.Parse(options)
	if err != nil {
		return nil, err
	}
	pl.Handler = s.watchAll
	return pl, nil
}

// watchAll used to watch whole consul services changes
func (s *ConsulRegistryService) watchAll(_ uint64, data interface{}) {
	switch d := data.(type) {
	// "services" watch type returns map[string][]string type. follow:https://www.consul.io/docs/dynamic-app-config/watches#services
	case map[string][]string:
		for k := range d {
			if _, ok := s.watchers[k]; ok || k == "consul" {
				continue
			}
			s.HealthyWatch(k)
		}

		// read watchers and delete deregister services
		s.RWMutex.RLock()
		defer s.RWMutex.RUnlock()
		watchers := s.watchers
		for k, plan := range watchers {
			if _, ok := d[k]; !ok {
				plan.Stop()
				delete(watchers, k)
			}
		}
	default:
		fmt.Printf("can't decide the watch type: %v\n", &d)
	}
}

func (s *ConsulRegistryService) HealthyWatch(serviceName string) {
	options := map[string]interface{}{
		"type":    "service",
		"service": serviceName,
	}
	pl, err := watch.Parse(options)
	if err != nil {
		return
	}

	pl.Handler = func(u uint64, raw interface{}) {
		switch d := raw.(type) {
		case []*api.ServiceEntry:
			pairs := make([]*ServiceInstance, 0, len(d))
			for _, entry := range d {
				// filter some
				if entry.Checks.AggregatedStatus() == api.HealthPassing {
					pairs = append(pairs, &ServiceInstance{
						Addr: entry.Service.Address,
						Port: entry.Service.Port,
					})
				}
			}
			s.serverMap.Store(serviceName, pairs)
		}
	}

	go func() {
		_ = runWatchPlan(pl, s.config.ServerAddr)
	}()
	defer s.RWMutex.Unlock()
	s.RWMutex.Lock()
	s.watchers[serviceName] = pl
}

func runWatchPlan(plan *watch.Plan, address string) error {
	defer plan.Stop()
	err := plan.Run(address)
	if err != nil {
		fmt.Println("run consul error: ", err)
		return err
	}
	return nil
}

// func (s *ConsulRegistryService) watch(key string) {
//	var params = map[string]interface{}{
//		"type":    "service",
//		"service": key,
//	}
//	plan, err := watch.Parse(params)
//	if err != nil {
//		return
//	}
//
//	plan.Handler = func(u uint64, raw interface{}) {
//		entries := raw.([]*api.ServiceEntry)
//		pairs := make([]*ServiceInstance, 0, len(entries))
//		for _, entry := range entries {
//			// filter some
//			if entry.Checks.AggregatedStatus() == api.HealthPassing {
//				pairs = append(pairs, &ServiceInstance{
//					Addr: entry.Service.Address,
//					Port: entry.Service.Port,
//				})
//			}
//		}
//		s.serverMap.Store(key, pairs)
//	}
//
//	if err = plan.Run(s.config.ServerAddr); err != nil {
//		return
//	}
// }
