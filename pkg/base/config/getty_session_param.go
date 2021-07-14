package config

import (
	"time"
)

// GettySessionParam getty session param
type GettySessionParam struct {
	CompressEncoding bool          `default:"false" yaml:"compress_encoding" json:"compress_encoding,omitempty"`
	TCPNoDelay       bool          `default:"true" yaml:"tcp_no_delay" json:"tcp_no_delay,omitempty"`
	TCPKeepAlive     bool          `default:"true" yaml:"tcp_keep_alive" json:"tcp_keep_alive,omitempty"`
	KeepAlivePeriod  time.Duration `default:"180s" yaml:"keep_alive_period" json:"keep_alive_period,omitempty"`
	TCPRBufSize      int           `default:"262144" yaml:"tcp_r_buf_size" json:"tcp_r_buf_size,omitempty"`
	TCPWBufSize      int           `default:"65536" yaml:"tcp_w_buf_size" json:"tcp_w_buf_size,omitempty"`
	TCPReadTimeout   time.Duration `default:"1s" yaml:"tcp_read_timeout" json:"tcp_read_timeout,omitempty"`
	TCPWriteTimeout  time.Duration `default:"5s" yaml:"tcp_write_timeout" json:"tcp_write_timeout,omitempty"`
	WaitTimeout      time.Duration `default:"7s" yaml:"wait_timeout" json:"wait_timeout,omitempty"`
	MaxMsgLen        int           `default:"4096" yaml:"max_msg_len" json:"max_msg_len,omitempty"`
	SessionName      string        `default:"rpc" yaml:"session_name" json:"session_name,omitempty"`
}
