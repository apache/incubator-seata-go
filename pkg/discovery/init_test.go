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
		name         string
		args         args
		hasPanic     bool
		expectedType string
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				if registryServiceInstance != nil {
					registryServiceInstance.Close()
					registryServiceInstance = nil
				}
			})
			defer func() {
				if r := recover(); r != nil {
					if !tt.hasPanic {
						t.Errorf("panic is not expected!")
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

func TestInitRegistryWithErrorUnsupportedType(t *testing.T) {
	registryServiceInstance = nil
	t.Cleanup(func() {
		registryServiceInstance = nil
	})

	err := InitRegistryWithError(&ServiceConfig{}, &RegistryConfig{Type: "unknown"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "service registry not support registry type:unknown" {
		t.Fatalf("unexpected error: %v", err)
	}
	if GetRegistry() != nil {
		t.Fatal("registry should not be initialized on error")
	}
}

func TestInitRegistryWithErrorNilRegistryConfig(t *testing.T) {
	err := InitRegistryWithError(&ServiceConfig{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "registry config is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitRegistryWithErrorReturnsProviderError(t *testing.T) {
	registryServiceInstance = nil
	t.Cleanup(func() {
		registryServiceInstance = nil
	})

	err := InitRegistryWithError(nil, &RegistryConfig{Type: FILE})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "service config is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
	if GetRegistry() != nil {
		t.Fatal("registry should not be initialized on error")
	}
}

func TestInitRegistryWithErrorKeepsExistingRegistryOnError(t *testing.T) {
	registryServiceInstance = nil
	t.Cleanup(func() {
		registryServiceInstance = nil
	})

	if err := InitRegistryWithError(&ServiceConfig{}, &RegistryConfig{Type: FILE}); err != nil {
		t.Fatalf("InitRegistryWithError() error = %v", err)
	}
	existing := GetRegistry()
	if existing == nil {
		t.Fatal("registry is nil")
	}

	err := InitRegistryWithError(nil, &RegistryConfig{Type: FILE})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "service config is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
	if GetRegistry() != existing {
		t.Fatal("existing registry should be kept on error")
	}
}

func TestInitRegistryWithErrorNilProviderResult(t *testing.T) {
	const providerType = "empty"
	oldProvider, hadOldProvider := registryProviders[providerType]
	registryProviders[providerType] = func(*ServiceConfig, *RegistryConfig) (RegistryService, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		if hadOldProvider {
			registryProviders[providerType] = oldProvider
		} else {
			delete(registryProviders, providerType)
		}
	})

	err := InitRegistryWithError(&ServiceConfig{}, &RegistryConfig{Type: providerType})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "registry provider returned nil for type:empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitRegistryWithErrorSupportedProviders(t *testing.T) {
	tests := []struct {
		name           string
		serviceConfig  *ServiceConfig
		registryConfig *RegistryConfig
		expectedType   string
		cleanup        func()
	}{
		{
			name: "raft",
			serviceConfig: &ServiceConfig{
				VgroupMapping: map[string]string{
					"default_tx_group": "default",
				},
			},
			registryConfig: &RegistryConfig{
				Type: RAFT,
				Raft: RaftConfig{
					ServerAddr: "127.0.0.1:7091",
				},
			},
			expectedType: "RaftRegistryService",
		},
		{
			name:          "namingserver",
			serviceConfig: nil,
			registryConfig: &RegistryConfig{
				Type: NAMINGSERVER,
				NamingServer: NamingServerConfig{
					ServerAddr:      "127.0.0.1:8081",
					HeartbeatPeriod: 5000,
				},
			},
			expectedType: "NamingServerRegistryService",
			cleanup:      resetInstance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registryServiceInstance = nil
			if tt.cleanup != nil {
				tt.cleanup()
			}
			t.Cleanup(func() {
				if tt.cleanup != nil {
					tt.cleanup()
					registryServiceInstance = nil
					return
				}
				if registryServiceInstance != nil {
					registryServiceInstance.Close()
					registryServiceInstance = nil
				}
			})

			if err := InitRegistryWithError(tt.serviceConfig, tt.registryConfig); err != nil {
				t.Fatalf("InitRegistryWithError() error = %v", err)
			}
			instance := GetRegistry()
			if instance == nil {
				t.Fatal("registry is nil")
			}
			actualType := reflect.TypeOf(instance).Elem().Name()
			if actualType != tt.expectedType {
				t.Fatalf("type = %v, want %v", actualType, tt.expectedType)
			}
		})
	}
}

func TestInitRegistryPanicCompatibility(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if !strings.Contains(fmt.Sprint(r), "init service registry err:service registry not support registry type:unknown") {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	InitRegistry(&ServiceConfig{}, &RegistryConfig{Type: "unknown"})
}
