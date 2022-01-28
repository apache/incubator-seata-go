package config

import (
	"io"
	"io/ioutil"
	"os"
	"time"
)

import (
	"github.com/creasty/defaults"

	"github.com/imdario/mergo"

	"gopkg.in/yaml.v2"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/config"
	"github.com/transaction-wg/seata-golang/pkg/base/config_center"
	"github.com/transaction-wg/seata-golang/pkg/base/extension"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/parser"
)

var clientConfig *ClientConfig

type ClientConfig struct {
	ApplicationID                string      `yaml:"application_id" json:"application_id,omitempty"`
	TransactionServiceGroup      string      `yaml:"transaction_service_group" json:"transaction_service_group,omitempty"`
	EnableClientBatchSendRequest bool        `yaml:"enable-rpc_client-batch-send-request" json:"enable-rpc_client-batch-send-request,omitempty"`
	SeataVersion                 string      `yaml:"seata_version" json:"seata_version,omitempty"`
	GettyConfig                  GettyConfig `yaml:"getty" json:"getty,omitempty"`

	TMConfig TMConfig `yaml:"tm" json:"tm,omitempty"`

	ATConfig struct {
		DSN                 string        `yaml:"dsn" json:"dsn,omitempty"`
		ReportRetryCount    int           `default:"5" yaml:"report_retry_count" json:"report_retry_count,omitempty"`
		ReportSuccessEnable bool          `default:"false" yaml:"report_success_enable" json:"report_success_enable,omitempty"`
		LockRetryInterval   time.Duration `default:"10ms" yaml:"lock_retry_interval" json:"lock_retry_interval,omitempty"`
		LockRetryTimes      int           `default:"30" yaml:"lock_retry_times" json:"lock_retry_times,omitempty"`
	} `yaml:"at" json:"at,omitempty"`

	RegistryConfig     config.RegistryConfig     `yaml:"registry_config" json:"registry_config,omitempty"` //注册中心配置信息
	ConfigCenterConfig config.ConfigCenterConfig `yaml:"config_center" json:"config_center,omitempty"`     //配置中心配置信息
}

func GetClientConfig() *ClientConfig {
	return clientConfig
}

func GetTMConfig() TMConfig {
	return clientConfig.TMConfig
}

func GetDefaultClientConfig(applicationID string) ClientConfig {
	return ClientConfig{
		ApplicationID:                applicationID,
		TransactionServiceGroup:      "127.0.0.1:8091",
		EnableClientBatchSendRequest: false,
		SeataVersion:                 "1.1.0",
		GettyConfig:                  GetDefaultGettyConfig(),
		TMConfig:                     GetDefaultTmConfig(),
	}
}

// Parse parses an input configuration yaml document into a Configuration struct
//
// Environment variables may be used to override configuration parameters other than version,
// following the scheme below:
// Configuration.Abc may be replaced by the value of SEATA_ABC,
// Configuration.Abc.Xyz may be replaced by the value of SEATA_ABC_XYZ, and so forth
func parse(rd io.Reader) (*ClientConfig, error) {
	in, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	p := parser.NewParser("seata")

	config := new(ClientConfig)
	err = p.Parse(in, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func loadConfigCenterConfig(config *ClientConfig) {
	if config.ConfigCenterConfig.Mode == "" {
		return
	}

	cc, err := extension.GetConfigCenter(config.ConfigCenterConfig.Mode, &config.ConfigCenterConfig)
	if err != nil {
		log.Error("ConfigCenter can not connect success.Error message is %s", err.Error())
	}
	remoteConfig := config_center.LoadConfigCenterConfig(cc, &config.ConfigCenterConfig, &ClientConfigListener{})
	updateConf(clientConfig, remoteConfig)
}

type ClientConfigListener struct {
}

func (ClientConfigListener) Process(event *config_center.ConfigChangeEvent) {
	conf := GetClientConfig()
	updateConf(conf, event.Value.(string))
}

func updateConf(config *ClientConfig, remoteConfig string) {
	newConf := &ClientConfig{}
	err := defaults.Set(newConf)
	if err != nil {
		log.Errorf("config set default value failed, %s", err.Error())
	}
	confByte := []byte(remoteConfig)
	yaml.Unmarshal(confByte, newConf)
	if err := mergo.Merge(config, newConf, mergo.WithOverride); err != nil {
		log.Error("merge config fail %s ", err.Error())
	}
}

// InitConf init ClientConfig from a file path
func InitConf(confFile string) error {
	fp, err := os.Open(confFile)
	if err != nil {
		log.Fatalf("open configuration file fail, %v", err)
	}

	defer fp.Close()

	conf, err := parse(fp)
	if err != nil {
		log.Fatalf("error parsing %s: %v", confFile, err)
	}

	loadConfigCenterConfig(conf)
	config.InitRegistryConfig(&conf.RegistryConfig)
	clientConfig = conf

	return nil
}
