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
	if err := InitRegistryWithError(serviceConfig, registryConfig); err != nil {
		panic(fmt.Errorf("init service registry err:%v", err))
	}
}

func InitRegistryWithError(serviceConfig *ServiceConfig, registryConfig *RegistryConfig) error {
	if registryConfig == nil {
		return fmt.Errorf("registry config is nil")
	}

	provider, ok := registryProviderFor(registryConfig.Type)
	if !ok {
		return unsupportedRegistryTypeError(registryConfig.Type)
	}

	registryService, err := provider(serviceConfig, registryConfig)
	if err != nil {
		return err
	}
	if registryService == nil {
		return fmt.Errorf("registry provider returned nil for type:%s", registryConfig.Type)
	}
	registryServiceInstance = registryService
	return nil
}

func GetRegistry() RegistryService {
	return registryServiceInstance
}

func GetNamingServerRegistry() (NamingServerRegistry, error) {
	if registryServiceInstance == nil {
		return nil, fmt.Errorf("registry service not initialized")
	}
	namingReg, ok := registryServiceInstance.(NamingServerRegistry)
	if !ok {
		return nil, fmt.Errorf("current registry is not namingserver")
	}
	return namingReg, nil
}

func GetNamingserverRegistry() (NamingServerRegistry, error) {
	return GetNamingServerRegistry()
}
