package config

import (
	"time"
)

import (
	"github.com/dubbogo/getty"
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/base/getty/config"
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

	// getty_session
	SessionTmt     string `default:"60s" yaml:"session_timeout" json:"session_timeout,omitempty"`
	SessionTimeout time.Duration

	// Connection Pool
	PoolSize int `default:"2" yaml:"pool_size" json:"pool_size,omitempty"`
	PoolTTL  int `default:"180" yaml:"pool_ttl" json:"pool_ttl,omitempty"`

	// grpool
	GrPoolSize  int `default:"0" yaml:"gr_pool_size" json:"gr_pool_size,omitempty"`
	QueueLen    int `default:"0" yaml:"queue_len" json:"queue_len,omitempty"`
	QueueNumber int `default:"0" yaml:"queue_number" json:"queue_number,omitempty"`

	// getty_session tcp parameters
	GettySessionParam config.GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
}

// CheckValidity ...
func (c *GettyConfig) CheckValidity() error {
	var err error

	if c.HeartbeatPeriod, err = time.ParseDuration(c.HeartbeatPrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(HeartbeatPeriod{%#v})", c.HeartbeatPrd)
	}

	if c.SessionTimeout, err = time.ParseDuration(c.SessionTmt); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(SessionTimeout{%#v})", c.SessionTimeout)
	}

	if c.SessionTimeout >= time.Duration(getty.MaxWheelTimeSpan) {
		return errors.WithMessagef(err, "session_timeout %s should be less than %s",
			c.SessionTimeout, time.Duration(getty.MaxWheelTimeSpan))
	}

	return errors.WithStack(c.GettySessionParam.CheckValidity())
}

// GetDefaultGettyConfig ...
func GetDefaultGettyConfig() GettyConfig {
	return GettyConfig{
		ReconnectInterval: 0,
		ConnectionNum:     1,
		HeartbeatPrd:      "10s",
		SessionTmt:        "180s",
		PoolSize:          4,
		PoolTTL:           600,
		GrPoolSize:        200,
		QueueLen:          64,
		QueueNumber:       10,
		GettySessionParam: config.GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePrd:     "180s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			PkgWQSize:        512,
			TcpReadTmt:       "1s",
			TcpWriteTmt:      "5s",
			WaitTmt:          "1s",
			MaxMsgLen:        4096,
			SessionName:      "client",
		},
	}
}
