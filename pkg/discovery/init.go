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
)

var (
	registryServiceInstance RegistryService
)

func InitRegistry(serviceConfig *ServiceConfig, registryConfig *RegistryConfig) {
	var registryService RegistryService
	var err error
	switch registryConfig.Type {
	case FILE:
		//init file registry
		registryService = newFileRegistryService(serviceConfig)
	case ETCD:
		//init etcd registry
		registryService = newEtcdRegistryService(serviceConfig, &registryConfig.Etcd3)
	case NACOS:
		//TODO: init nacos registry
	case EUREKA:
		//TODO: init eureka registry
	case REDIS:
		//TODO: init redis registry
	case ZK:
		//TODO: init zk registry
	case CONSUL:
		//TODO: init consul registry
	case SOFA:
		//TODO: init sofa registry
	default:
		err = fmt.Errorf("service registry not support registry type:%s", registryConfig.Type)
	}

	if err != nil {
		panic(fmt.Errorf("init service registry err:%v", err))
	}
	registryServiceInstance = registryService
}

func GetRegistry() RegistryService {
	return registryServiceInstance
}
