package config

import "time"

// ConfigCenterConfig config center config
type ConfigCenterConfig struct {
	Mode        string            `yaml:"type" json:"type,omitempty"` //类型
	NacosConfig NacosConfigCenter `yaml:"nacos" json:"nacos,omitempty"`
	ETCDConfig  EtcdConfigCenter  `yaml:"etcdv3" json:"etcdv3,omitempty"`
}

// NacosConfigCenter nacos config center
type NacosConfigCenter struct {
	ServerAddr string `yaml:"server_addr" json:"server_addr,omitempty"`
	Group      string `default:"SEATA_GROUP" yaml:"group" json:"group,omitempty"`
	Namespace  string `yaml:"namespace" json:"namespace,omitempty"`
	Cluster    string `yaml:"cluster" json:"cluster,omitempty"`
	UserName   string `yaml:"username" json:"username,omitempty"`
	Password   string `yaml:"password" json:"password,omitempty"`
	DataID     string `default:"seata" yaml:"data_id" json:"data_id,omitempty"`
}

type EtcdConfigCenter struct {
	Name       string        `default:"seata-config-center" yaml:"name" json:"name"`
	ConfigKey  string        `default:"config-seata" yaml:"config_key" json:"config_key,omitempty"`
	Endpoints  string        `yaml:"endpoints" json:"endpoints,omitempty"`
	Heartbeats int           `yaml:"heartbeats" json:"heartbeats"`
	Timeout    time.Duration `yaml:"timeout" json:"timeout"`
}
