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

package discovery

import (
	"flag"

	"github.com/seata/seata-go/pkg/util/flagext"
)

type ServiceConfig struct {
	// Primarily used for multiple data centers (service/tx groups). It holds each data center's cluster name.
	// map key is data center name (service/tx group name), while value is the cluster name used inside that data center.
	VgroupMapping flagext.StringMap `yaml:"vgroup-mapping" json:"vgroup-mapping" koanf:"vgroup-mapping"`
	// This is only useful for registry file. Otherwise this is useless.
	Grouplist                flagext.StringMap `yaml:"grouplist" json:"grouplist" koanf:"grouplist"`
	EnableDegrade            bool              `yaml:"enable-degrade" json:"enable-degrade" koanf:"enable-degrade"`
	DisableGlobalTransaction bool              `yaml:"disable-global-transaction" json:"disable-global-transaction" koanf:"disable-global-transaction"`
}

func (cfg *ServiceConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.EnableDegrade, prefix+".enable-degrade", false, "degrade current not support.")
	f.BoolVar(&cfg.DisableGlobalTransaction, prefix+".disable-global-transaction", false, "disable globalTransaction.")
	f.Var(&cfg.VgroupMapping, prefix+".vgroup-mapping", "The vgroup mapping.")
	f.Var(&cfg.Grouplist, prefix+".grouplist", "The group list.")
}
