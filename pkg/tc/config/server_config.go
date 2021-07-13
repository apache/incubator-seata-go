package config

import (
	"fmt"
	"github.com/go-xorm/xorm"
	"io"
	"io/ioutil"
	"os"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
	"github.com/transaction-wg/seata-golang/pkg/base/extension"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/parser"
)

var serverConfig *ServerConfig

type ServerConfig struct {
	Port                             string `default:"8091" yaml:"port" json:"port,omitempty"`
	MaxRollbackRetryTimeout          int64  `default:"-1" yaml:"max_rollback_retry_timeout" json:"max_rollback_retry_timeout,omitempty"`
	RollbackRetryTimeoutUnlockEnable bool   `default:"false" yaml:"rollback_retry_timeout_unlock_enable" json:"rollback_retry_timeout_unlock_enable,omitempty"`
	MaxCommitRetryTimeout            int64  `default:"-1" yaml:"max_commit_retry_timeout" json:"max_commit_retry_timeout,omitempty"`

	TimeoutRetryPeriod         time.Duration `default:"1s" yaml:"timeout_retry_period" json:"timeout_retry_period,omitempty"`
	RollingBackRetryPeriod     time.Duration `default:"1s" yaml:"rolling_back_retry_period" json:"rolling_back_retry_period,omitempty"`
	CommittingRetryPeriod      time.Duration `default:"1s" yaml:"committing_retry_period" json:"committing_retry_period,omitempty"`
	AsyncCommittingRetryPeriod time.Duration `default:"1s" yaml:"async_committing_retry_period" json:"async_committing_retry_period,omitempty"`
	LogDeletePeriod            time.Duration `default:"24h" yaml:"log_delete_period" json:"log_delete_period,omitempty"`

	GettyConfig struct {
		SessionTimeout time.Duration `default:"60s" yaml:"session_timeout" json:"session_timeout,omitempty"`

		GettySessionParam config.GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
	} `required:"true" yaml:"getty_config" json:"getty_config,omitempty"`

	UndoConfig struct {
		LogSaveDays int16 `default:"7" yaml:"log_save_days" json:"log_save_days,omitempty"`
	} `required:"true" yaml:"undo_config" json:"undo_config,omitempty"`

	StoreConfig        StoreConfig               `required:"true" yaml:"store_config" json:"store_config,omitempty"`
	RegistryConfig     config.RegistryConfig     `yaml:"registry_config" json:"registry_config,omitempty"` //注册中心配置信息
	ConfigCenterConfig config.ConfigCenterConfig `yaml:"config_center" json:"config_center,omitempty"`     //配置中心配置信息
}

func GetServerConfig() *ServerConfig {
	return serverConfig
}

func GetStoreConfig() StoreConfig {
	return serverConfig.StoreConfig
}

// Parse parses an input configuration yaml document into a ServerConfig struct
//
// Environment variables may be used to override configuration parameters other than version,
// following the scheme below:
// ServerConfig.Abc may be replaced by the value of SEATA_ABC,
// ServerConfig.Abc.Xyz may be replaced by the value of SEATA_ABC_XYZ, and so forth
func parse(rd io.Reader) (*ServerConfig, error) {
	in, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	p := parser.NewParser("seata")

	config := new(ServerConfig)
	err = p.Parse(in, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func loadConfigCenterConfig(config *ServerConfig) {
	if config.ConfigCenterConfig.Mode == "" {
		return
	}
	cc, err := extension.GetConfigCenter(config.ConfigCenterConfig.Mode, &config.ConfigCenterConfig)
	if err != nil {
		log.Error("ConfigCenter can not connect success.Error message is %s", err.Error())
	}
	remoteConfig := config_center.LoadConfigCenterConfig(cc, &config.ConfigCenterConfig, &ServerConfigListener{})
	updateConf(config, remoteConfig)
}

type ServerConfigListener struct {

}

func (ServerConfigListener) Process(event *config_center.ConfigChangeEvent) {
	serverConfig := GetServerConfig()
	updateConf(serverConfig, event.Value.(string))
}

func updateConf(config *ServerConfig, remoteConfig string) {
	newConf := &ServerConfig{}
	confByte := []byte(remoteConfig)
	yaml.Unmarshal(confByte, newConf)
	if err := mergo.Merge(config, newConf, mergo.WithOverride); err != nil {
		log.Error("merge config fail %s ", err.Error())
	}
}

func InitConf(configPath string) (*ServerConfig, error) {
	var configFilePath string

	if configPath != "" {
		configFilePath = configPath
	} else if os.Getenv("SEATA_CONFIGURATION_PATH") != "" {
		configFilePath = os.Getenv("SEATA_CONFIGURATION_PATH")
	}

	if configFilePath == "" {
		return nil, fmt.Errorf("configuration path unspecified")
	}

	fp, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}

	defer fp.Close()

	conf, err := parse(fp)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", configFilePath, err)
	}

	if conf.GettyConfig.SessionTimeout >= time.Duration(getty.MaxWheelTimeSpan) {
		return nil, errors.Errorf("session_timeout %s should be less than %s",
			serverConfig.GettyConfig.SessionTimeout, time.Duration(getty.MaxWheelTimeSpan))
	}

	loadConfigCenterConfig(conf)
	config.InitRegistryConfig(&conf.RegistryConfig)
	serverConfig = conf

	if conf.StoreConfig.StoreMode == "db" && conf.StoreConfig.DBStoreConfig.DSN != "" {
		engine, err := xorm.NewEngine("mysql", conf.StoreConfig.DBStoreConfig.DSN)
		if err != nil {
			panic(err)
		}
		conf.StoreConfig.DBStoreConfig.Engine = engine
	}

	return serverConfig, nil
}
