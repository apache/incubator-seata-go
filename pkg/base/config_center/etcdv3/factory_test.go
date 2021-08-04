package etcdv3

import (
	"context"
	"strings"
	"testing"
	"time"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
)

import (
	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v2"
)

var conf config.ConfigCenterConfig

func getConfig(t *testing.T) {
	confStr := `
type: etcdv3
etcdv3:
  endpoints: 127.0.0.1:52379
  config_key: test-config-key
`

	conf = config.ConfigCenterConfig{}
	err := defaults.Set(&conf)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte(confStr), &conf)
	assert.NoError(t, err)
}

func initConfigCenter(t *testing.T) (*config_center.DynamicConfigurationFactory, *clientv3.Client) {
	configCenter, err := newEtcdConfigCenter(&conf)
	assert.NoError(t, err)

	client, err := clientv3.New(clientv3.Config{Endpoints: []string{"127.0.0.1:52379"}})
	assert.NoError(t, err)

	return &configCenter, client
}

func TestEtcdConfigCenter_GetConfig(t *testing.T) {
	getConfig(t)
	configCenter, etcdClient := initConfigCenter(t)
	_, err := etcdClient.Put(context.Background(), conf.ETCDConfig.ConfigKey, "test-config")
	assert.NoError(t, err)

	res := (*configCenter).GetConfig(&conf)
	match := strings.Contains(res, "test-config")
	assert.Equal(t, true, match)

	err = (*configCenter).Stop()
	assert.NoError(t, err)
	err = etcdClient.Close()
	assert.NoError(t, err)
}

type mockListener struct {
	t      *testing.T
	newVal string
}

func (l *mockListener) Process(event *config_center.ConfigChangeEvent) {
	val := event.Value.([]byte)
	assert.Equal(l.t, conf.ETCDConfig.ConfigKey, event.Key)
	if string(val) == "test-config" {
		return
	}
	match := strings.Contains(string(val), "new-test-config")
	assert.Equal(l.t, true, match)
}

func TestEtcdConfigCenter_AddListener(t *testing.T) {
	getConfig(t)
	configCenter, etcdClient := initConfigCenter(t)
	_, err := etcdClient.Put(context.Background(), conf.ETCDConfig.ConfigKey, "test-config")
	assert.NoError(t, err)

	(*configCenter).AddListener(&conf, &mockListener{
		t:      t,
		newVal: "new-test-config",
	})

	_, err = etcdClient.Put(context.Background(), conf.ETCDConfig.ConfigKey, "new-test-config")
	assert.NoError(t, err)

	time.Sleep(time.Duration(1) * time.Second)

	err = (*configCenter).Stop()
	assert.NoError(t, err)
	err = etcdClient.Close()
	assert.NoError(t, err)
}

func TestEtcdConfigCenter_Stop(t *testing.T) {

}
