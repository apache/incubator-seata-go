package etcdv3

// TODO: Import Standard
import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

import (
	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	utils "github.com/transaction-wg/seata-golang/pkg/util/etcdv3"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

func init() {
	extension.SetRegistry(constant.ETCDV3_KEY, newETCDRegistry)
}

type etcdEventListener struct {
}

func (l *etcdEventListener) OnEvent(service []*registry.Service) error {
	data, err := json.Marshal(service)
	if err != nil {
		return err
	}
	log.Info("service info change: " + string(data))
	return nil
}

// TODO: Dynamic Configuration Support
type etcdRegistry struct {
	client      *clientv3.Client
	clusterName string
	leaseWrp    leaseWrapper
}

type leaseWrapper struct {
	rwMutex        sync.RWMutex
	wg             sync.WaitGroup
	leaseId        *clientv3.LeaseID
	isLeaseRunning bool
}

// Lookup Service Discovery
func (r *etcdRegistry) Lookup() ([]string, error) {
	// TODO: Discover Service and Subscribe
	//resp, err := r.client.Get(context.Background(), constant.ETCDV3_REGISTRY_PREFIX, clientv3.WithPrefix())
	//if err != nil {
	//	return nil, err
	//}
	//for _, kv := range resp.Kvs {
	//
	//}
	return nil, nil
}

func (r *etcdRegistry) Register(addr *registry.Address) error {
	// Make Sure Address Not Nil and Port Not Equals Zero
	if res := utils.IsAddressValid(*addr); !res {
		return errors.New("the address to register is invalid")
	}
	// RegistryKey Format: etcdv3-seata-clusterName-ipAddress:port
	// RegistryValue Format: ipAddress:port
	_, err := r.client.Put(context.Background(), utils.BuildRegistryKey(r.clusterName, addr), utils.BuildRegistryValue(addr), clientv3.WithLease(*r.leaseWrp.leaseId))
	return err
}

func (r *etcdRegistry) UnRegister(addr *registry.Address) error {
	_, err := r.client.Delete(context.Background(), utils.BuildRegistryKey(r.clusterName, addr))
	return err
}

func (r *etcdRegistry) Subscribe(listener registry.EventListener) error {
	// TODO: Implement Subscribe
	return nil
}

func (r *etcdRegistry) UnSubscribe(listener registry.EventListener) error {
	// TODO: Implement UnSubscribe
	return nil
}

// leaseKeeper Run in the Background to Renew Lease
func (r *etcdRegistry) leaseKeeper() {
	defer r.leaseWrp.wg.Done()

	r.leaseWrp.rwMutex.Lock()
	r.leaseWrp.isLeaseRunning = true
	r.leaseWrp.rwMutex.Unlock()

	for {
		r.leaseWrp.rwMutex.RLock()
		isRunning := r.leaseWrp.isLeaseRunning
		r.leaseWrp.rwMutex.RUnlock()

		if isRunning {
			ttl, err := r.client.TimeToLive(context.Background(), *r.leaseWrp.leaseId)
			if err != nil {
				log.Warnf("failed to attain ttl info, %s", err.Error())
			}

			if ttl == nil {
				log.Warn("failed to renew ttl, ttl info in resp is nil")
				continue
			}

			if ttl.TTL <= constant.ETCDV3_LEASE_TTL_CRITICAL {
				_, err := r.client.KeepAliveOnce(context.Background(), *r.leaseWrp.leaseId)
				if err != nil {
					log.Warnf("failed to renew ttl, %s", err.Error())
				}
			}

			time.Sleep(time.Duration(constant.ETCDV3_LEASE_RENEW_INTERVAL) * time.Second)
		} else {
			break
		}
	}

}

// Stop wait for goroutines to stop
func (r *etcdRegistry) Stop() {
	r.leaseWrp.rwMutex.Lock()
	// Make LeaseKeeper in the Goroutine Stop
	r.leaseWrp.isLeaseRunning = false
	r.leaseWrp.rwMutex.Unlock()
	// Wait for Goroutine Ended
	r.leaseWrp.wg.Wait()
}

func newETCDRegistry() (registry.Registry, error) {
	// TODO: Handle Registry Error
	registryConfig := config.GetRegistryConfig()
	etcdConfig, err := utils.ToEtcdConfig(registryConfig.ETCDConfig, context.Background())
	if err != nil {
		return &etcdRegistry{}, err
	}

	client, err := clientv3.New(etcdConfig)
	if err != nil {
		return &etcdRegistry{}, err
	}

	resp, err := client.Grant(context.Background(), constant.ETCDV3_LEASE_TTL)
	if err != nil {
		return &etcdRegistry{}, err
	}

	r := &etcdRegistry{
		client:      client,
		clusterName: registryConfig.ETCDConfig.ClusterName,
		leaseWrp: leaseWrapper{
			rwMutex:        sync.RWMutex{},
			wg:             sync.WaitGroup{},
			leaseId:        &resp.ID,
			isLeaseRunning: false,
		},
	}

	r.leaseWrp.wg.Add(1)
	go r.leaseKeeper()

	return r, nil
}
