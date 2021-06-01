package extension

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
)

var (
	configCenters = make(map[string]func() (config_center.DynamicConfigurationFactory, error))
)

func SetConfigCenter(name string, v func() (config_center.DynamicConfigurationFactory, error)) {
	configCenters[name] = v
}

func GetConfigCenter(name string) (config_center.DynamicConfigurationFactory, error) {
	if configCenters[name] == nil {
		panic("config center for " + name + " is not existing, make sure you have import the package.")
	}
	return configCenters[name]()

}
