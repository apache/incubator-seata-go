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

// Config holds supported types by the multiconfig package
type GettyConfig struct {
	// getty_session
	SessionTmt     string `default:"60s" yaml:"session_timeout" json:"session_timeout,omitempty"`
	SessionTimeout time.Duration
	SessionNumber  int `default:"1000" yaml:"session_number" json:"session_number,omitempty"`

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
		SessionTmt:    "180s",
		SessionNumber: 700,
		GrPoolSize:    200,
		QueueNumber:   6,
		QueueLen:      64,
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
			MaxMsgLen:        102400,
			SessionName:      "server",
		},
	}
}
