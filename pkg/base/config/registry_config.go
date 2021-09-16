package config

import "time"

var config *RegistryConfig

// RegistryConfig registry config
type RegistryConfig struct {
	Mode        string      `yaml:"type" json:"type,omitempty"` //类型
	NacosConfig NacosConfig `yaml:"nacos" json:"nacos,omitempty"`
	EtcdConfig  EtcdConfig  `yaml:"etcdv3" json:"etcdv3"`
}

// NacosConfig nacos config
type NacosConfig struct {
	Application string `yaml:"application" json:"application,omitempty"`
	ServerAddr  string `yaml:"server_addr" json:"server_addr,omitempty"`
	Group       string `default:"SEATA_GROUP" yaml:"group" json:"group,omitempty"`
	Namespace   string `yaml:"namespace" json:"namespace,omitempty"`
	Cluster     string `yaml:"cluster" json:"cluster,omitempty"`
	UserName    string `yaml:"username" json:"username,omitempty"`
	Password    string `yaml:"password" json:"password,omitempty"`
}

// InitRegistryConfig init registry config
func InitRegistryConfig(registryConfig *RegistryConfig) {
	config = registryConfig
}

// GetRegistryConfig get registry config
func GetRegistryConfig() *RegistryConfig {
	return config
}

type EtcdConfig struct {
	ClusterName string        `default:"default" yaml:"cluster_name" json:"cluster_name,omitempty"`
	Endpoints   string        `yaml:"endpoints" json:"endpoints,omitempty"`
	Heartbeats  int           `yaml:"heartbeats" json:"heartbeats"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
}
