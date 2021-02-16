package config

import (
	"time"
)

import (
	"github.com/pkg/errors"
)

type GettySessionParam struct {
	CompressEncoding bool   `default:"false" yaml:"compress_encoding" json:"compress_encoding,omitempty"`
	TcpNoDelay       bool   `default:"true" yaml:"tcp_no_delay" json:"tcp_no_delay,omitempty"`
	TcpKeepAlive     bool   `default:"true" yaml:"tcp_keep_alive" json:"tcp_keep_alive,omitempty"`
	KeepAlivePrd     string `default:"180s" yaml:"keep_alive_period" json:"keep_alive_period,omitempty"`
	KeepAlivePeriod  time.Duration
	TcpRBufSize      int    `default:"262144" yaml:"tcp_r_buf_size" json:"tcp_r_buf_size,omitempty"`
	TcpWBufSize      int    `default:"65536" yaml:"tcp_w_buf_size" json:"tcp_w_buf_size,omitempty"`
	TcpReadTmt       string `default:"1s" yaml:"tcp_read_timeout" json:"tcp_read_timeout,omitempty"`
	TcpReadTimeout   time.Duration
	TcpWriteTmt      string `default:"5s" yaml:"tcp_write_timeout" json:"tcp_write_timeout,omitempty"`
	TcpWriteTimeout  time.Duration
	WaitTmt          string `default:"7s" yaml:"wait_timeout" json:"wait_timeout,omitempty"`
	WaitTimeout      time.Duration
	MaxMsgLen        int    `default:"4096" yaml:"max_msg_len" json:"max_msg_len,omitempty"`
	SessionName      string `default:"rpc" yaml:"session_name" json:"session_name,omitempty"`
}

// CheckValidity ...
func (c *GettySessionParam) CheckValidity() error {
	var err error

	if c.KeepAlivePeriod, err = time.ParseDuration(c.KeepAlivePrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(KeepAlivePeriod{%#v})", c.KeepAlivePrd)
	}

	if c.TcpReadTimeout, err = time.ParseDuration(c.TcpReadTmt); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(TcpReadTimeout{%#v})", c.TcpReadTmt)
	}

	if c.TcpWriteTimeout, err = time.ParseDuration(c.TcpWriteTmt); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(TcpWriteTimeout{%#v})", c.TcpWriteTmt)
	}

	if c.WaitTimeout, err = time.ParseDuration(c.WaitTmt); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(WaitTimeout{%#v})", c.WaitTmt)
	}

	return nil
}
