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

package config

import (
	"flag"
	"time"

	"seata.apache.org/seata-go/pkg/util/flagext"
)

var seataConfig *SeataConfig

type Config struct {
	ReconnectInterval int           `yaml:"reconnect-interval" json:"reconnect-interval" koanf:"reconnect-interval"`
	ConnectionNum     int           `yaml:"connection-num" json:"connection-num" koanf:"connection-num"`
	LoadBalanceType   string        `yaml:"load-balance-type" json:"load-balance-type" koanf:"load-balance-type"`
	SessionConfig     SessionConfig `yaml:"session" json:"session" koanf:"session"`
}

// RegisterFlagsWithPrefix for Config.
func (cfg *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.IntVar(&cfg.ReconnectInterval, prefix+".reconnect-interval", 0, "Reconnect interval.")
	f.IntVar(&cfg.ConnectionNum, prefix+".connection-num", 1, "The getty_session pool.")
	f.StringVar(&cfg.LoadBalanceType, prefix+".load-balance-type", "XID", "default load balance type")
	cfg.SessionConfig.RegisterFlagsWithPrefix(prefix+".session", f)
}

type ShutdownConfig struct {
	Wait time.Duration `yaml:"wait" json:"wait" konaf:"wait"`
}

func (cfg *ShutdownConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.Wait, prefix+".wait", 3*time.Second, "Shutdown wait time.")
}

type TransportConfig struct {
	ShutdownConfig                 ShutdownConfig `yaml:"shutdown" json:"shutdown" koanf:"shutdown"`
	Type                           string         `yaml:"type" json:"type" koanf:"type"`
	Server                         string         `yaml:"server" json:"server" koanf:"server"`
	Heartbeat                      bool           `yaml:"heartbeat" json:"heartbeat" koanf:"heartbeat"`
	Serialization                  string         `yaml:"serialization" json:"serialization" koanf:"serialization"`
	Compressor                     string         `yaml:"compressor" json:"compressor" koanf:"compressor"`
	EnableTmClientBatchSendRequest bool           `yaml:"enable-tm-client-batch-send-request" json:"enable-tm-client-batch-send-request" koanf:"enable-tm-client-batch-send-request"`
	EnableRmClientBatchSendRequest bool           `yaml:"enable-rm-client-batch-send-request" json:"enable-rm-client-batch-send-request" koanf:"enable-rm-client-batch-send-request"`
	RPCRmRequestTimeout            time.Duration  `yaml:"rpc-rm-request-timeout" json:"rpc-rm-request-timeout" koanf:"rpc-rm-request-timeout"`
	RPCTmRequestTimeout            time.Duration  `yaml:"rpc-tm-request-timeout" json:"rpc-tm-request-timeout" koanf:"rpc-tm-request-timeout"`
}

func (cfg *TransportConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	cfg.ShutdownConfig.RegisterFlagsWithPrefix(prefix+".shutdown", f)
	f.StringVar(&cfg.Type, prefix+".type", "TCP", "Transport protocol type.")
	f.StringVar(&cfg.Server, prefix+".server", "NIO", "Server type.")
	f.BoolVar(&cfg.Heartbeat, prefix+".heartbeat", true, "Heartbeat.")
	f.StringVar(&cfg.Serialization, prefix+".serialization", "seata", "Encoding and decoding mode.")
	f.StringVar(&cfg.Compressor, prefix+".compressor", "none", "Message compression mode.")
	f.BoolVar(&cfg.EnableTmClientBatchSendRequest, prefix+".enable-tm-client-batch-send-request", false, "Allow batch sending of requests (TM).")
	f.BoolVar(&cfg.EnableRmClientBatchSendRequest, prefix+".enable-rm-client-batch-send-request", true, "Allow batch sending of requests (RM).")
	f.DurationVar(&cfg.RPCRmRequestTimeout, prefix+".rpc-rm-request-timeout", 30*time.Second, "RM send request timeout.")
	f.DurationVar(&cfg.RPCTmRequestTimeout, prefix+".rpc-tm-request-timeout", 30*time.Second, "TM send request timeout.")
}

// todo refactor config
type SeataConfig struct {
	ApplicationID        string
	TxServiceGroup       string
	ServiceVgroupMapping flagext.StringMap
	ServiceGrouplist     flagext.StringMap
	LoadBalanceType      string
}

func IniConfig(seataConf *SeataConfig) {
	seataConfig = seataConf
}

func GetSeataConfig() *SeataConfig {
	return seataConfig
}
