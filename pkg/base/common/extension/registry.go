package extension

import (
	"errors"
	"sync"
)
import (
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
)

var (
	registrysMu sync.RWMutex
	registrys   = make(map[string]func() (registry.Registry, error))
)

// SetRegistry sets the registry extension with @name
func SetRegistry(name string, v func() (registry.Registry, error)) {
	//写加锁，参考sql.go
	registrysMu.Lock()
	defer registrysMu.Unlock()
	registrys[name] = v
}

// GetRegistry finds the registry extension with @name
func GetRegistry(name string) (registry.Registry, error) {
	registrysMu.RLock()
	registry := registrys[name]
	registrysMu.RUnlock()
	if registry == nil {
		return nil, errors.New("registry for " + name + " is not existing, make sure you have import the package.")
	}
	return registry()

}
