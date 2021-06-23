package config_center

import "github.com/transaction-wg/seata-golang/pkg/base/config"

func AddLisenter(cc DynamicConfigurationFactory, conf *config.ConfigCenterConfig, listener ConfigurationListener) {
	if conf.Mode == "" {
		return
	}
	cc.AddListener(conf, listener)
}

func LoadConfigCenterConfig(cc DynamicConfigurationFactory, conf *config.ConfigCenterConfig, listener ConfigurationListener) string {
	confStr := cc.GetConfig(conf)
	//监听远程配置情况，发生变更进行配置修改
	AddLisenter(cc, conf, listener)
	return confStr
}
