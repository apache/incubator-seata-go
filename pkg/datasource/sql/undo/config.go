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

package undo

import (
	"flag"

	"seata.apache.org/seata-go/pkg/compressor"
)

var (
	UndoConfig Config
)

func InitUndoConfig(cfg Config) {
	UndoConfig = cfg
}

type CompressConfig struct {
	Enable    bool   `yaml:"enable" json:"enable,omitempty" koanf:"enable"`
	Type      string `yaml:"type" json:"type,omitempty" koanf:"type"`
	Threshold string `yaml:"threshold" json:"threshold,omitempty"  koanf:"threshold"`
}

type Config struct {
	DataValidation        bool           `yaml:"data-validation" json:"data-validation,omitempty" koanf:"data-validation"`
	LogSerialization      string         `yaml:"log-serialization" json:"log-serialization,omitempty" koanf:"log-serialization"`
	LogTable              string         `yaml:"log-table" json:"log-table,omitempty" koanf:"log-table"`
	OnlyCareUpdateColumns bool           `yaml:"only-care-update-columns" json:"only-care-update-columns,omitempty" koanf:"only-care-update-columns"`
	CompressConfig        CompressConfig `yaml:"compress" json:"compress,omitempty" koanf:"compress"`
}

func (u *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.BoolVar(&u.DataValidation, prefix+".data-validation", true, "Judge whether the before image and after image are the same，If it is the same, undo will not be recorded")
	f.StringVar(&u.LogSerialization, prefix+".log-serialization", "json", "Serialization method.")
	f.StringVar(&u.LogTable, prefix+".log-table", "undo_log", "undo log table name.")
	f.BoolVar(&u.OnlyCareUpdateColumns, prefix+".only-care-update-columns", true, "The switch for degrade check.")
	u.CompressConfig.RegisterFlagsWithPrefix(prefix+".compress", f)
}

func (c *CompressConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&c.Type, prefix+".type", string(compressor.CompressorNone), "Compression type")
	f.StringVar(&c.Threshold, prefix+".threshold", "64k", "Compression threshold")
}
