package nacos

import (
	"net"
	"strconv"
	"strings"
)
import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
)

func init() {
	extension.SetConfigCenter("nacos", newNacosConfigCenterFactory)
}

type nacosConfigCenter struct {
	client config_client.IConfigClient
}

func (nc *nacosConfigCenter) AddListener(listener config_center.ConfigurationListener) {
	configCenterConfig := config.GetConfigCenterConfig()
	dataId := configCenterConfig.NacosConfig.DataId
	if dataId == "" {
		dataId = constant.NACOS_DEFAULT_DATA_ID
	}
	group := configCenterConfig.NacosConfig.Group
	if group == "" {
		group = constant.NACOS_DEFAULT_GROUP
	}
	_ = nc.client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			go listener.Process(&config_center.ConfigChangeEvent{Key: dataId, Value: data})
		},
	})
}

func (nc *nacosConfigCenter) GetConfig() string {
	configCenterConfig := config.GetConfigCenterConfig()
	dataId := configCenterConfig.NacosConfig.DataId
	if dataId == "" {
		dataId = constant.NACOS_DEFAULT_DATA_ID
	}
	group := configCenterConfig.NacosConfig.Group
	if group == "" {
		group = constant.NACOS_DEFAULT_GROUP
	}
	config, _ := nc.client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group})
	return config
}

func newNacosConfigCenterFactory() (factory config_center.DynamicConfigurationFactory, e error) {
	nacosConfig, err := getNacosConfig()
	if err != nil {
		return &nacosConfigCenter{}, err
	}
	client, err := clients.CreateConfigClient(nacosConfig)
	if err != nil {
		return &nacosConfigCenter{}, err
	}
	cc := &nacosConfigCenter{
		client: client,
	}
	return cc, nil
}

//获取Nacos配置信息
func getNacosConfig() (map[string]interface{}, error) {
	configCenterConfig := config.GetConfigCenterConfig()
	configMap := make(map[string]interface{}, 2)
	addr := configCenterConfig.NacosConfig.ServerAddr

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
	clientConfig.Username = configCenterConfig.NacosConfig.UserName
	clientConfig.Password = configCenterConfig.NacosConfig.Password
	configMap[nacosConstant.KEY_CLIENT_CONFIG] = clientConfig

	return configMap, nil
}
