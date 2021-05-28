package nacos

import (
	"encoding/json"
	"errors"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	"log"

	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	clientConfig "github.com/transaction-wg/seata-golang/pkg/client/config"
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
type nacosEventListener struct {
}

func (nr *nacosEventListener) OnEvent(service []*registry.Service) error {
	data, err := json.Marshal(service)

	log.Print("servie info change：" + string(data))
	return err
}
func (nr *nacosRegistry) Register(addr *registry.Address) error {
	registryConfig := config.GetRegistryConfig()

	param := createRegisterParam(registryConfig, addr)
	isRegistry, err := nr.namingClient.RegisterInstance(param)
	if err != nil {
		return err
	}
	if !isRegistry {
		return errors.New("registry [" + registryConfig.NacosConfig.Application + "] to  nacos failed")
	}
	return nil
}

//创建服务注册信息
func createRegisterParam(registryConfig config.RegistryConfig, addr *registry.Address) vo.RegisterInstanceParam {
	serviceName := registryConfig.NacosConfig.Application
	params := make(map[string]string)

	instance := vo.RegisterInstanceParam{
		Ip:          addr.IP,
		Port:        addr.Port,
		Metadata:    params,
		Weight:      1,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		ServiceName: serviceName,
		ClusterName: registryConfig.NacosConfig.Cluster, // default value is DEFAULT
		GroupName:   registryConfig.NacosConfig.Group,   // default value is DEFAULT_GROUP
	}
	return instance
}

func (nr *nacosRegistry) UnRegister(addr *registry.Address) error {
	return nil
}

//noinspection ALL
func (nr *nacosRegistry) Lookup() ([]string, error) {
	registryConfig := clientConfig.GetRegistryConfig()
	clusterName := registryConfig.NacosConfig.Cluster
	instances, err := nr.namingClient.SelectInstances(vo.SelectInstancesParam{
		ServiceName: registryConfig.NacosConfig.Application,
		GroupName:   registryConfig.NacosConfig.Group, // default value is DEFAULT_GROUP
		Clusters:    []string{clusterName},            // default value is DEFAULT
		HealthyOnly: true,
	})
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0)
	for _, instance := range instances {
		addrs = append(addrs, instance.Ip+":"+strconv.FormatUint(instance.Port, 10))
	}
	//订阅服务
	nr.Subscribe(&nacosEventListener{})
	return addrs, nil
}
func (nr *nacosRegistry) Subscribe(notifyListener registry.EventListener) error {
	registryConfig := clientConfig.GetRegistryConfig()
	clusterName := registryConfig.NacosConfig.Cluster
	err := nr.namingClient.Subscribe(&vo.SubscribeParam{
		ServiceName: registryConfig.NacosConfig.Application,
		GroupName:   registryConfig.NacosConfig.Group, // default value is DEFAULT_GROUP
		Clusters:    []string{clusterName},            // default value is DEFAULT
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			serviceList := make([]*registry.Service, 0)
			for _, s := range services {
				serviceList = append(serviceList, &registry.Service{
					IP:   s.Ip,
					Port: s.Port,
					Name: s.ServiceName,
				})
			}
			notifyListener.OnEvent(serviceList)
		},
	})

	return err
}

func (nr *nacosRegistry) UnSubscribe(notifyListener registry.EventListener) error {
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
	if registryConfig.Mode == "" {
		registryConfig = clientConfig.GetRegistryConfig()
	}
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
