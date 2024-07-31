package discovery

import (
	"reflect"
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
			name: "nacos",
			args: args{
				registryConfig: &RegistryConfig{
					Type: NACOS,
				},
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
				},
			},
			hasPanic:     false,
			expectedType: "NacosRegistryService",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
