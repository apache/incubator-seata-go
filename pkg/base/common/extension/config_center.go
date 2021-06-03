package extension

import (
	"errors"
	"sync"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
)

var (
	configCentersMu sync.RWMutex
	configCenters   = make(map[string]func(conf *config.ConfigCenterConfig) (config_center.DynamicConfigurationFactory, error))
)

func SetConfigCenter(name string, v func(conf *config.ConfigCenterConfig) (config_center.DynamicConfigurationFactory, error)) {
	configCentersMu.Lock()
	defer configCentersMu.Unlock()
	configCenters[name] = v
}

func GetConfigCenter(name string, conf *config.ConfigCenterConfig) (config_center.DynamicConfigurationFactory, error) {
	configCentersMu.RLock()
	configCenter := configCenters[name]
	configCentersMu.RUnlock()
	if configCenter == nil {
		return nil, errors.New("config center for " + name + " is not existing, make sure you have import the package.")
	}
	return configCenter(conf)
}
