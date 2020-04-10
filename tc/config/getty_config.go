package config

import "time"

type GettySessionParam struct {
	CompressEncoding bool   `default:"false" yaml:"compress_encoding" json:"compress_encoding,omitempty"`
	TcpNoDelay       bool   `default:"true" yaml:"tcp_no_delay" json:"tcp_no_delay,omitempty"`
	TcpKeepAlive     bool   `default:"true" yaml:"tcp_keep_alive" json:"tcp_keep_alive,omitempty"`
	KeepAlivePrd     string `default:"180s" yaml:"keep_alive_period" json:"keep_alive_period,omitempty"`
	KeepAlivePeriod  time.Duration
	TcpRBufSize      int    `default:"262144" yaml:"tcp_r_buf_size" json:"tcp_r_buf_size,omitempty"`
	TcpWBufSize      int    `default:"65536" yaml:"tcp_w_buf_size" json:"tcp_w_buf_size,omitempty"`
	PkgWQSize        int    `default:"1024" yaml:"pkg_wq_size" json:"pkg_wq_size,omitempty"`
	TcpReadTmt       string `default:"1s" yaml:"tcp_read_timeout" json:"tcp_read_timeout,omitempty"`
	TcpReadTimeout   time.Duration
	TcpWriteTmt      string `default:"5s" yaml:"tcp_write_timeout" json:"tcp_write_timeout,omitempty"`
	TcpWriteTimeout  time.Duration
	WaitTmt      string `default:"7s" yaml:"wait_timeout" json:"wait_timeout,omitempty"`
	WaitTimeout      time.Duration
	MaxMsgLen        int    `default:"1024" yaml:"max_msg_len" json:"max_msg_len,omitempty"`
	SessionName      string `default:"rpc" yaml:"session_name" json:"session_name,omitempty"`
}

// Config holds supported types by the multiconfig package
type GettyConfig struct {
	// session
	SessionTmt     string `default:"60s" yaml:"session_timeout" json:"session_timeout,omitempty"`
	SessionTimeout time.Duration
	SessionNumber  int `default:"1000" yaml:"session_number" json:"session_number,omitempty"`

	// grpool
	GrPoolSize  int `default:"0" yaml:"gr_pool_size" json:"gr_pool_size,omitempty"`
	QueueLen    int `default:"0" yaml:"queue_len" json:"queue_len,omitempty"`
	QueueNumber int `default:"0" yaml:"queue_number" json:"queue_number,omitempty"`

	// session tcp parameters
	GettySessionParam GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
}