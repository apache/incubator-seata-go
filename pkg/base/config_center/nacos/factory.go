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
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
	"github.com/transaction-wg/seata-golang/pkg/base/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/extension"
)

func init() {
	extension.SetConfigCenter(constant.NacosKey, newNacosConfigCenterFactory)
}

type nacosConfigCenter struct {
	client config_client.IConfigClient
}

func (nc *nacosConfigCenter) AddListener(conf *config.ConfigCenterConfig, listener config_center.ConfigurationListener) {
	dataID := getDataID(conf)
	group := getGroup(conf)
	_ = nc.client.ListenConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			go listener.Process(&config_center.ConfigChangeEvent{Key: dataId, Value: data})
		},
	})
}

func getGroup(conf *config.ConfigCenterConfig) string {
	group := conf.NacosConfig.Group
	if group == "" {
		group = constant.NacosDefaultGroup
	}
	return group
}

func getDataID(conf *config.ConfigCenterConfig) string {
	dataID := conf.NacosConfig.DataID
	if dataID == "" {
		dataID = constant.NacosDefaultDataID
	}
	return dataID
}

func (nc *nacosConfigCenter) GetConfig(conf *config.ConfigCenterConfig) string {
	dataID := getDataID(conf)
	group := getGroup(conf)
	config, _ := nc.client.GetConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group})
	return config
}

func (nc *nacosConfigCenter) Stop() error {
	// TODO: Handle nacos config center shutdown
	return nil
}

func newNacosConfigCenterFactory(conf *config.ConfigCenterConfig) (factory config_center.DynamicConfigurationFactory, e error) {
	nacosConfig, err := getNacosConfig(conf)
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
func getNacosConfig(conf *config.ConfigCenterConfig) (map[string]interface{}, error) {
	configMap := make(map[string]interface{}, 2)
	addr := conf.NacosConfig.ServerAddr

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
	clientConfig.Username = conf.NacosConfig.UserName
	clientConfig.Password = conf.NacosConfig.Password
	configMap[nacosConstant.KEY_CLIENT_CONFIG] = clientConfig

	return configMap, nil
}
