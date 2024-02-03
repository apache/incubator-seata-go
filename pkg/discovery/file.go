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

	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	endPointSplitChar = ";"
	ipPortSplitChar   = ":"
)

type FileRegistryService struct {
	serviceConfig *ServiceConfig
}

func newFileRegistryService(config *ServiceConfig) RegistryService {
	if config == nil {
		log.Fatalf("service config is nil")
		panic("service config is nil")
	}
	return &FileRegistryService{
		serviceConfig: config,
	}
}

func (s *FileRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	var group string
	if v, ok := s.serviceConfig.VgroupMapping[key]; ok {
		group = v
	}
	if group == "" {
		log.Errorf("vgroup is empty. key: %s", key)
		return nil, fmt.Errorf("vgroup is empty. key: %s", key)
	}

	var addrStr string
	if v, ok := s.serviceConfig.Grouplist[group]; ok {
		addrStr = v
	}
	if addrStr == "" {
		log.Errorf("endpoint is empty. key: %s group: %s", group)
		return nil, fmt.Errorf("endpoint is empty. key: %s group: %s", key, group)
	}

	addrs := strings.Split(addrStr, endPointSplitChar)
	instances := make([]*ServiceInstance, 0)
	for _, addr := range addrs {
		ipPort := strings.Split(addr, ipPortSplitChar)
		if len(ipPort) != 2 {
			return nil, fmt.Errorf("endpoint format should like ip:port. endpoint: %s", addr)
		}
		ip := ipPort[0]
		port, err := strconv.Atoi(ipPort[1])
		if err != nil {
			return nil, err
		}
		instances = append(instances, &ServiceInstance{
			Addr: ip,
			Port: port,
		})
	}
	return instances, nil
}

func (s *FileRegistryService) Close() {

}
