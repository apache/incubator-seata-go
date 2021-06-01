package config

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"
)

import (
	"github.com/go-xorm/xorm"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)
import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

var (
	conf ServerConfig
)

func GetServerConfig() ServerConfig {
	return conf
}

func GetStoreConfig() StoreConfig {
	return conf.StoreConfig
}
func GetRegistryConfig() RegistryConfig {
	return conf.RegistryConfig
}
func GetConfigCenterConfig() ConfigCenterConfig {
	return conf.ConfigCenterConfig
}

type ServerConfig struct {
	Host                             string `yaml:"host" default:"127.0.0.1" json:"host,omitempty"`
	Port                             string `default:"8091" yaml:"port" json:"port,omitempty"`
	MaxRollbackRetryTimeout          int64  `default:"-1" yaml:"max_rollback_retry_timeout" json:"max_rollback_retry_timeout,omitempty"`
	RollbackRetryTimeoutUnlockEnable bool   `default:"false" yaml:"rollback_retry_timeout_unlock_enable" json:"rollback_retry_timeout_unlock_enable,omitempty"`
	MaxCommitRetryTimeout            int64  `default:"-1" yaml:"max_commit_retry_timeout" json:"max_commit_retry_timeout,omitempty"`
	Timeout_Retry_Period             string `default:"1s" yaml:"timeout_retry_period" json:"timeout_retry_period,omitempty"`
	TimeoutRetryPeriod               time.Duration
	Rollbacking_Retry_Period         string `default:"1s" yaml:"rollbacking_retry_period" json:"rollbacking_retry_period,omitempty"`
	RollbackingRetryPeriod           time.Duration
	Committing_Retry_Period          string `default:"1s" yaml:"committing_retry_period" json:"committing_retry_period,omitempty"`
	CommittingRetryPeriod            time.Duration
	Async_Committing_Retry_Period    string `default:"1s" yaml:"async_committing_retry_period" json:"async_committing_retry_period,omitempty"`
	AsyncCommittingRetryPeriod       time.Duration
	Log_Delete_Period                string `default:"24h" yaml:"log_delete_period" json:"log_delete_period,omitempty"`
	LogDeletePeriod                  time.Duration
	GettyConfig                      GettyConfig        `required:"true" yaml:"getty_config" json:"getty_config,omitempty"`
	UndoConfig                       UndoConfig         `required:"true" yaml:"undo_config" json:"undo_config,omitempty"`
	StoreConfig                      StoreConfig        `required:"true" yaml:"store_config" json:"store_config,omitempty"`
	RegistryConfig                   RegistryConfig     `yaml:"registry_config" json:"registry_config,omitempty"` //注册中心配置信息
	ConfigCenterConfig               ConfigCenterConfig `yaml:"config_center" json:"config_center,omitempty"`     //配置中心配置信息
}

func (c *ServerConfig) CheckValidity() error {
	var err error
	if conf.TimeoutRetryPeriod, err = time.ParseDuration(conf.Timeout_Retry_Period); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(Timeout_Retry_Period{%#v})", conf.Timeout_Retry_Period)
	}

	if conf.RollbackingRetryPeriod, err = time.ParseDuration(conf.Rollbacking_Retry_Period); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(Rollbacking_Retry_Period{%#v})", conf.Rollbacking_Retry_Period)
	}

	if conf.CommittingRetryPeriod, err = time.ParseDuration(conf.Committing_Retry_Period); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(Committing_Retry_Period{%#v})", conf.Committing_Retry_Period)
	}

	if conf.AsyncCommittingRetryPeriod, err = time.ParseDuration(conf.Async_Committing_Retry_Period); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(Async_Committing_Retry_Period{%#v})", conf.Async_Committing_Retry_Period)
	}

	if conf.LogDeletePeriod, err = time.ParseDuration(conf.Log_Delete_Period); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(Log_Delete_Period{%#v})", conf.Log_Delete_Period)
	}

	return errors.WithStack(c.GettyConfig.CheckValidity())
}

func InitConf(confFile string) error {
	var err error

	if confFile == "" {
		return errors.WithMessagef(err, fmt.Sprintf("application configure file name is nil"))
	}
	if path.Ext(confFile) != ".yml" {
		return errors.WithMessagef(err, fmt.Sprintf("application configure file name{%v} suffix must be .yml", confFile))
	}

	conf = ServerConfig{}
	confFileStream, err := ioutil.ReadFile(confFile)
	if err != nil {
		return errors.WithMessagef(err, fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", confFile, err))
	}
	err = yaml.Unmarshal(confFileStream, &conf)
	if err != nil {
		return errors.WithMessagef(err, fmt.Sprintf("yaml.Unmarshal() = error:%s", err))
	}
	//这里加载配置中心配置和本地配置做合并
	loadConfigCenterConfig(&conf)
	//监听远程配置情况，发生变更进行配置修改
	addLisenter()
	(&conf).CheckValidity()
	if conf.StoreConfig.StoreMode == "db" && conf.StoreConfig.DBStoreConfig.DSN != "" {
		engine, err := xorm.NewEngine("mysql", conf.StoreConfig.DBStoreConfig.DSN)
		if err != nil {
			panic(err)
		}
		conf.StoreConfig.DBStoreConfig.Engine = engine
	}

	return nil
}

type ConfigLisnter struct {
}

func (ConfigLisnter) Process(event *config_center.ConfigChangeEvent) {
	//更新conf
	conf := GetServerConfig()
	updateConf(&conf, event.Value.(string))
}

func addLisenter() {
	if conf.ConfigCenterConfig.Mode == "" {
		return
	}
	configCenterConfig := conf.ConfigCenterConfig
	cc, _ := extension.GetConfigCenter(configCenterConfig.Mode)
	listener := &ConfigLisnter{}
	cc.AddListener(listener)
}

func loadConfigCenterConfig(conf *ServerConfig) {
	if conf.ConfigCenterConfig.Mode == "" {
		return
	}
	configCenterConfig := conf.ConfigCenterConfig
	cc, err := extension.GetConfigCenter(configCenterConfig.Mode)
	if err != nil {
		log.Error("ConfigCenter can not connect success.Error message is %s", err.Error())
	}
	confStr := cc.GetConfig()
	updateConf(conf, confStr)
}

func updateConf(config *ServerConfig, confStr string) {
	newConf := &ServerConfig{}
	confByte := []byte(confStr)
	yaml.Unmarshal(confByte, newConf)
	//合并配置中心的配置和本地文件的配置，形成新的配置
	if err := mergo.Merge(config, newConf, mergo.WithOverride); err != nil {
		log.Error("merge config fail %s ", err.Error())
	}
}
