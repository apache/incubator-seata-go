package config

import (
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/getty/config"
)

// Config holds supported types by the multiconfig package
type GettyConfig struct {
	// getty_session
	Session_Timeout string `default:"60s" yaml:"session_timeout" json:"session_timeout,omitempty"`
	SessionTimeout  time.Duration

	// getty_session tcp parameters
	GettySessionParam config.GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
}

// CheckValidity ...
func (c *GettyConfig) CheckValidity() error {
	var err error

	if c.SessionTimeout, err = time.ParseDuration(c.Session_Timeout); err != nil {
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
		Session_Timeout: "180s",
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
			MaxMsgLen:        102400,
			SessionName:      "server",
		},
	}
}
