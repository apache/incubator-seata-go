package etcdv3

// TODO: Import Standard
import (
	"context"
	"sync"
	"testing"
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
`
	regCfg := config.RegistryConfig{}
	defaults.Set(&regCfg)
	err := yaml.Unmarshal([]byte(confStr), &regCfg)
	assert.NoError(t, err)

	etcdConfig, err := utils.ToEtcdConfig(regCfg.ETCDConfig, ctx)
	assert.NoError(t, err)
	client, err := clientv3.New(etcdConfig)
	assert.NoError(t, err)
	_, err = client.Delete(ctx, "", clientv3.WithPrefix())
	assert.NoError(t, err)

	resp, err := client.Grant(ctx, constant.ETCDV3_LEASE_TTL)
	assert.NoError(t, err, "failed to recv lease response")
	assert.NotNil(t, resp)

	return &etcdRegistry{
		client:      client,
		clusterName: regCfg.ETCDConfig.ClusterName,
		leaseWrp: leaseWrapper{
			rwMutex:        sync.RWMutex{},
			wg:             sync.WaitGroup{},
			leaseId:        &resp.ID,
			isLeaseRunning: false,
		},
	}
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

	resp, err := r.client.Get(ctx, utils.BuildRegistryValue(addr))
	assert.NoError(t, err)
	assert.NotZero(t, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		assert.Equal(t, utils.BuildRegistryValue(addr), kv.Value)
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
}
