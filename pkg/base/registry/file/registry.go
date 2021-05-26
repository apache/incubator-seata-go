package file

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	"github.com/transaction-wg/seata-golang/pkg/client/config"
	"log"
	"strings"
)

func init() {
	extension.SetRegistry(constant.FILE_KEY, newFileRegistry)
}

type fileRegistry struct {
}

func (nr *fileRegistry) Register(addr *registry.Address) error {
	//文件不需要注册
	log.Print("file register")
	return nil
}

func (nr *fileRegistry) UnRegister(addr *registry.Address) error {
	return nil
}
func (nr *fileRegistry) Lookup() ([]string, error) {
	addressList := strings.Split(config.GetClientConfig().TransactionServiceGroup, ",")
	return addressList, nil
}

// newNacosRegistry will create new instance
func newFileRegistry() (registry.Registry, error) {

	tmpRegistry := &fileRegistry{}
	return tmpRegistry, nil
}
