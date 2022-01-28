package etcdv3

import (
	"strings"
	"sync"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"

	clientv3 "go.etcd.io/etcd/client/v3"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
	"github.com/transaction-wg/seata-golang/pkg/base/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/extension"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

func init() {
	extension.SetConfigCenter(constant.Etcdv3Key, newEtcdConfigCenter)
}

type etcdConfigCenter struct {
	clitMutex sync.RWMutex
	wg        sync.WaitGroup
	client    *gxetcd.Client
}

func (c *etcdConfigCenter) GetConfig(conf *config.ConfigCenterConfig) string {
	// dynamic config's key default is "config-seata"
	configKey := conf.ETCDConfig.ConfigKey
	resp, err := c.client.Get(configKey)
	if err != nil {
		log.Errorf("failed to attain config from etcd server, %s", err.Error())
		return ""
	}
	return resp
}

func (c *etcdConfigCenter) AddListener(conf *config.ConfigCenterConfig, listener config_center.ConfigurationListener) {
	// Dynamic Config's Key Default is "config-seata"
	configKey := conf.ETCDConfig.ConfigKey
	wc, err := c.client.Watch(configKey)
	if err != nil {
		log.Errorf("watch config failed, %s", err.Error())
	}
	c.wg.Add(1)
	go c.handleEvents(wc, listener)
}

func (c *etcdConfigCenter) handleEvents(wc clientv3.WatchChan, listener config_center.ConfigurationListener) {
	defer c.wg.Done()

	for {
		select {
		case <-c.client.Done():
			log.Info("etcd config center listener quit ...")
			return
		case resp := <-wc:
			if resp.Events == nil {
				continue
			}
			for _, event := range resp.Events {
				listener.Process(&config_center.ConfigChangeEvent{
					Key:   string(event.Kv.Key),
					Value: event.Kv.Value,
				})
			}
		}
	}
}

func (c *etcdConfigCenter) Stop() error {
	c.client.Close()
	c.wg.Wait()
	c.client = nil
	return nil
}

func newEtcdConfigCenter(conf *config.ConfigCenterConfig) (config_center.DynamicConfigurationFactory, error) {
	etcdConfig := conf.ETCDConfig
	eps := strings.Split(etcdConfig.Endpoints, ",")
	client, err := gxetcd.NewClient(etcdConfig.Name, eps, etcdConfig.Timeout, etcdConfig.Heartbeats)
	if err != nil {
		return nil, err
	}

	return &etcdConfigCenter{
		clitMutex: sync.RWMutex{},
		wg:        sync.WaitGroup{},
		client:    client,
	}, nil
}
