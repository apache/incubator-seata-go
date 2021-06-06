package extension

import (
	"sync"
)
import (
	"github.com/pkg/errors"
)
import (
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
)

var (
	registriesMu sync.RWMutex
	registries   = make(map[string]func() (registry.Registry, error))
)

// SetRegistry sets the registry extension with @name
func SetRegistry(name string, v func() (registry.Registry, error)) {
	//写加锁，参考sql.go
	registriesMu.Lock()
	defer registriesMu.Unlock()
	if v == nil {
		panic("registry: Register  v is nil")
	}
	if _, dup := registries[name]; dup {
		panic("registry: Register called twice for registry " + name)
	}
	registries[name] = v
}

// GetRegistry finds the registry extension with @name
func GetRegistry(name string) (registry.Registry, error) {
	registriesMu.RLock()
	registry := registries[name]
	registriesMu.RUnlock()
	if registry == nil {
		return nil, errors.Errorf("registry for " + name + " is not existing, make sure you have import the package.")
	}
	return registry()

}
