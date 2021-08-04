package etcdv3

import (
	"context"

	"sync"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
	"github.com/transaction-wg/seata-golang/pkg/base/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/extension"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

import (
	clientv3 "go.etcd.io/etcd/client/v3"
)

func init() {
	extension.SetConfigCenter(constant.Etcdv3Key, newEtcdConfigCenter)
}

type etcdConfigCenter struct {
	clitMutex sync.RWMutex
	wg        sync.WaitGroup
	client    *clientv3.Client
	done      chan struct{}
}

func (c *etcdConfigCenter) GetConfig(conf *config.ConfigCenterConfig) string {
	// dynamic config's key default is "config-seata"
	configKey := conf.ETCDConfig.ConfigKey
	resp, err := c.client.Get(context.Background(), configKey)
	if err != nil {
		return ""
	}

	// Should Be Only One KV in the Etcdv3
	if len(resp.Kvs) != 1 {
		log.Warn("failed to attain config from etcd server, too many or too few config found")
		return ""
	}

	var res string
	for _, value := range resp.Kvs {
		res = value.String()
	}
	return res
}

func (c *etcdConfigCenter) AddListener(conf *config.ConfigCenterConfig, listener config_center.ConfigurationListener) {
	// Dynamic Config's Key Default is "config-seata"
	configKey := conf.ETCDConfig.ConfigKey
	wc := c.client.Watch(context.Background(), configKey)
	c.wg.Add(1)
	go c.handleEvents(wc, listener)
}

func (c *etcdConfigCenter) handleEvents(wc clientv3.WatchChan, listener config_center.ConfigurationListener) {
	defer c.wg.Done()

	for {
		select {
		case <-c.done:
			log.Warn("etcd connection already closed")
			return
		case e := <-wc:
			for _, event := range e.Events {
				listener.Process(&config_center.ConfigChangeEvent{
					Key:   string(event.Kv.Key),
					Value: event.Kv.Value,
				})
			}
		}
	}
}

func (c *etcdConfigCenter) Stop() error {
	c.done <- struct{}{}
	c.wg.Wait()
	close(c.done)
	err := c.client.Close()
	c.client = nil
	return err
}

func newEtcdConfigCenter(conf *config.ConfigCenterConfig) (config_center.DynamicConfigurationFactory, error) {
	etcdConfig := conf.ETCDConfig
	config, err := etcdConfig.ParseEtcdConfig(context.Background())
	if err != nil {
		return nil, err
	}
	client, err := clientv3.New(config)
	if err != nil {
		return nil, err
	}
	return &etcdConfigCenter{
		clitMutex: sync.RWMutex{},
		wg:        sync.WaitGroup{},
		client:    client,
		done:      make(chan struct{}, 1),
	}, nil
}
