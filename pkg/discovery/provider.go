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

type RegistryProvider func(serviceConfig *ServiceConfig, registryConfig *RegistryConfig) (RegistryService, error)

var registryProviders = map[string]RegistryProvider{
	FILE: func(serviceConfig *ServiceConfig, _ *RegistryConfig) (RegistryService, error) {
		if serviceConfig == nil {
			return nil, fmt.Errorf("service config is nil")
		}
		return newFileRegistryService(serviceConfig), nil
	},
	ETCD3: func(serviceConfig *ServiceConfig, registryConfig *RegistryConfig) (RegistryService, error) {
		return newEtcdRegistryService(serviceConfig, &registryConfig.Etcd3)
	},
	RAFT: func(serviceConfig *ServiceConfig, registryConfig *RegistryConfig) (RegistryService, error) {
		if serviceConfig == nil {
			return nil, fmt.Errorf("service config is nil")
		}
		return NewRaftRegistryService(serviceConfig, registryConfig), nil
	},
	NAMINGSERVER: func(serviceConfig *ServiceConfig, registryConfig *RegistryConfig) (RegistryService, error) {
		return newNamingServerRegistryService(serviceConfig, &registryConfig.NamingServer), nil
	},
}

func registryProviderFor(registryType string) (RegistryProvider, bool) {
	provider, ok := registryProviders[normalizeRegistryType(registryType)]
	return provider, ok
}

func normalizeRegistryType(registryType string) string {
	if registryType == ETCD {
		return ETCD3
	}
	return registryType
}

func unsupportedRegistryTypeError(registryType string) error {
	return fmt.Errorf("service registry not support registry type:%s", registryType)
}
