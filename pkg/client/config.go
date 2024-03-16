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

package client

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"

	"seata.apache.org/seata-go/pkg/discovery"

	"seata.apache.org/seata-go/pkg/datasource/sql"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	remoteConfig "seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/rm/tcc"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/flagext"
)

const (
	configFileEnvKey = "SEATA_GO_CONFIG_PATH"
	configPrefix     = "seata"
)

const (
	jsonSuffix = "json"
	tomlSuffix = "toml"
	yamlSuffix = "yaml"
	ymlSuffix  = "yml"
)

type ClientConfig struct {
	TmConfig   tm.TmConfig  `yaml:"tm" json:"tm,omitempty" koanf:"tm"`
	RmConfig   rm.Config    `yaml:"rm" json:"rm,omitempty" koanf:"rm"`
	UndoConfig undo.Config  `yaml:"undo" json:"undo,omitempty" koanf:"undo"`
	XaConfig   sql.XAConfig `yaml:"xa" json:"xa" koanf:"xa"`
}

func (c *ClientConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	c.TmConfig.RegisterFlagsWithPrefix(prefix+".tm", f)
	c.RmConfig.RegisterFlagsWithPrefix(prefix+".rm", f)
	c.UndoConfig.RegisterFlagsWithPrefix(prefix+".undo", f)
	c.XaConfig.RegisterFlagsWithPrefix(prefix+".xa", f)
}

type Config struct {
	Enabled                   bool   `yaml:"enabled" json:"enabled,omitempty" koanf:"enabled"`
	ApplicationID             string `yaml:"application-id" json:"application-id,omitempty" koanf:"application-id"`
	TxServiceGroup            string `yaml:"tx-service-group" json:"tx-service-group,omitempty" koanf:"tx-service-group"`
	AccessKey                 string `yaml:"access-key" json:"access-key,omitempty" koanf:"access-key"`
	SecretKey                 string `yaml:"secret-key" json:"secret-key,omitempty" koanf:"secret-key"`
	EnableAutoDataSourceProxy bool   `yaml:"enable-auto-data-source-proxy" json:"enable-auto-data-source-proxy,omitempty" koanf:"enable-auto-data-source-proxy"`
	DataSourceProxyMode       string `yaml:"data-source-proxy-mode" json:"data-source-proxy-mode,omitempty" koanf:"data-source-proxy-mode"`

	AsyncWorkerConfig sql.AsyncWorkerConfig        `yaml:"async" json:"async" koanf:"async"`
	TCCConfig         tcc.Config                   `yaml:"tcc" json:"tcc" koanf:"tcc"`
	ClientConfig      ClientConfig                 `yaml:"client" json:"client" koanf:"client"`
	GettyConfig       remoteConfig.Config          `yaml:"getty" json:"getty" koanf:"getty"`
	TransportConfig   remoteConfig.TransportConfig `yaml:"transport" json:"transport" koanf:"transport"`
	ServiceConfig     discovery.ServiceConfig      `yaml:"service" json:"service" koanf:"service"`
	RegistryConfig    discovery.RegistryConfig     `yaml:"registry" json:"registry" koanf:"registry"`
}

func (c *Config) RegisterFlags(f *flag.FlagSet) {
	f.BoolVar(&c.Enabled, "enabled", true, "Whether enable auto configuration.")
	f.StringVar(&c.ApplicationID, "application-id", "seata-go", "Application id.")
	f.StringVar(&c.TxServiceGroup, "tx-service-group", "default_tx_group", "Transaction service group.")
	f.StringVar(&c.AccessKey, "access-key", "", "Used for aliyun accessKey.")
	f.StringVar(&c.SecretKey, "secret-key", "", "Used for aliyun secretKey.")
	f.BoolVar(&c.EnableAutoDataSourceProxy, "enable-auto-data-source-proxy", true, "Whether enable auto proxying of datasource bean.")
	f.StringVar(&c.DataSourceProxyMode, "data-source-proxy-mode", "AT", "Data source proxy mode.")

	c.AsyncWorkerConfig.RegisterFlagsWithPrefix("async-worker", f)
	c.TCCConfig.RegisterFlagsWithPrefix("tcc", f)
	c.ClientConfig.RegisterFlagsWithPrefix("client", f)
	c.GettyConfig.RegisterFlagsWithPrefix("getty", f)
	c.TransportConfig.RegisterFlagsWithPrefix("transport", f)
	c.RegistryConfig.RegisterFlagsWithPrefix("registry", f)
	c.ServiceConfig.RegisterFlagsWithPrefix("service", f)
}

type loaderConf struct {
	suffix string // loaderConf file extension default yaml
	path   string // loaderConf file path default ./conf/seatago.yaml
	delim  string // loaderConf file delim default .
	bytes  []byte // config bytes
	name   string // config file name
}

// Load parse config from user config path
func LoadPath(configFilePath string) *Config {
	if configFilePath == "" {
		configFilePath = os.Getenv(configFileEnvKey)
		if configFilePath == "" {
			panic("system variable SEATA_GO_CONFIG_PATH is empty")
		}
	}

	var cfg Config
	// This sets default values from flags to the config.
	// It needs to be called before parsing the config file!
	flagext.RegisterFlags(&cfg)

	conf := newLoaderConf(configFilePath)
	koan := getConfigResolver(conf)
	if err := koan.UnmarshalWithConf(configPrefix, &cfg, koanf.UnmarshalConf{Tag: yamlSuffix}); err != nil {
		panic(err)
	}
	return &cfg
}

// Load parse config from json bytes
func LoadJson(bytes []byte) *Config {
	var cfg Config
	// This sets default values from flags to the config.
	// It needs to be called before parsing the config file!
	flagext.RegisterFlags(&cfg)

	koan := getJsonConfigResolver(bytes)
	if err := koan.Unmarshal("", &cfg); err != nil {
		panic(err)
	}
	return &cfg
}

// getJsonConfigResolver get json config resolver
func getJsonConfigResolver(bytes []byte) *koanf.Koanf {
	k := koanf.New(".")
	if err := k.Load(rawbytes.Provider(bytes), json.Parser()); err != nil {
		panic(err)
	}
	return k
}

// resolverFilePath resolver file path
// eg: give a ./conf/seatago.yaml return seatago and yaml
func resolverFilePath(path string) (name, suffix string) {
	paths := strings.Split(path, "/")
	fileName := strings.Split(paths[len(paths)-1], ".")
	if len(fileName) < 2 {
		return fileName[0], yamlSuffix
	}
	return fileName[0], fileName[1]
}

// getConfigResolver get config resolver
func getConfigResolver(conf *loaderConf) *koanf.Koanf {
	var (
		k   *koanf.Koanf
		err error
	)
	if len(conf.suffix) <= 0 {
		conf.suffix = yamlSuffix
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
	case yamlSuffix, ymlSuffix:
		err = k.Load(rawbytes.Provider(bytes), yaml.Parser())
	case jsonSuffix:
		err = k.Load(rawbytes.Provider(bytes), json.Parser())
	case tomlSuffix:
		err = k.Load(rawbytes.Provider(bytes), toml.Parser())
	default:
		err = fmt.Errorf("no support %s file suffix", conf.suffix)
	}

	if err != nil {
		panic(err)
	}
	return k
}

func newLoaderConf(configFilePath string) *loaderConf {
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

// userHomeDir get gopath
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
