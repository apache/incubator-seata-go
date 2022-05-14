package config

import (
	"time"
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

	RegistryConfig     RegistryConfig     `yaml:"registry_config" json:"registry_config,omitempty"` //注册中心配置信息
	ConfigCenterConfig ConfigCenterConfig `yaml:"config_center" json:"config_center,omitempty"`     //配置中心配置信息
}

func GetClientConfig() *ClientConfig {
	// todo mock data
	//return clientConfig
	return &ClientConfig{
		GettyConfig: GetDefaultGettyConfig(),
	}
}

func GetTMConfig() TMConfig {
	return clientConfig.TMConfig
}

func GetDefaultClientConfig(applicationID string) *ClientConfig {
	return &ClientConfig{
		ApplicationID:                applicationID,
		TransactionServiceGroup:      "127.0.0.1:8091",
		EnableClientBatchSendRequest: false,
		SeataVersion:                 "1.1.0",
		GettyConfig:                  GetDefaultGettyConfig(),
		TMConfig:                     GetDefaultTmConfig(),
	}
}
