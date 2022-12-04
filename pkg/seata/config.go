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

package seata

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/constant/file"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/util/flagext"
)

const (
	configFileEnvKey = "SEATA_GO_CONFIG_PATH"
	configPrefix     = "seata"
)

type Config struct {
	TCCConfig tcc.Config `yaml:"tcc"`
}

func (c *Config) RegisterFlags(f *flag.FlagSet) {
	c.TCCConfig.FenceConfig.RegisterFlagsWithPrefix("tcc", f)
}

type loaderConf struct {
	suffix string // loaderConf file extension default yaml
	path   string // loaderConf file path default ./conf/seatago.yaml
	delim  string // loaderConf file delim default .
	bytes  []byte // config bytes
	name   string // config file name
}

func Load() *Config {
	configFilePath := "../../conf/seatago.yaml"
	if configFilePathFromEnv := os.Getenv(configFileEnvKey); configFilePathFromEnv != "" {
		configFilePath = configFilePathFromEnv
	}
	return LoadPath(configFilePath)
}

func LoadPath(configFilePath string) *Config {
	conf := NewLoaderConf(configFilePath)
	koan := GetConfigResolver(conf)

	var cfg Config
	// This sets default values from flags to the config.
	// It needs to be called before parsing the config file!
	flagext.RegisterFlags(&cfg)

	//koan = conf.MergeConfig(koan)
	if err := koan.UnmarshalWithConf(configPrefix, &cfg, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		panic(err)
	}
	return &cfg
}

//resolverFilePath resolver file path
// eg: give a ./conf/seatago.yaml return seatago and yaml
func resolverFilePath(path string) (name, suffix string) {
	paths := strings.Split(path, "/")
	fileName := strings.Split(paths[len(paths)-1], ".")
	if len(fileName) < 2 {
		return fileName[0], string(file.YAML)
	}
	return fileName[0], fileName[1]
}

// GetConfigResolver get config resolver
func GetConfigResolver(conf *loaderConf) *koanf.Koanf {
	var (
		k   *koanf.Koanf
		err error
	)
	if len(conf.suffix) <= 0 {
		conf.suffix = string(file.YAML)
	}
	if len(conf.delim) <= 0 {
		conf.delim = "."
	}
	bytes := conf.bytes
	if len(bytes) <= 0 {
		panic(fmt.Errorf("bytes is nil,please set bytes or file path"))
	}
	k = koanf.New(conf.delim)

	switch conf.suffix {
	case "yaml", "yml":
		err = k.Load(rawbytes.Provider(bytes), yaml.Parser())
	case "json":
		err = k.Load(rawbytes.Provider(bytes), json.Parser())
	case "toml":
		err = k.Load(rawbytes.Provider(bytes), toml.Parser())
	default:
		err = fmt.Errorf("no support %s file suffix", conf.suffix)
	}

	if err != nil {
		panic(err)
	}
	return k
}

func NewLoaderConf(configFilePath string) *loaderConf {
	name, suffix := resolverFilePath(configFilePath)
	conf := &loaderConf{
		suffix: suffix,
		path:   absolutePath(configFilePath),
		delim:  ".",
		name:   name,
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

// LoadYMLConfig Load yml config byte from file check file type is *.yml or *.yaml`
func LoadYMLConfig(confProFile string) ([]byte, error) {
	if len(confProFile) == 0 {
		return nil, fmt.Errorf("application configure(provider) file name is nil")
	}

	if path.Ext(confProFile) != ".yml" && path.Ext(confProFile) != ".yaml" {
		return nil, fmt.Errorf("application configure file name{%v} suffix must be .yml or .yaml", confProFile)
	}

	return ioutil.ReadFile(confProFile)
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
