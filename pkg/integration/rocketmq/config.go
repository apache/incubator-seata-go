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

package rocketmq

type Config struct {
	Addr      string
	Namespace string
	GroupName string
}

type ConfigBuilder struct {
	config *Config
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &Config{},
	}
}

func (b *ConfigBuilder) WithAddr(addr string) *ConfigBuilder {
	b.config.Addr = addr
	return b
}

func (b *ConfigBuilder) WithNamespace(namespace string) *ConfigBuilder {
	b.config.Namespace = namespace
	return b
}

func (b *ConfigBuilder) WithGroupName(groupName string) *ConfigBuilder {
	b.config.GroupName = groupName
	return b
}

func (b *ConfigBuilder) Build() Config {
	return *b.config
}
