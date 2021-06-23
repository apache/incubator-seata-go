package config_center

import "github.com/transaction-wg/seata-golang/pkg/base/config"

type DynamicConfigurationFactory interface {
	GetConfig(conf *config.ConfigCenterConfig) string                            //返回配置信息
	AddListener(conf *config.ConfigCenterConfig, listener ConfigurationListener) //添加配置监听

}
