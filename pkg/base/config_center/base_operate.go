package config_center

import "github.com/transaction-wg/seata-golang/pkg/base/config"

// AddListener add config center listener
func AddListener(cc DynamicConfigurationFactory, conf *config.ConfigCenterConfig, listener ConfigurationListener) {
	if conf.Mode == "" {
		return
	}
	cc.AddListener(conf, listener)
}

// LoadConfigCenterConfig load config center config
func LoadConfigCenterConfig(cc DynamicConfigurationFactory, conf *config.ConfigCenterConfig, listener ConfigurationListener) string {
	remoteConfig := cc.GetConfig(conf)
	// listen remote config, change config item
	AddListener(cc, conf, listener)
	return remoteConfig
}
