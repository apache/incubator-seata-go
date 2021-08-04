package config

import (
	"context"
	"crypto/tls"
	"strings"
	"time"
)

import (
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

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
	ConfigKey            string        `default:"config-seata" yaml:"config_key" json:"config_key,omitempty"`
	Endpoints            string        `yaml:"endpoints" json:"endpoints,omitempty"`
	Username             string        `yaml:"username" json:"username,omitempty"`
	Password             string        `yaml:"password" json:"password,omitempty"`
	AutoSyncInterval     time.Duration `default:"0" yaml:"auto_sync_interval" json:"auto_sync_interval,omitempty"`
	DialTimeout          time.Duration `default:"5s" yaml:"dial_timeout" json:"dial_timeout,omitempty"`
	DialKeepAliveTime    time.Duration `default:"1s" yaml:"dial_keep_alive_time" json:"dial_keep_alive_time,omitempty"`
	DialKeepAliveTimeout time.Duration `default:"3s" yaml:"dial_keep_alive_timeout" json:"dial_keep_alive_timeout,omitempty"`
	MaxCallSendMsgSize   int           `default:"0" yaml:"max_call_msg_size" json:"max_call_msg_size,omitempty"`
	MaxCallRecvMsgSize   int           `default:"0" yaml:"max_recv_msg_size" json:"max_recv_msg_size,omitempty"`
	TLSConfig            *TLSConfig    `yaml:"tls" json:"tls,omitempty"`
}

type TLSConfig struct {
	CertFile      string `yaml:"cert_file" json:"cert_file,omitempty"`
	KeyFile       string `yaml:"key_file" json:"key_file,omitempty"`
	TrustedCAFile string `yaml:"trusted_ca_file" json:"trusted_ca_file,omitempty"`
}

func (c EtcdConfigCenter) ParseEtcdConfig(ctx context.Context) (clientv3.Config, error) {
	// Endpoints eg: "127.0.0.1:11451,127.0.0.1:11452"
	endpoints := strings.Split(c.Endpoints, ",")

	var tlsConfig *tls.Config
	if c.TLSConfig != nil {
		tcpInfo := transport.TLSInfo{
			CertFile:      c.TLSConfig.CertFile,
			KeyFile:       c.TLSConfig.KeyFile,
			TrustedCAFile: c.TLSConfig.TrustedCAFile,
		}

		var err error
		tlsConfig, err = tcpInfo.ClientConfig()
		if err != nil {
			return clientv3.Config{}, err
		}
	}

	return clientv3.Config{
		Endpoints:            endpoints,
		AutoSyncInterval:     c.AutoSyncInterval,
		DialTimeout:          c.DialTimeout,
		DialKeepAliveTime:    c.DialKeepAliveTime,
		DialKeepAliveTimeout: c.DialKeepAliveTimeout,
		MaxCallSendMsgSize:   c.MaxCallSendMsgSize,
		MaxCallRecvMsgSize:   c.MaxCallRecvMsgSize,
		Username:             c.Username,
		Password:             c.Password,
		RejectOldCluster:     false,
		DialOptions:          []grpc.DialOption{grpc.WithBlock()}, // Disable Async
		PermitWithoutStream:  true,
		TLS:                  tlsConfig,
		Context:              ctx,
	}, nil
}
