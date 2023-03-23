package registry

import (
	"flag"
	"github.com/seata/seata-go/pkg/util/flagext"
)

type NacosConfig struct {
	NamespaceId          string            `yaml:"namespaceId" json:"namespaceId" koanf:"namespaceId"` // the namespaceId of Nacos
	ServerAddr           string            `yaml:"server-addr" json:"server-addr" koanf:"server-addr"`
	GroupName            string            `yaml:"groupName" json:"groupName" koanf:"groupName"`
	Username             string            `yaml:"username" json:"username" koanf:"username"`    // the username for nacos auth
	Password             string            `yaml:"password" json:"password" koanf:"password"`    // the password for nacos auth
	AccessKey            string            `yaml:"accessKey" json:"accessKey" koanf:"accessKey"` // the AccessKey for ACM & KMS
	SecretKey            string            `yaml:"secretKey" json:"secretKey" koanf:"secretKey"` // the SecretKey for ACM & KMS
	Port                 uint64            `yaml:"port" json:"port" koanf:"port"`
	TimeoutMs            uint64            `yaml:"timeoutMs" json:"timeoutMs" koanf:"timeoutMs"`                               // timeout for requesting Nacos server, default value is 10000ms
	NotLoadCacheAtStart  bool              `yaml:"notLoadCacheAtStart" json:"notLoadCacheAtStart" koanf:"notLoadCacheAtStart"` // not to load persistent nacos service info in CacheDir at start time
	LogDir               string            `yaml:"logDir" json:"logDir" koanf:"logDir"`                                        // the directory for log, default is current path
	CacheDir             string            `yaml:"GroupName" json:"GroupName" koanf:"GroupName"`                               // the directory for persist nacos service info,default value is current path
	LogLevel             string            `yaml:"logLevel" json:"logLevel" koanf:"logLevel"`                                  // the level of log, it's must be debug,info,warn,error, default value is info
	ServiceName          string            `yaml:"serviceName" json:"serviceName" koanf:"serviceName"`
	Cluster              string            `yaml:"cluster" json:"cluster" koanf:"cluster"`
	HealthyOnly          bool              `yaml:"healthyOnly" json:"healthyOnly" koanf:"healthyOnly"`
	Weight               float64           `yaml:"weight" json:"weight" koanf:"weight"`
	Enable               bool              `yaml:"enable" json:"enable" koanf:"enable"`
	Healthy              bool              `yaml:"healthy" json:"healthy" koanf:"healthy"`
	Ephemeral            bool              `yaml:"ephemeral" json:"ephemeral" koanf:"ephemeral"`
	MetaData             flagext.StringMap `yaml:"metaData" json:"metaData" koanf:"metaData"`
	Endpoint             string            `yaml:"endpoint" json:"endpoint" koanf:"endpoint"`                                     // the endpoint for ACM. https://help.aliyun.com/document_detail/130146.html
	RegionId             string            `yaml:"regionId" json:"regionId" koanf:"regionId"`                                     // the regionId for ACM & KMS
	OpenKMS              bool              `yaml:"openKMS" json:"openKMS" koanf:"openKMS"`                                        // it's to open KMS, default is false. https://help.aliyun.com/product/28933.html  to enable encrypt/decrypt, DataId should be start with "cipher-"
	UpdateThreadNum      int               `yaml:"updateThreadNum" json:"updateThreadNum" koanf:"updateThreadNum"`                // the number of goroutine for update nacos service info,default value is 20
	UpdateCacheWhenEmpty bool              `yaml:"updateCacheWhenEmpty" json:"updateCacheWhenEmpty" koanf:"updateCacheWhenEmpty"` // update cache when get empty service instance from server
	RotateTime           string            `yaml:"rotateTime" json:"rotateTime" koanf:"rotateTime"`                               // the rotate time for log, eg: 30m, 1h, 24h, default is 24h
	MaxAge               int64             `yaml:"maxAge" json:"maxAge" koanf:"maxAge"`                                           // the max age of a log file, default value is 3

}

func (c *NacosConfig) NacosFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&c.NamespaceId, prefix+".namespace", "public", "nacos registry center namespace")
	f.StringVar(&c.ServerAddr, prefix+".serverAddr", "127.0.0.1", "nacos registry center server address")
	f.StringVar(&c.GroupName, prefix+".group", "DEFAULT_GROUP", "nacos registry center group name")
	f.StringVar(&c.Username, prefix+".username", "nacos", "nacos registry center username")
	f.StringVar(&c.Password, prefix+".password", "nacos", "nacos registry center password")
	f.StringVar(&c.AccessKey, prefix+".accessKey", "", "nacos registry center access key")
	f.StringVar(&c.SecretKey, prefix+".secretKey", "", "nacos registry center secret key")
	f.Uint64Var(&c.Port, prefix+".port", 8848, "nacos registry center port")
	f.Uint64Var(&c.TimeoutMs, prefix+".timeoutMs", 5000, "nacos registry center timeoutMs")
	f.BoolVar(&c.NotLoadCacheAtStart, prefix+".notLoadCacheAtStart", true, "nacos registry center NotLoadCacheAtStart")
	f.StringVar(&c.LogDir, prefix+".logDir", "/tmp/nacos/log", "nacos registry center LogDir")
	f.StringVar(&c.CacheDir, prefix+".cacheDir", "/tmp/nacos/cache", "nacos registry center CacheDir")
	f.BoolVar(&c.HealthyOnly, prefix+".healthyOnly", true, "nacos registry center HealthyOnly")
	f.BoolVar(&c.Healthy, prefix+".healthy", true, "nacos registry center Healthy")
	f.Float64Var(&c.Weight, prefix+".weight", 10, "nacos registry center Weight")
	f.BoolVar(&c.Enable, prefix+".enable", true, "nacos registry center Enable")
	f.BoolVar(&c.Ephemeral, prefix+".ephemeral", true, "nacos registry center Ephemeral")
	f.StringVar(&c.Cluster, prefix+".cluster", "DEFAULT", "nacos registry center cluster")
	f.StringVar(&c.Endpoint, prefix+".endpoint", "", "nacos registry center Endpoint")
	f.StringVar(&c.RegionId, prefix+".regionId", "", "nacos registry center RegionId")
	f.BoolVar(&c.OpenKMS, prefix+".openKMS", true, "nacos registry center OpenKMS")
	f.IntVar(&c.UpdateThreadNum, prefix+".updateThreadNum", 10, "nacos registry center UpdateThreadNum")
	f.BoolVar(&c.UpdateCacheWhenEmpty, prefix+".updateCacheWhenEmpty", true, "nacos registry center UpdateCacheWhenEmpty")
	f.StringVar(&c.RotateTime, prefix+".rotateTime", "", "nacos registry center RotateTime")
	f.Int64Var(&c.MaxAge, prefix+".maxAge", 10, "nacos registry center MaxAge")
	f.Var(&c.MetaData, prefix+".metaData", "nacos registry center metaData")
}

type Config struct {
	Type        string      `yaml:"type" json:"type" koanf:"type"`
	FileConfig  string      `yaml:"file.name" json:"file.name" koanf:"file.name"`
	NacosConfig NacosConfig `yaml:"nacos" json:"nacos" koanf:"nacos"`
}

func (c *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&c.Type, prefix+".type", "file", "registry center type")
	f.StringVar(&c.FileConfig, prefix+".file.name", "", "registry center file type path")
	c.NacosConfig.NacosFlagsWithPrefix(prefix+".nacos", f)
}
