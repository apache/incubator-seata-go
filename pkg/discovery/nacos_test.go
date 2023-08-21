package discovery

import (
	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/seata/seata-go/pkg/discovery/mock"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestNacosRegistryService_Lookup(t *testing.T) {
	ctrl := gomock.NewController(t)
	tests := []struct {
		name             string
		nacosConfig      NacosConfig
		nacosAllInstance []model.Instance
		expect           []*ServiceInstance
		hasErr           bool
	}{
		{
			name:        "normal",
			nacosConfig: NacosConfig{ServerAddr: "127.0.0.1:8848", Group: "seata-server", Application: "seata"},
			nacosAllInstance: []model.Instance{
				{Ip: "127.0.0.1", Healthy: true, Port: 8091},
			},
			expect: []*ServiceInstance{
				{
					Addr: "127.0.0.1",
					Port: 8091,
				},
			},
		},
		{

			name:        "multi instance in nacos server",
			nacosConfig: NacosConfig{ServerAddr: "127.0.0.1:8848", Group: "seata-server", Application: "seata"},
			nacosAllInstance: []model.Instance{
				{Ip: "127.0.0.1", Healthy: true, Port: 8091},
				{Ip: "127.0.0.1", Healthy: true, Port: 8081},
			},
			expect: []*ServiceInstance{
				{Addr: "127.0.0.1", Port: 8091},
				{Addr: "127.0.0.1", Port: 8081},
			},
		},
	}

	for _, tt := range tests {
		mockNacosClient := mock.NewMockNacosClient(ctrl)
		mockNacosClient.EXPECT().SelectAllInstances(gomock.Any()).Return(tt.nacosAllInstance, nil).AnyTimes()
		registryService := &NacosRegistryService{nc: mockNacosClient, nacosServerConfig: tt.nacosConfig}
		serviceInstances, ok := registryService.Lookup("")
		if tt.hasErr {
			assert.Truef(t, ok != nil, "expected throw erorr ,actual erorr is nil")
			return
		}
		assert.True(t, reflect.DeepEqual(serviceInstances, tt.expect))
	}

}

func TestNewNacosRegistryService(t *testing.T) {
	tests := []struct {
		name        string
		nacosConfig NacosConfig
		expected    []constant.ServerConfig
		hasErr      bool
	}{
		{
			name: "single server addr",
			nacosConfig: NacosConfig{
				ServerAddr: "127.0.0.1:8848",
			},
			expected: []constant.ServerConfig{
				{
					IpAddr: "127.0.0.1",
					Port:   uint64(8848),
				},
			},
		},
		{
			name: "multi server addr with ';' split",
			nacosConfig: NacosConfig{
				ServerAddr: "127.0.0.1:8848;127.0.0.1:8849",
			},
			expected: []constant.ServerConfig{
				{
					IpAddr: "127.0.0.1",
					Port:   uint64(8848),
				},
				{
					IpAddr: "127.0.0.1",
					Port:   uint64(8849),
				},
			},
		},
		{
			name:        "invalid addr",
			nacosConfig: NacosConfig{ServerAddr: "127.0.0.18848"},
			hasErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseNacosServerConfig(tt.nacosConfig)
			if tt.hasErr {
				assert.Truef(t, err != nil, "expected throw erorr ,actual erorr is nil")
				return
			}
			reflect.DeepEqual(actual, tt.expected)
			nacosRegistryClient := NewNacosRegistryService(tt.nacosConfig)
			properties := make(map[string]interface{})
			properties[constant.KEY_SERVER_CONFIGS] = actual
			client, _ := clients.CreateNamingClient(properties)
			reflect.DeepEqual(nacosRegistryClient, &NacosRegistryService{nc: client, nacosServerConfig: tt.nacosConfig})

		})
	}

}
