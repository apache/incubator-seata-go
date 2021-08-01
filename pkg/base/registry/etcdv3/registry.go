package etcdv3

import (
	"context"
	"strconv"
	"strings"
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

func (l *etcdEventListener) OnEvent(services []*registry.Service) error {
	for _, service := range services {
		var eventType string
		if service.EventType == 0 {
			eventType = "PUT"
		} else {
			eventType = "DELETE"
		}
		log.Infof("service info change: {name=%s, eventType=%s, ip=%s, port=%d}", service.Name, eventType, service.IP, service.Port)
	}
	return nil
}

// TODO: Dynamic Configuration Support
type etcdRegistry struct {
	client           *clientv3.Client
	clusterName      string
	leaseWrp         leaseWrapper
	listenersChanMap sync.Map
	regWg            sync.WaitGroup
}

type leaseWrapper struct {
	rwMutex        sync.RWMutex
	wg             sync.WaitGroup
	leaseId        *clientv3.LeaseID
	isLeaseRunning bool
}

// Lookup Service Discovery
func (r *etcdRegistry) Lookup() ([]string, error) {
	resp, err := r.client.Get(context.Background(), constant.ETCDV3_REGISTRY_PREFIX+r.clusterName, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	addrs := make([]string, 0)
	for _, kv := range resp.Kvs {
		addrs = append(addrs, string(kv.Value))
	}

	err = r.Subscribe("", &etcdEventListener{})
	if err != nil {
		return nil, err
	}

	return addrs, nil
}

func (r *etcdRegistry) Register(addr *registry.Address) error {
	// Make Sure Address Not Nil and Port Not Equals Zero
	if res := utils.IsAddressValid(*addr); !res {
		return errors.New("the address to register is invalid")
	}
	// RegistryKey Format: etcdv3-seata-clusterName-ipAddress:port
	// RegistryValue Format: ipAddress:port
	key := utils.BuildRegistryKey(r.clusterName, addr)
	log.Infof("register key service: %s", key)
	_, err := r.client.Put(context.Background(), key, utils.BuildRegistryValue(addr), clientv3.WithLease(*r.leaseWrp.leaseId))
	return err
}

func (r *etcdRegistry) UnRegister(addr *registry.Address) error {
	key := utils.BuildRegistryKey(r.clusterName, addr)
	log.Infof("unregister key service: %s", key)
	_, err := r.client.Delete(context.Background(), key)
	return err
}

func (r *etcdRegistry) Subscribe(cluster string, listener registry.EventListener) error {
	resp, err := r.client.Get(context.Background(), constant.ETCDV3_REGISTRY_PREFIX+r.clusterName, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	wcCh := r.client.Watch(context.Background(), constant.ETCDV3_REGISTRY_PREFIX+r.clusterName, clientv3.WithPrefix(), clientv3.WithRev(resp.Header.Revision))
	stopChan := make(chan struct{})
	r.listenersChanMap.Store(r.clusterName, stopChan)
	r.regWg.Add(1)
	go r.watch(wcCh, listener, stopChan)
	return nil
}

func (r *etcdRegistry) watch(wcCh clientv3.WatchChan, listener registry.EventListener, stop chan struct{}) {
	defer r.regWg.Done()
LOOP:
	for {
		select {
		case resp := <-wcCh:
			services := make([]*registry.Service, 0)
			for _, event := range resp.Events {
				if event.Kv.Value != nil {
					addr := strings.Split(string(event.Kv.Value), ":")
					port, _ := strconv.ParseUint(addr[1], 10, 64)
					services = append(services, &registry.Service{
						EventType: uint32(event.Type),
						IP:        addr[0],
						Port:      port,
						Name:      string(event.Kv.Key),
					})
				} else {
					services = append(services, &registry.Service{
						EventType: uint32(event.Type),
						Name:      string(event.Kv.Key),
					})
				}

			}
			err := listener.OnEvent(services)
			if err != nil {
				log.Warnf("etcd listener error: %s", err.Error())
			}
		case <-stop:
			log.Info("etcd listener quit ...")
			break LOOP
		}
	}
}

func (r *etcdRegistry) UnSubscribe(cluster string, listener registry.EventListener) error {
	stopChanUnCast, ok := r.listenersChanMap.Load(constant.ETCDV3_REGISTRY_PREFIX + r.clusterName)
	if !ok {
		return errors.New("failed to unsubscribe, not matching key in the map")
	}
	stopChan, _ := stopChanUnCast.(chan struct{})
	stopChan <- struct{}{}
	r.listenersChanMap.Delete(constant.ETCDV3_REGISTRY_PREFIX + r.clusterName)
	return nil
}

// leaseKeeper Run in the Background to Renew Lease
func (r *etcdRegistry) leaseKeeper() {
	defer r.leaseWrp.wg.Done()

	r.leaseWrp.rwMutex.Lock()
	r.leaseWrp.isLeaseRunning = true
	r.leaseWrp.rwMutex.Unlock()

LOOP:
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
			break LOOP
		}
	}

	log.Info("lease keeper quit...")
}

// Stop wait for goroutines to stop
func (r *etcdRegistry) Stop() {
	r.listenersChanMap.Range(func(key, value interface{}) bool {
		stopChan, ok := value.(chan struct{})
		if !ok {
			log.Warnf("failed to attain stop channel")
			return true
		}

		stopChan <- struct{}{}
		r.listenersChanMap.Delete(key)
		return true
	})
	r.leaseWrp.rwMutex.Lock()
	// Make LeaseKeeper in the Goroutine Stop
	r.leaseWrp.isLeaseRunning = false
	r.leaseWrp.rwMutex.Unlock()
	// Wait for Goroutines Ended
	r.leaseWrp.wg.Wait()
	r.regWg.Wait()
	r.client = nil
}

func newETCDRegistry() (registry.Registry, error) {
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

type pair struct {
	key   interface{}
	value interface{}
}
