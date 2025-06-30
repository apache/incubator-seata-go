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

import "flag"

type Config struct {
	ReconnectInterval int           `yaml:"reconnect-interval" json:"reconnect-interval" koanf:"reconnect-interval"`
	ConnectionNum     int           `yaml:"connection-num" json:"connection-num" koanf:"connection-num"`
	SessionConfig     SessionConfig `yaml:"session" json:"session" koanf:"session"`
	StreamConfig      StreamConfig  `yaml:"stream" json:"stream" koanf:"stream"`
}

// RegisterFlagsWithPrefix for Config.
func (cfg *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.IntVar(&cfg.ReconnectInterval, prefix+".reconnect-interval", 0, "Reconnect interval.")
	f.IntVar(&cfg.ConnectionNum, prefix+".connection-num", 1, "The getty_session pool.")
	cfg.SessionConfig.RegisterFlagsWithPrefix(prefix+".session", f)
	cfg.StreamConfig.RegisterFlagsWithPrefix(prefix+".stream", f)
}
