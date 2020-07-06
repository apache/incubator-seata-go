package config

import (
	"fmt"
	"io/ioutil"
	"path"
)

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/dk-lockdown/seata-golang/client/at/sql/schema/cache"
)

type ClientConfig struct {
	ApplicationId           string      `yaml:"application_id" json:"application_id,omitempty"`
	TransactionServiceGroup string      `yaml:"transaction_service_group" json:"transaction_service_group,omitempty"`
	SeataVersion            string      `yaml:"seata_version" json:"seata_version,omitempty"`
	GettyConfig             GettyConfig `yaml:"getty" json:"getty,omitempty"`
	TMConfig                TMConfig    `yaml:"tm" json:"tm,omitempty"`
	ATConfig                ATConfig    `yaml:"at" json:"at,omitempty"`
}

var clientConfig ClientConfig

func GetClientConfig() ClientConfig {
	return clientConfig
}

func GetTMConfig() TMConfig {
	return clientConfig.TMConfig
}

func GetATConfig() ATConfig {
	return clientConfig.ATConfig
}

func GetDefaultClientConfig(applicationId string) ClientConfig {
	return ClientConfig{
		ApplicationId:           applicationId,
		SeataVersion:            "1.1.0",
		TransactionServiceGroup: "127.0.0.1:8091",
		GettyConfig:             GetDefaultGettyConfig(),
		TMConfig:                GetDefaultTmConfig(),
	}
}

func InitConf(confFile string) error {
	var err error

	if confFile == "" {
		return errors.WithMessagef(err, fmt.Sprintf("application configure file name is nil"))
	}
	if path.Ext(confFile) != ".yml" {
		return errors.WithMessagef(err, fmt.Sprintf("application configure file name{%v} suffix must be .yml", confFile))
	}

	clientConfig = ClientConfig{}
	confFileStream, err := ioutil.ReadFile(confFile)
	if err != nil {
		return errors.WithMessagef(err, fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", confFile, err))
	}
	err = yaml.Unmarshal(confFileStream, &clientConfig)
	if err != nil {
		return errors.WithMessagef(err, fmt.Sprintf("yaml.Unmarshal() = error:%s", err))
	}

	(&clientConfig).GettyConfig.CheckValidity()
	(&clientConfig).ATConfig.CheckValidity()

	if clientConfig.ATConfig.DSN != "" {
		cache.SetTableMetaCache(cache.NewMysqlTableMetaCache(clientConfig.ATConfig.DSN))
	}
	return nil
}

func InitConfWithDefault(applicationId string) {
	clientConfig = GetDefaultClientConfig(applicationId)
	(&clientConfig).GettyConfig.CheckValidity()
}
