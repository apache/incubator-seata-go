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

type ClientConfig struct {
	ApplicationId		    string
	TransactionServiceGroup string
	GettyConfig             GettyConfig
	TMConfig                TMConfig
}

var clientConfig ClientConfig

func GetClientConfig() ClientConfig {
	return clientConfig
}

func GetDefaultClientConfig(applicationId string) ClientConfig {
	return ClientConfig{
		ApplicationId:           applicationId,
		TransactionServiceGroup: "127.0.0.1:8091",
		GettyConfig:             GetDefaultGettyConfig(),
		TMConfig:                GetDefaultTmConfig(),
	}
}


func InitConf(confFile string) error {
	var err error

	if confFile == "" {
		return errors.WithMessagef(err,fmt.Sprintf("application configure file name is nil"))
	}
	if path.Ext(confFile) != ".yml" {
		return errors.WithMessagef(err,fmt.Sprintf("application configure file name{%v} suffix must be .yml", confFile))
	}

	clientConfig = ClientConfig{}
	confFileStream, err := ioutil.ReadFile(confFile)
	if err != nil {
		return errors.WithMessagef(err,fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", confFile, err))
	}
	err = yaml.Unmarshal(confFileStream, &tmConfig)
	if err != nil {
		return errors.WithMessagef(err,fmt.Sprintf("yaml.Unmarshal() = error:%s", err))
	}

	(&clientConfig).GettyConfig.CheckValidity()

	return nil
}

func InitConfWithDefault(applicationId string) {
	clientConfig = GetDefaultClientConfig(applicationId)
	(&clientConfig).GettyConfig.CheckValidity()
	tmConfig = clientConfig.TMConfig
}