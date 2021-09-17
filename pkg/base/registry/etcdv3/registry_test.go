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
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	utils "github.com/transaction-wg/seata-golang/pkg/util/etcdv3"
)

var (
	addr = &registry.Address{
		IP:   "127.0.0.1",
		Port: 8888,
	}
)

func initRegistry(t *testing.T) registry.Registry {
	t.Helper()

	confStr := `
type: etcdv3
etcdv3:
  cluster_name: seata-golang-etcdv3-test
  endpoints: 127.0.0.1:52379
`

	rc := config.RegistryConfig{}
	err := defaults.Set(&rc)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte(confStr), &rc)
	assert.NoError(t, err)

	config.InitRegistryConfig(&rc)

	r, err := newETCDRegistry()
	if err != nil {
		return nil
	}

	return r
}

func TestRegister(t *testing.T) {
	r := initRegistry(t)
	defer r.Stop()

	err := r.Register(addr)
	assert.NoError(t, err)

	er := r.(*etcdRegistry)

	val, err := er.client.Get(utils.BuildRegistryKey("seata-golang-etcdv3-test", addr))
	assert.NoError(t, err)
	assert.Equal(t, utils.BuildRegistryValue(addr), val)
}

func TestEtcdRegistry_Subscribe(t *testing.T) {
	r := initRegistry(t)
	defer r.Stop()

	er := r.(*etcdRegistry)

	_ = er.client.Put(utils.BuildRegistryKey("seata-golang-etcdv3-test", addr), utils.BuildRegistryValue(addr))
	err := er.Subscribe(&mockEventListener{t: t})
	assert.NoError(t, err)
}

type mockEventListener struct {
	t *testing.T
}

func (l *mockEventListener) OnEvent(service []*registry.Service) error {
	assert.Equal(l.t, 1, len(service))
	assert.Equal(l.t, "127.0.0.1", service[0].IP)
	assert.Equal(l.t, 8888, service[0].Port)
	return nil
}
