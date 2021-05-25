package nacos

import (
	"errors"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"net"
	"strconv"
	"strings"
)

func init() {
	extension.SetRegistry(constant.NACOS_KEY, newNacosRegistry)
}

type nacosRegistry struct {
	namingClient naming_client.INamingClient
}

func (nr *nacosRegistry) Register(addr *registry.Address) error {
	registryConfig := config.GetRegistryConfig()
	serviceName := registryConfig.NacosConfig.Application
	param := createRegisterParam(serviceName, addr)
	isRegistry, err := nr.namingClient.RegisterInstance(param)
	if err != nil {
		return err
	}
	if !isRegistry {
		return errors.New("registry [" + serviceName + "] to  nacos failed")
	}
	return nil
}

//创建服务注册信息
func createRegisterParam(serviceName string, addr *registry.Address) vo.RegisterInstanceParam {
	params := make(map[string]string)

	instance := vo.RegisterInstanceParam{
		Ip:          addr.IP,
		Port:        uint64(addr.Port),
		Metadata:    params,
		Weight:      1,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		ServiceName: serviceName,
	}
	return instance
}

func (nr *nacosRegistry) UnRegister(addr *registry.Address) error {
	return nil
}

// newNacosRegistry will create new instance
func newNacosRegistry() (registry.Registry, error) {
	nacosConfig, err := getNacosConfig()
	if err != nil {
		return &nacosRegistry{}, err
	}
	client, err := clients.CreateNamingClient(nacosConfig)
	if err != nil {
		return &nacosRegistry{}, err
	}
	tmpRegistry := &nacosRegistry{
		namingClient: client,
	}
	return tmpRegistry, nil
}

//获取Nacos配置信息
func getNacosConfig() (map[string]interface{}, error) {
	registryConfig := config.GetRegistryConfig()
	configMap := make(map[string]interface{}, 2)
	addr := registryConfig.NacosConfig.ServerAddr

	addresses := strings.Split(addr, ",")
	serverConfigs := make([]nacosConstant.ServerConfig, 0, len(addresses))
	for _, addr := range addresses {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		port, _ := strconv.Atoi(portStr)
		serverConfigs = append(serverConfigs, nacosConstant.ServerConfig{
			IpAddr: ip,
			Port:   uint64(port),
		})
	}
	configMap[nacosConstant.KEY_SERVER_CONFIGS] = serverConfigs

	var clientConfig nacosConstant.ClientConfig
	clientConfig.Username = registryConfig.NacosConfig.UserName
	clientConfig.Password = registryConfig.NacosConfig.Password
	configMap[nacosConstant.KEY_CLIENT_CONFIG] = clientConfig

	return configMap, nil
}
