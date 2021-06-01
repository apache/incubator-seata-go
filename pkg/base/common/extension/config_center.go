package extension

import (
	"errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
)

var (
	configCenters = make(map[string]func(conf *config.ConfigCenterConfig) (config_center.DynamicConfigurationFactory, error))
)

func SetConfigCenter(name string, v func(conf *config.ConfigCenterConfig) (config_center.DynamicConfigurationFactory, error)) {
	configCenters[name] = v
}

func GetConfigCenter(name string, conf *config.ConfigCenterConfig) (config_center.DynamicConfigurationFactory, error) {
	if configCenters[name] == nil {
		return nil, errors.New("config center for " + name + " is not existing, make sure you have import the package.")
	}
	return configCenters[name](conf)
}
