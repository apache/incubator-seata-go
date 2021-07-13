package file

import (
	"strings"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	"github.com/transaction-wg/seata-golang/pkg/client/config"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

func init() {
	extension.SetRegistry(constant.FileKey, newFileRegistry)
}

type fileRegistry struct {
}

func (nr *fileRegistry) Register(addr *registry.Address) error {
	log.Info("file register")
	return nil
}

func (nr *fileRegistry) UnRegister(addr *registry.Address) error {
	return nil
}

func (nr *fileRegistry) Lookup() ([]string, error) {
	addressList := strings.Split(config.GetClientConfig().TransactionServiceGroup, ",")
	return addressList, nil
}

func (nr *fileRegistry) Subscribe(notifyListener registry.EventListener) error {
	return nil
}

func (nr *fileRegistry) UnSubscribe(notifyListener registry.EventListener) error {
	return nil
}

func newFileRegistry() (registry.Registry, error) {
	tmpRegistry := &fileRegistry{}
	return tmpRegistry, nil
}
