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
	"fmt"
	"github.com/knadh/koanf"
	"runtime"
)

var (
	rootConfig = NewRootConfigBuilder().Build()
)

func Load(opts ...LoaderConfOption) error {
	// conf
	conf := NewLoaderConf(opts...)
	if conf.rc == nil {
		koan := GetConfigResolver(conf)
		koan = conf.MergeConfig(koan)
		if err := koan.UnmarshalWithConf(rootConfig.Prefix(),
			rootConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
			var stackBuf [1024]byte
			stackBufLen := runtime.Stack(stackBuf[:], false)
			fmt.Printf("==> %s\n", string(stackBuf[:stackBufLen]))
			return err
		}
	} else {
		rootConfig = conf.rc
	}
	if err := rootConfig.Init(); err != nil {
		return err
	}
	return nil
}
func NewRootConfigBuilder() *RootConfigBuilder {
	return &RootConfigBuilder{rootConfig: newEmptyRootConfig()}
}

type RootConfigBuilder struct {
	rootConfig *RootConfig
}

// newEmptyRootConfig get empty root config
func newEmptyRootConfig() *RootConfig {
	newRootConfig := &RootConfig{}
	return newRootConfig
}

func (rb *RootConfigBuilder) Build() *RootConfig {
	return rb.rootConfig
}
