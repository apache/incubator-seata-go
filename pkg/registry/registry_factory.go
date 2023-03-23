package registry

import (
	"fmt"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

// GetRegistry get register by config type
func GetRegistry(config *Config) (rs RegistryService, e error) {
	switch config.Type {
	case types.File:
		rs = NewFileRegistryService(config.NacosConfig)
	case types.Nacos:
		rs = NewNacosRegistryService(config.NacosConfig)
	case types.Etcd:
		e = fmt.Errorf("not implemented etcd register center")
	default:
		rs = NewFileRegistryService(config.NacosConfig)
	}
	return rs, e
}
