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
	"strings"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/seata/seata-go/pkg/util/flagext"
	"github.com/seata/seata-go/pkg/util/log"
)

var (
	onceGetNacosClient       = &sync.Once{}
	onceGetSeataServerClient = &sync.Once{}
)

type NacosRegistryService struct {
	registry *NacosRegistry
	cluster  string
	client   naming_client.INamingClient
	// seata server's instances
	serverInstances []*ServiceInstance
}

func newNacosRegistryService(registry *NacosRegistry, vgroupMapping flagext.StringMap, txServiceGroup string) RegistryService {
	if registry == nil {
		log.Fatalf("registry is nil for nacos")
		panic("registry is nil for nacos")
	}

	// get cluster
	cluster, ok := vgroupMapping[txServiceGroup]
	if !ok {
		panic("tx service group lacks vgroup mapping value")
	}

	s := &NacosRegistryService{
		registry: registry,
		cluster:  cluster,
	}

	// Listen for changes of seata server instances
	s.subscribe()
	return s
}

func (s *NacosRegistryService) getClient() naming_client.INamingClient {
	if s.client == nil {
		onceGetNacosClient.Do(func() {
			addr := strings.Split(s.registry.ServerAddr, ":")
			if len(addr) != 2 {
				panic("nacos server address should be in format ip:port!")
			}
			port, err := strconv.Atoi(addr[1])
			if err != nil {
				panic("nacos server address port is not valid!")
			}
			// create ServerConfig
			sc := []constant.ServerConfig{
				*constant.NewServerConfig(addr[0], uint64(port), constant.WithContextPath(s.registry.ContextPath)),
			}

			// create ClientConfig
			opts := []constant.ClientOption{
				constant.WithNamespaceId(s.registry.Namespace),
				constant.WithTimeoutMs(5000),
				constant.WithNotLoadCacheAtStart(true),
				constant.WithUsername(s.registry.Username),
				constant.WithPassword(s.registry.Password),
			}
			if s.registry.AccessKey != "" {
				opts = append(opts, constant.WithAccessKey(s.registry.AccessKey))
			}
			if s.registry.SecretKey != "" {
				opts = append(opts, constant.WithSecretKey(s.registry.SecretKey))
			}
			cc := constant.NewClientConfig(
				opts...,
			)

			// create naming client
			client, err := clients.NewNamingClient(
				vo.NacosClientParam{
					ServerConfigs: sc,
					ClientConfig:  cc,
				},
			)

			if err != nil {
				panic("error getting nacos naming client: " + err.Error())
			}
			s.client = client
		})
	}
	return s.client
}

func (s *NacosRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	// key is txServiceGroup, and it has been processed at init newNacosRegistryService().
	if s.serverInstances == nil {
		// In most cases, this will not be called since `subscribe()` should get server
		// instances already.
		onceGetSeataServerClient.Do(func() {
			param := vo.SelectInstancesParam{
				ServiceName: s.registry.Application,
				GroupName:   s.registry.Group,
				Clusters:    []string{s.cluster},
				HealthyOnly: true,
			}

			instances, err := s.getClient().SelectInstances(param)
			if err != nil {
				log.Error("error selecting seata server instances at nacos: %w", err)
				panic("error selecting seata server instances at nacos: " + err.Error())
			}
			res := make([]*ServiceInstance, 0)
			for _, instance := range instances {
				if instance.Enable {
					res = append(res, &ServiceInstance{
						Addr: instance.Ip,
						Port: int(instance.Port),
					})
				}
			}

			if len(res) > 0 {
				s.serverInstances = res
			}
		})
	}

	return s.serverInstances, nil
}

func (s *NacosRegistryService) Close() {
	if s.client != nil {
		s.client.CloseClient()
	}
}

// Listen for changes of seata server instances
func (s *NacosRegistryService) subscribe() {
	subscribeParam := &vo.SubscribeParam{
		ServiceName: s.registry.Application,
		GroupName:   s.registry.Group,
		Clusters:    []string{s.cluster},
		SubscribeCallback: func(instances []model.Instance, err error) {
			if err != nil {
				log.Error("error received at subscribing seata server instance change at nacos: %w", err)
				return
			}
			res := make([]*ServiceInstance, 0)
			for _, instance := range instances {
				if instance.Healthy && instance.Enable {
					res = append(res, &ServiceInstance{
						Addr: instance.Ip,
						Port: int(instance.Port),
					})
				}
			}

			if len(res) > 0 {
				s.serverInstances = res
			}
		},
	}
	if err := s.getClient().Subscribe(subscribeParam); err != nil {
		log.Error("error subscribing to seata server instance change at nacos: %w", err)
	}
}
