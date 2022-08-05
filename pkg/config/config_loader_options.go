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
	"github.com/knadh/koanf"
	"github.com/seata/seata-go/pkg/common/constant"
	"github.com/seata/seata-go/pkg/common/file"
	"github.com/seata/seata-go/pkg/common/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type loaderConf struct {
	suffix string      // loaderConf file extension default yaml
	path   string      // loaderConf file path default ./conf/config.yaml
	delim  string      // loaderConf file delim default .
	bytes  []byte      // config bytes
	rc     *RootConfig // user provide rootConfig built by config api
	name   string      // config file name
}

type LoaderConfOption interface {
	apply(vc *loaderConf)
}

func (fn loaderConfigFunc) apply(vc *loaderConf) {
	fn(vc)
}

type loaderConfigFunc func(*loaderConf)

// WithPath set load config path
func WithPath(path string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.path = absolutePath(path)
		if bytes, err := ioutil.ReadFile(path); err != nil {
			panic(err)
		} else {
			conf.bytes = bytes
		}
		name, suffix := resolverFilePath(path)
		conf.suffix = suffix
		conf.name = name
	})
}

func NewLoaderConf(opts ...LoaderConfOption) *loaderConf {
	configFilePath := "../conf/config.yaml"
	if configFilePathFromEnv := os.Getenv(constant.ConfigFileEnvKey); configFilePathFromEnv != "" {
		configFilePath = configFilePathFromEnv
	}
	name, suffix := resolverFilePath(configFilePath)
	conf := &loaderConf{
		suffix: suffix,
		path:   absolutePath(configFilePath),
		delim:  ".",
		name:   name,
	}
	for _, opt := range opts {
		opt.apply(conf)
	}
	if conf.rc != nil {
		return conf
	}
	if len(conf.bytes) <= 0 {
		if bytes, err := ioutil.ReadFile(conf.path); err != nil {
			panic(err)
		} else {
			conf.bytes = bytes
		}
	}
	return conf
}

//resolverFilePath resolver file path
// eg: give a ./conf/seata.yaml return seata and yaml
func resolverFilePath(path string) (name, suffix string) {
	paths := strings.Split(path, "/")
	fileName := strings.Split(paths[len(paths)-1], ".")
	if len(fileName) < 2 {
		return fileName[0], string(file.YAML)
	}
	return fileName[0], fileName[1]
}

// absolutePath get absolut path
func absolutePath(inPath string) string {

	if inPath == "$HOME" || strings.HasPrefix(inPath, "$HOME"+string(os.PathSeparator)) {
		inPath = userHomeDir() + inPath[5:]
	}

	if filepath.IsAbs(inPath) {
		return filepath.Clean(inPath)
	}

	p, err := filepath.Abs(inPath)
	if err == nil {
		return filepath.Clean(p)
	}

	return ""
}

//userHomeDir get gopath
func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

//MergeConfig merge config file
func (conf *loaderConf) MergeConfig(koan *koanf.Koanf) *koanf.Koanf {
	var (
		activeKoan *koanf.Koanf
		activeConf *loaderConf
	)
	active := koan.String("seata.profiles.active")
	active = getLegalActive(active)
	log.Infof("The following profiles are active: %s", active)
	if defaultActive != active {
		path := conf.getActiveFilePath(active)
		if !pathExists(path) {
			log.Debugf("Config file:%s not exist skip config merge", path)
			return koan
		}
		activeConf = NewLoaderConf(WithPath(path))
		activeKoan = GetConfigResolver(activeConf)
		if err := koan.Merge(activeKoan); err != nil {
			log.Debugf("Config merge err %s", err)
		}
	}
	return koan
}

func (conf *loaderConf) getActiveFilePath(active string) string {
	suffix := constant.DotSeparator + conf.suffix
	return strings.ReplaceAll(conf.path, suffix, "") + "-" + active + suffix
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else {
		return !os.IsNotExist(err)
	}
}
