package etcdv3

import (
	"testing"
)

import (
	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	utils "github.com/transaction-wg/seata-golang/pkg/util/etcdv3"
)

func initRegistry(t *testing.T) registry.Registry {
	t.Helper()

	confStr := `
type: etcdv3
etcdv3:
  endpoints: 127.0.0.1:2379
`
	rc := config.GetServerConfig().RegistryConfig.EtcdConfig
	err := defaults.Set(&rc)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte(confStr), &rc)
	assert.NoError(t, err)

	r, err := newETCDRegistry()
	if err != nil {
		return nil
	}

	return r
}

func TestRegister(t *testing.T) {
	r := initRegistry(t)
	defer r.Stop()

	addr := &registry.Address{
		IP:   "127.0.0.1",
		Port: 8888,
	}

	err := r.Register(addr)
	assert.NoError(t, err)

	er := r.(*etcdRegistry)

	val, err := er.client.Get(utils.BuildRegistryValue(addr))
	assert.NoError(t, err)
	assert.Equal(t, utils.BuildRegistryValue(addr), val)
}

func TestUnRegister(t *testing.T) {
	r := initRegistry(t)
	defer r.Stop()

	addr := &registry.Address{
		IP:   "127.0.0.1",
		Port: 8888,
	}

	err := r.Register(addr)
	assert.NoError(t, err)

	err = r.UnRegister(addr)
	assert.NoError(t, err)
}
