package etcdv3

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	perrors "github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	utils "github.com/transaction-wg/seata-golang/pkg/util/etcdv3"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

func init() {
	extension.SetRegistry(constant.Etcdv3Key, newETCDRegistry)
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

type etcdRegistry struct {
	clLock sync.Mutex
	regWg  sync.WaitGroup

	client      *gxetcd.Client
	clusterName string

	leaseWrp leaseWrapper
}

type leaseWrapper struct {
	wg      sync.WaitGroup
	leaseId *clientv3.LeaseID
}

// Lookup Service Discovery
func (r *etcdRegistry) Lookup() ([]string, error) {
	kList, vList, err := r.client.GetChildren(constant.Etcdv3RegistryPrefix)
	if err != nil {
		return nil, err
	}

	kvLength := len(kList)
	for i := 0; i < kvLength; i++ {
		err = r.Subscribe(&etcdEventListener{})
		if err != nil {
			return nil, err
		}
	}

	return vList, nil
}

func (r *etcdRegistry) Register(addr *registry.Address) error {
	// Make Sure Address Not Nil and Port Not Equals Zero
	if res := utils.IsAddressValid(*addr); !res {
		return perrors.New("the address to register is invalid")
	}
	// RegistryKey Format: etcdv3-seata-clusterName-ipAddress:port
	// RegistryValue Format: ipAddress:port
	return r.client.Put(utils.BuildRegistryKey(r.clusterName, addr), utils.BuildRegistryValue(addr))
}

func (r *etcdRegistry) UnRegister(addr *registry.Address) error {
	return perrors.New("DoUnregister is not support in etcdV3Registry")
}

func (r *etcdRegistry) Subscribe(listener registry.EventListener) error {
	wcCh, err := r.client.WatchWithOption(constant.Etcdv3RegistryPrefix+r.clusterName, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	r.regWg.Add(1)
	go r.watch(wcCh, listener)
	return nil
}

func (r *etcdRegistry) watch(wcCh clientv3.WatchChan, listener registry.EventListener) {
	defer r.regWg.Done()
LOOP:
	for {
		select {
		case <-r.client.Done():
			log.Info("watch goroutine quit...")
			break LOOP
		case resp := <-wcCh:
			services := make([]*registry.Service, len(resp.Events))
			if resp.Events == nil {
				continue
			}
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
		}
	}
}

func (r *etcdRegistry) UnSubscribe(listener registry.EventListener) error {
	return perrors.New("UnSubscribe is not support in etcdV3Registry")
}

// leaseKeeper Run in the Background to Renew Lease
func (r *etcdRegistry) leaseKeeper() {
	defer r.leaseWrp.wg.Done()

	for {
		select {
		case <-r.client.Done():
			log.Info("leaseKeeper goroutine quit...")
			break
		default:
			ttl, err := r.client.GetRawClient().TimeToLive(r.client.GetCtx(), *r.leaseWrp.leaseId)
			if err != nil {
				log.Warnf("failed to attain ttl info, %s", err.Error())
			}

			if ttl == nil {
				log.Warn("failed to renew ttl, ttl info in resp is nil")
				continue
			}

			if ttl.TTL <= constant.Etcdv3LeaseTtlCritical {
				_, err := r.client.GetRawClient().KeepAliveOnce(context.Background(), *r.leaseWrp.leaseId)
				if err != nil {
					log.Warnf("failed to renew ttl, %s", err.Error())
				}
			}

			time.Sleep(time.Duration(constant.Etcdv3LeaseRenewInterval) * time.Second)
		}
	}

}

// Stop wait for goroutines to stop
func (r *etcdRegistry) Stop() {
	r.client.Close()
	// Wait for Goroutine Ended
	r.leaseWrp.wg.Wait()
	r.regWg.Wait()
}

func newETCDRegistry() (registry.Registry, error) {
	rc := config.GetRegistryConfig().EtcdConfig
	name := rc.ClusterName
	if name == "" {
		name = "seata-golang-etcdv3"
	}

	c := gxetcd.NewConfigClient(
		gxetcd.WithName(name),
		gxetcd.WithEndpoints(rc.Endpoints),
		gxetcd.WithTimeout(rc.Timeout),
	)

	lease := clientv3.NewLease(c.GetRawClient())
	resp, err := lease.Grant(context.Background(), constant.Etcdv3LeaseTtl)
	if err != nil {
		return nil, err
	}

	r := &etcdRegistry{
		client:      c,
		clusterName: name,
		leaseWrp: leaseWrapper{
			wg:      sync.WaitGroup{},
			leaseId: &resp.ID,
		},
	}

	go r.leaseKeeper()

	return r, nil
}
