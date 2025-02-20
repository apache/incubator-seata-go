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

package fence

import (
	"flag"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/config"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/handler"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/store/db/dao"
	"time"
)

var (
	FenceConfig Config
)

func InitFenceConfig(cfg Config) {
	FenceConfig = cfg

	if FenceConfig.Enable {
		dao.GetTccFenceStoreDatabaseMapper().InitLogTableName(FenceConfig.LogTableName)
		handler.GetFenceHandler().InitCleanPeriod(FenceConfig.CleanPeriod)
		config.InitCleanTask(FenceConfig.Url)
	}
}

type Config struct {
	Enable       bool          `yaml:"enable" json:"enable" koanf:"enable"`
	Url          string        `yaml:"url" json:"url" koanf:"url"`
	LogTableName string        `yaml:"log-table-name" json:"log-table-name" koanf:"log-table-name"`
	CleanPeriod  time.Duration `yaml:"clean-period" json:"clean-period" koanf:"clean-period"`
}

// RegisterFlagsWithPrefix for Config.
func (cfg *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.Enable, prefix+".enable", false, "Whether the fence is initialized.")
	f.StringVar(&cfg.Url, prefix+".url", "", "Data source name.")
	f.StringVar(&cfg.LogTableName, prefix+".log-table-name", "tcc_fence_log", "Undo log table name.")
	f.DurationVar(&cfg.CleanPeriod, prefix+".clean-period", 5*time.Minute, "Undo log retention time.")
}
