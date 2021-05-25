package extension

import "github.com/transaction-wg/seata-golang/pkg/base/registry"

var (
	registrys = make(map[string]func() (registry.Registry, error))
)

// SetRegistry sets the registry extension with @name
func SetRegistry(name string, v func() (registry.Registry, error)) {
	registrys[name] = v
}

// GetRegistry finds the registry extension with @name
func GetRegistry(name string) (registry.Registry, error) {
	if registrys[name] == nil {
		panic("registry for " + name + " does not exist.")
	}
	return registrys[name]()

}
