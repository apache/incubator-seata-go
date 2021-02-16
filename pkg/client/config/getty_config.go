package config

import (
	"time"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/getty/config"
)

// GettyConfig
//Config holds supported types by the multiconfig package
type GettyConfig struct {
	ReconnectInterval int `default:"0" yaml:"reconnect_interval" json:"reconnect_interval,omitempty"`
	// getty_session pool
	ConnectionNum int `default:"16" yaml:"connection_number" json:"connection_number,omitempty"`

	// heartbeat
	HeartbeatPrd    string `default:"15s" yaml:"heartbeat_period" json:"heartbeat_period,omitempty"`
	HeartbeatPeriod time.Duration

	// getty_session tcp parameters
	GettySessionParam config.GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
}

// CheckValidity ...
func (c *GettyConfig) CheckValidity() error {
	var err error

	if c.HeartbeatPeriod, err = time.ParseDuration(c.HeartbeatPrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(HeartbeatPeriod{%#v})", c.HeartbeatPrd)
	}

	return errors.WithStack(c.GettySessionParam.CheckValidity())
}

// GetDefaultGettyConfig ...
func GetDefaultGettyConfig() GettyConfig {
	return GettyConfig{
		ReconnectInterval: 0,
		ConnectionNum:     1,
		HeartbeatPrd:      "10s",
		GettySessionParam: config.GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePrd:     "180s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			TcpReadTmt:       "1s",
			TcpWriteTmt:      "5s",
			WaitTmt:          "1s",
			MaxMsgLen:        4096,
			SessionName:      "rpc_client",
		},
	}
}
