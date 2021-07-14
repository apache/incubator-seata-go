package config

import (
	config2 "github.com/transaction-wg/seata-golang/pkg/base/config"
	"time"
)

// GettyConfig
//Config holds supported types by the multiconfig package
type GettyConfig struct {
	ReconnectInterval int `default:"0" yaml:"reconnect_interval" json:"reconnect_interval,omitempty"`
	// getty_session pool
	ConnectionNum int `default:"16" yaml:"connection_number" json:"connection_number,omitempty"`

	// heartbeat
	HeartbeatPeriod time.Duration `default:"15s" yaml:"heartbeat_period" json:"heartbeat_period,omitempty"`

	// getty_session tcp parameters
	GettySessionParam config2.GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
}


// GetDefaultGettyConfig ...
func GetDefaultGettyConfig() GettyConfig {
	return GettyConfig{
		ReconnectInterval: 0,
		ConnectionNum:     1,
		HeartbeatPeriod:      10 * time.Second,
		GettySessionParam: config2.GettySessionParam{
			CompressEncoding: false,
			TCPNoDelay:       true,
			TCPKeepAlive:     true,
			KeepAlivePeriod:  180 * time.Second,
			TCPRBufSize:      262144,
			TCPWBufSize:      65536,
			TCPReadTimeout:   time.Second,
			TCPWriteTimeout:  5 * time.Second,
			WaitTimeout:      time.Second,
			MaxMsgLen:        4096,
			SessionName:      "rpc_client",
		},
	}
}
