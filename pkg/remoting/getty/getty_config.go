/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package getty

import (
	"flag"
	"time"
)

// Config holds supported types by the multiconfig package
type Config struct {
	ReconnectInterval int `yaml:"reconnect_interval" json:"reconnect_interval,omitempty"`
	// getty_session pool
	ConnectionNum int `yaml:"connection_number" json:"connection_number,omitempty"`
	// heartbeat
	HeartbeatPeriod time.Duration `yaml:"heartbeat_period" json:"heartbeat_period,omitempty"`
	// getty_session tcp parameters
	GettySessionParam SessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
}

// RegisterFlags registers the Config flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	f.IntVar(&cfg.ReconnectInterval, "remoting.getty.reconnect.interval", 0, "the reconnect interval of getty")
	f.IntVar(&cfg.ConnectionNum, "remoting.getty.connection.num", 1, "the num of getty connection")
	f.DurationVar(&cfg.HeartbeatPeriod, "remoting.getty.heartbeat.period", 10*time.Second, "the heartbeat period of getty")

	cfg.GettySessionParam.RegisterFlags(f)
}

// SessionParam getty session param
type SessionParam struct {
	CompressEncoding bool          `yaml:"compress_encoding" json:"compress_encoding,omitempty"`
	TCPNoDelay       bool          `yaml:"tcp_no_delay" json:"tcp_no_delay,omitempty"`
	TCPKeepAlive     bool          `yaml:"tcp_keep_alive" json:"tcp_keep_alive,omitempty"`
	KeepAlivePeriod  time.Duration `yaml:"keep_alive_period" json:"keep_alive_period,omitempty"`
	CronPeriod       time.Duration `yaml:"cron_period" json:"cron_period,omitempty"`
	TCPRBufSize      int           `yaml:"tcp_r_buf_size" json:"tcp_r_buf_size,omitempty"`
	TCPWBufSize      int           `yaml:"tcp_w_buf_size" json:"tcp_w_buf_size,omitempty"`
	TCPReadTimeout   time.Duration `yaml:"tcp_read_timeout" json:"tcp_read_timeout,omitempty"`
	TCPWriteTimeout  time.Duration `yaml:"tcp_write_timeout" json:"tcp_write_timeout,omitempty"`
	WaitTimeout      time.Duration `yaml:"wait_timeout" json:"wait_timeout,omitempty"`
	MaxMsgLen        int           `yaml:"max_msg_len" json:"max_msg_len,omitempty"`
	SessionName      string        `yaml:"session_name" json:"session_name,omitempty"`
}

func (cfg *SessionParam) RegisterFlags(f *flag.FlagSet) {
	f.BoolVar(&cfg.CompressEncoding, "remoting.getty.session.compress.enable", false, "whether compress getty session")
	f.BoolVar(&cfg.TCPNoDelay, "remoting.getty.session.tcp.no.delay", true, "whether getty session tcp no delay")
	f.BoolVar(&cfg.TCPKeepAlive, "remoting.getty.session.tcp.keepalive", true, "whether getty session tcp keep alive")
	f.DurationVar(&cfg.KeepAlivePeriod, "remoting.getty.session.keepalive.period", 180*time.Second, "the period of session key alive")
	f.IntVar(&cfg.TCPRBufSize, "remoting.getty.session.tcpr.bufsize", 2144, "the tcp read buffer size")
	f.IntVar(&cfg.TCPWBufSize, "remoting.getty.session.tcpw.bufsize", 65536, "the tcp write buffer size")
	f.DurationVar(&cfg.TCPReadTimeout, "remoting.getty.session.tcp.read.timeout", time.Second, "the tcp read timeout")
	f.DurationVar(&cfg.TCPWriteTimeout, "remoting.getty.session.tcp.write.timeout", 5*time.Second, "the tcp write timeout")
	f.DurationVar(&cfg.WaitTimeout, "remoting.getty.session.wait.timeout", time.Second, "the tcp wait timeout")
	f.DurationVar(&cfg.CronPeriod, "remoting.getty.session.cron.period", time.Second, "the session's cron period")
	f.IntVar(&cfg.MaxMsgLen, "remoting.getty.session.max.msg.len", 4096, "the max message len")
	f.StringVar(&cfg.SessionName, "remoting.getty.session.name", "rpc_client", "the name of getty session")
}
