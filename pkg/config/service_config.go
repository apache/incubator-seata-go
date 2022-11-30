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

type Grouplist struct {
	Default string `yaml:"default" json:"default,omitempty" property:"default"`
}

type VgroupMapping struct {
	DefaultTxGroup string `yaml:"default_tx_group" json:"default_tx_group,omitempty" property:"default_tx_group"`
}

type Service struct {
	VgroupMapping            VgroupMapping `yaml:"vgroup-mapping" json:"vgroup-mapping,omitempty" property:"vgroup-mapping"`
	Grouplist                Grouplist     `yaml:"grouplist" json:"grouplist,omitempty" property:"grouplist"`
	EnableDegrade            bool          `yaml:"enable-degrade" json:"enable-degrade,omitempty" property:"enable-degrade"`
	DisableGlobalTransaction bool          `yaml:"disable-global-transaction" json:"disable-global-transaction,omitempty" property:"disable-global-transaction"`
}
