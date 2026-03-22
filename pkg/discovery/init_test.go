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
	"reflect"
	"strings"
	"testing"
)

func TestInitRegistry(t *testing.T) {
	type args struct {
		serviceConfig  *ServiceConfig
		registryConfig *RegistryConfig
	}
	tests := []struct {
		name                 string
		args                 args
		hasPanic             bool
		expectedType         string
		panicMessageContains string
	}{
		{
			name: "file",
			args: args{
				registryConfig: &RegistryConfig{
					Type: FILE,
				},
				serviceConfig: &ServiceConfig{},
			},
			expectedType: "FileRegistryService",
		},
		{
			name: "etcd",
			args: args{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
				},
				registryConfig: &RegistryConfig{
					Type: ETCD,
					Etcd3: Etcd3Config{
						ServerAddr: "127.0.0.1:2379",
						Cluster:    "default",
					},
				},
			},
			hasPanic:     false,
			expectedType: "EtcdRegistryService",
		},
		{
			name: "unknown type",
			args: args{
				registryConfig: &RegistryConfig{
					Type: "unknown",
				},
				serviceConfig: &ServiceConfig{},
			},
			hasPanic: true,
		},
		{
			name: "nacos type is not implemented",
			args: args{
				registryConfig: &RegistryConfig{
					Type: NACOS,
				},
				serviceConfig: &ServiceConfig{},
			},
			hasPanic:             true,
			panicMessageContains: fmt.Sprintf("service registry type %s is not implemented", NACOS),
		},
		{
			name: "eureka type is not implemented",
			args: args{
				registryConfig: &RegistryConfig{
					Type: EUREKA,
				},
				serviceConfig: &ServiceConfig{},
			},
			hasPanic:             true,
			panicMessageContains: fmt.Sprintf("service registry type %s is not implemented", EUREKA),
		},
		{
			name: "redis type is not implemented",
			args: args{
				registryConfig: &RegistryConfig{
					Type: REDIS,
				},
				serviceConfig: &ServiceConfig{},
			},
			hasPanic:             true,
			panicMessageContains: fmt.Sprintf("service registry type %s is not implemented", REDIS),
		},
		{
			name: "zk type is not implemented",
			args: args{
				registryConfig: &RegistryConfig{
					Type: ZK,
				},
				serviceConfig: &ServiceConfig{},
			},
			hasPanic:             true,
			panicMessageContains: fmt.Sprintf("service registry type %s is not implemented", ZK),
		},
		{
			name: "consul type is not implemented",
			args: args{
				registryConfig: &RegistryConfig{
					Type: CONSUL,
				},
				serviceConfig: &ServiceConfig{},
			},
			hasPanic:             true,
			panicMessageContains: fmt.Sprintf("service registry type %s is not implemented", CONSUL),
		},
		{
			name: "sofa type is not implemented",
			args: args{
				registryConfig: &RegistryConfig{
					Type: SOFA,
				},
				serviceConfig: &ServiceConfig{},
			},
			hasPanic:             true,
			panicMessageContains: fmt.Sprintf("service registry type %s is not implemented", SOFA),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registryServiceInstance = nil
			defer func() {
				if r := recover(); r != nil {
					if !tt.hasPanic {
						t.Errorf("panic is not expected!")
						return
					}
					if tt.panicMessageContains != "" && !strings.Contains(fmt.Sprint(r), tt.panicMessageContains) {
						t.Errorf("panic = %v, want message containing %q", r, tt.panicMessageContains)
					}
				} else if tt.hasPanic {
					t.Errorf("Expected a panic but did not receive one")
				}
			}()
			InitRegistry(tt.args.serviceConfig, tt.args.registryConfig)
			instance := GetRegistry()
			if !tt.hasPanic {
				actualType := reflect.TypeOf(instance).Elem().Name()
				if actualType != tt.expectedType {
					t.Errorf("type = %v, want %v", actualType, tt.expectedType)
				}
			}
		})
	}
}
