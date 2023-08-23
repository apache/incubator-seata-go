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
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/seata/seata-go/pkg/util/log"
)

type NacosRegistryService struct {
	nc                naming_client.INamingClient
	nacosServerConfig NacosConfig
}

func NewNacosRegistryService(nacosConfig NacosConfig) RegistryService {
	properties := make(map[string]interface{})
	serverConfigs, _ := parseNacosServerConfig(nacosConfig)
	properties[constant.KEY_SERVER_CONFIGS] = serverConfigs
	client, err := clients.CreateNamingClient(properties)
	if err != nil {
		log.Fatalf("nacos client init error")
		panic("nacos client init error")

	}
	return &NacosRegistryService{client, nacosConfig}
}

func parseNacosServerConfig(config NacosConfig) ([]constant.ServerConfig, error) {
	serviceConfigs := make([]constant.ServerConfig, 0)
	serverAddr := config.ServerAddr
	addresses := strings.Split(serverAddr, ";")
	for _, addr := range addresses {
		ipPort := strings.Split(addr, ":")
		if len(ipPort) != 2 {
			return nil, fmt.Errorf("endpoint format should like ip:port. endpoint: %s", addr)
		}
		ip := ipPort[0]
		port, err := strconv.Atoi(ipPort[1])
		if err != nil {
			return nil, err
		}
		serviceConfigs = append(serviceConfigs, constant.ServerConfig{
			IpAddr: ip,
			Port:   uint64(port),
		})
	}
	return serviceConfigs, nil
}

func (s *NacosRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	instances, err := s.nc.SelectAllInstances(vo.SelectAllInstancesParam{
		GroupName:   s.nacosServerConfig.Group,
		ServiceName: s.nacosServerConfig.Application,
	})
	if err != nil {
		log.Fatalf("select all nacos service instance error")
		return nil, err
	}
	serviceInstances := make([]*ServiceInstance, 0)
	for _, instance := range instances {
		if !instance.Healthy {
			log.Warnf("%s status is un-healthy", instance.InstanceId)
			continue
		}
		serviceInstances = append(serviceInstances, &ServiceInstance{
			Addr: instance.Ip,
			Port: int(instance.Port),
		})
	}
	log.Infof("look up %d service instance from nacos server", len(serviceInstances))
	return serviceInstances, nil

}

func (NacosRegistryService) Close() {
	//TODO implement me
	panic("implement me")
}
