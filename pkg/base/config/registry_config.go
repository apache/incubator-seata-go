package config

import "time"

var config *RegistryConfig

// RegistryConfig registry config
type RegistryConfig struct {
	Mode        string      `yaml:"type" json:"type,omitempty"` //类型
	NacosConfig NacosConfig `yaml:"nacos" json:"nacos,omitempty"`
	ETCDConfig  EtcdConfig  `yaml:"etcdv3" json:"etcdv3,omitempty"`
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

type EtcdConfig struct {
	Endpoints            string        `yaml:"endpoints" json:"endpoints,omitempty"`
	Username             string        `yaml:"username" json:"username,omitempty"`
	Password             string        `yaml:"password" json:"password,omitempty"`
	AutoSyncInterval     time.Duration `default:"0" yaml:"auto_sync_interval" json:"auto_sync_interval,omitempty"`
	DialTimeout          time.Duration `default:"5s" yaml:"dial_timeout" json:"dial_timeout,omitempty"`
	DialKeepAliveTime    time.Duration `default:"1s" yaml:"dial_keep_alive_time" json:"dial_keep_alive_time,omitempty"`
	DialKeepAliveTimeout time.Duration `default:"3s" yaml:"dial_keep_alive_timeout" json:"dial_keep_alive_timeout,omitempty"`
	MaxCallSendMsgSize   int           `default:"0" yaml:"max_call_msg_size" json:"max_call_msg_size,omitempty"`
	MaxCallRecvMsgSize   int           `default:"0" yaml:"max_recv_msg_size" json:"max_recv_msg_size,omitempty"`
	ClusterName          string        `default:"default" yaml:"cluster_name" json:"cluster_name,omitempty"`
	TLSConfig            *TLSConfig    `yaml:"tls" json:"tls,omitempty"`
}

type TLSConfig struct {
	CertFile      string `yaml:"cert_file" json:"cert_file,omitempty"`
	KeyFile       string `yaml:"key_file" json:"key_file,omitempty"`
	TrustedCAFile string `yaml:"trusted_ca_file" json:"trusted_ca_file,omitempty"`
}

// InitRegistryConfig init registry config
func InitRegistryConfig(registryConfig *RegistryConfig) {
	config = registryConfig
}

func GetRegistryConfig() *RegistryConfig {
	return config
}
