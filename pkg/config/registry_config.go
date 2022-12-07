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

type File struct {
	Name string `yaml:"name" json:"name,omitempty" property:"name"`
}

type NacosRegistry struct {
	Application string `yaml:"application" json:"application,omitempty" property:"application"`
	ServerAddr  string `yaml:"server-addr" json:"server-addr,omitempty" property:"server-addr"`
	Group       string `yaml:"group" json:"group,omitempty" property:"group"`
	Namespace   string `yaml:"namespace" json:"namespace,omitempty" property:"namespace"`
	Username    string `yaml:"username" json:"username,omitempty" property:"username"`
	Password    string `yaml:"password" json:"password,omitempty" property:"password"`
}

// Registry is Registration Center configuration file
type Registry struct {
	Type  string        `yaml:"type" json:"type,omitempty" property:"type"`
	File  File          `yaml:"file" json:"file,omitempty" property:"file"`
	Nacos NacosRegistry `yaml:"nacos" json:"nacos,omitempty" property:"nacos"`
}
