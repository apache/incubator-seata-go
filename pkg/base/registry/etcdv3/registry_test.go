package etcdv3

import (
	"context"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	utils "github.com/transaction-wg/seata-golang/pkg/util/etcdv3"
)

func initRegistry(ctx context.Context, t *testing.T) *etcdRegistry {
	t.Helper()

	confStr := `
type: etcdv3
etcdv3:
  endpoints: 127.0.0.1:2379
  cluster_name: test
`
	regCfg := config.RegistryConfig{}
	err := defaults.Set(&regCfg)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte(confStr), &regCfg)
	assert.NoError(t, err)

	etcdConfig, err := utils.ToEtcdConfig(regCfg.ETCDConfig, ctx)
	assert.NoError(t, err)
	client, err := clientv3.New(etcdConfig)
	assert.NoError(t, err)
	_, err = client.Delete(ctx, "", clientv3.WithPrefix())
	assert.NoError(t, err)

	resp, err := client.Grant(ctx, constant.Etcdv3LeaseTtl)
	assert.NoError(t, err, "failed to recv lease response")
	assert.NotNil(t, resp)

	r := &etcdRegistry{
		client:      client,
		clusterName: regCfg.ETCDConfig.ClusterName,
		leaseWrp: leaseWrapper{
			rwMutex:        sync.RWMutex{},
			wg:             sync.WaitGroup{},
			leaseId:        &resp.ID,
			isLeaseRunning: false,
		},
	}

	r.leaseWrp.wg.Add(1)
	go r.leaseKeeper()

	return r
}

func TestRegister(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := initRegistry(ctx, t)
	defer r.Stop()

	addr := &registry.Address{
		IP:   "127.0.0.1",
		Port: 8888,
	}

	err := r.Register(addr)
	assert.NoError(t, err)

	key := utils.BuildRegistryKey(r.clusterName, addr)
	resp, err := r.client.Get(ctx, key, clientv3.WithLease(*r.leaseWrp.leaseId))
	assert.NoError(t, err)
	assert.NotZero(t, resp.Count)
	for _, kv := range resp.Kvs {
		assert.Equal(t, utils.BuildRegistryValue(addr), string(kv.Value))
	}
}

func TestUnRegister(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := initRegistry(ctx, t)
	defer r.Stop()

	addr := &registry.Address{
		IP:   "127.0.0.1",
		Port: 8888,
	}

	err := r.Register(addr)
	assert.NoError(t, err)

	err = r.UnRegister(addr)
	assert.NoError(t, err)

	resp, err := r.client.Get(ctx, utils.BuildRegistryKey(r.clusterName, addr), clientv3.WithLease(*r.leaseWrp.leaseId))
	assert.NoError(t, err)
	assert.Zero(t, resp.Count)

	_, err = r.client.Delete(ctx, constant.Etcdv3RegistryPrefix+r.clusterName, clientv3.WithLease(*r.leaseWrp.leaseId))
	assert.NoError(t, err)
}

type mockListener struct {
	counter     int
	t           *testing.T
	servicesMap map[string]string
}

func (m *mockListener) OnEvent(services []*registry.Service) error {
	assert.NotEmpty(m.t, services)
	for _, service := range services {
		_, ok := m.servicesMap[service.Name]
		assert.Equal(m.t, true, ok)
		m.counter++
	}
	return nil
}

func TestSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := initRegistry(ctx, t)
	defer r.Stop()

	services := []string{
		"127.0.0.1:11451",
		"127.0.0.1:11452",
		"127.0.0.1:11453",
		"127.0.0.1:11454",
	}

	servicesMap := make(map[string]string)

	prefixKey := constant.Etcdv3RegistryPrefix + r.clusterName + "-"
	l := &mockListener{servicesMap: servicesMap, t: t, counter: 0}
	err := r.Subscribe(l)
	assert.NoError(t, err)

	for _, service := range services {
		resp, err := r.client.Put(ctx, prefixKey+service, service)
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		l.servicesMap[prefixKey+service] = service
	}

	time.Sleep(1 * time.Second)
	assert.Equal(t, len(services), l.counter)

	_, err = r.client.Delete(ctx, prefixKey, clientv3.WithPrefix())
	assert.NoError(t, err)
}
