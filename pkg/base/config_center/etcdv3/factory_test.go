package etcdv3

import (
	"strings"
	"testing"
	"time"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
)

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

var conf config.ConfigCenterConfig

func getConfig(t *testing.T) {
	confStr := `
type: etcdv3
etcdv3:
  name: seata-config-center 
  endpoints: 127.0.0.1:52379
  config_key: test-config-key
`

	conf = config.ConfigCenterConfig{}
	err := yaml.Unmarshal([]byte(confStr), &conf)
	assert.NoError(t, err)
}

func initConfigCenter() (config_center.DynamicConfigurationFactory, error) {
	return newEtcdConfigCenter(&conf)
}

func TestEtcdConfigCenter_GetConfig(t *testing.T) {
	getConfig(t)
	cc, err := initConfigCenter()
	assert.NoError(t, err)

	err = cc.(*etcdConfigCenter).client.Put(conf.ETCDConfig.ConfigKey, "test-config")
	assert.NoError(t, err)

	res := cc.GetConfig(&conf)
	t.Logf("%v", res)
	match := strings.Contains(res, "test-config")
	assert.Equal(t, true, match)

	err = cc.Stop()
	assert.NoError(t, err)
}

type mockListener struct {
	t      *testing.T
	newVal string
}

func (l *mockListener) Process(event *config_center.ConfigChangeEvent) {
	val := event.Value.([]byte)
	l.t.Logf("%s", val)
	assert.Equal(l.t, conf.ETCDConfig.ConfigKey, event.Key)
	if string(val) == "test-config" {
		return
	}
	match := strings.Contains(string(val), "new-test-config")
	assert.Equal(l.t, true, match)
}

func TestEtcdConfigCenter_AddListener(t *testing.T) {
	getConfig(t)
	cc, err := initConfigCenter()
	assert.NoError(t, err)

	err = cc.(*etcdConfigCenter).client.Put(conf.ETCDConfig.ConfigKey, "test-config")
	assert.NoError(t, err)

	cc.AddListener(&conf, &mockListener{
		t:      t,
		newVal: "new-test-config",
	})

	err = cc.(*etcdConfigCenter).client.Put(conf.ETCDConfig.ConfigKey, "new-test-config")
	assert.NoError(t, err)

	time.Sleep(time.Duration(1) * time.Second)

	err = cc.Stop()
	assert.NoError(t, err)
}
