package config

import (
	"io"
	"io/ioutil"
	"os"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/opentrx/seata-golang/v2/pkg/util/log"
	"github.com/opentrx/seata-golang/v2/pkg/util/parser"
)

var configuration *Configuration

// Configuration client configuration
type Configuration struct {
	Port             int32  `yaml:"port" json:"port"`
	Addressing       string `yaml:"addressing" json:"addressing"`
	ServerAddressing string `yaml:"serverAddressing" json:"serverAddressing"`

	TMConfig TMConfig `yaml:"tm" json:"tm,omitempty"`

	ATConfig ATConfig `yaml:"at" json:"at,omitempty"`

	EnforcementPolicy struct {
		MinTime             time.Duration `yaml:"minTime"`
		PermitWithoutStream bool          `yaml:"permitWithoutStream"`
	} `yaml:"enforcementPolicy"`

	ServerParameters struct {
		MaxConnectionIdle     time.Duration `yaml:"maxConnectionIdle"`
		MaxConnectionAge      time.Duration `yaml:"maxConnectionAge"`
		MaxConnectionAgeGrace time.Duration `yaml:"maxConnectionAgeGrace"`
		Time                  time.Duration `yaml:"time"`
		Timeout               time.Duration `yaml:"timeout"`
	} `yaml:"serverParameters"`

	ClientParameters struct {
		Time                time.Duration `yaml:"time"`
		Timeout             time.Duration `yaml:"timeout"`
		PermitWithoutStream bool          `yaml:"permitWithoutStream"`
	} `yaml:"clientParameters"`

	Log struct {
		LogPath  string       `yaml:"logPath"`
		LogLevel log.LogLevel `yaml:"logLevel"`
	} `yaml:"log"`
}

// TMConfig
type TMConfig   struct {
	CommitRetryCount   int32 `default:"5" yaml:"commitRetryCount" json:"commitRetryCount,omitempty"`
	RollbackRetryCount int32 `default:"5" yaml:"rollbackRetryCount" json:"rollbackRetryCount,omitempty"`
}

// ATConfig
type ATConfig struct {
	DSN                 string        `yaml:"dsn" json:"dsn,omitempty"`
	ReportRetryCount    int           `default:"5" yaml:"reportRetryCount" json:"reportRetryCount,omitempty"`
	ReportSuccessEnable bool          `default:"false" yaml:"reportSuccessEnable" json:"reportSuccessEnable,omitempty"`
	LockRetryInterval   time.Duration `default:"10ms" yaml:"lockRetryInterval" json:"lockRetryInterval,omitempty"`
	LockRetryTimes      int           `default:"30" yaml:"lockRetryTimes" json:"lockRetryTimes,omitempty"`
}

// GetTMConfig return TMConfig
func GetTMConfig() TMConfig {
	return configuration.TMConfig
}

// GetATConfig return ATConfig
func GetATConfig() ATConfig {
	return configuration.ATConfig
}

// GetEnforcementPolicy used to config grpc connection keep alive
func GetEnforcementPolicy() keepalive.EnforcementPolicy {
	return configuration.GetEnforcementPolicy()
}

// GetServerParameters used to config grpc connection keep alive
func GetServerParameters() keepalive.ServerParameters {
	return configuration.GetServerParameters()
}

// GetClientParameters used to config grpc connection keep alive
func GetClientParameters() keepalive.ClientParameters {
	return configuration.GetClientParameters()
}

// GetEnforcementPolicy used to config grpc connection keep alive
func (configuration *Configuration) GetEnforcementPolicy() keepalive.EnforcementPolicy {
	return keepalive.EnforcementPolicy{
		MinTime:             configuration.EnforcementPolicy.MinTime,
		PermitWithoutStream: configuration.EnforcementPolicy.PermitWithoutStream,
	}
}

// GetServerParameters used to config grpc connection keep alive
func (configuration *Configuration) GetServerParameters() keepalive.ServerParameters {
	return keepalive.ServerParameters{
		MaxConnectionIdle:     configuration.ServerParameters.MaxConnectionIdle,
		MaxConnectionAge:      configuration.ServerParameters.MaxConnectionAge,
		MaxConnectionAgeGrace: configuration.ServerParameters.MaxConnectionAgeGrace,
		Time:                  configuration.ServerParameters.Time,
		Timeout:               configuration.ServerParameters.Timeout,
	}
}

// GetClientParameters used to config grpc connection keep alive
func (configuration *Configuration) GetClientParameters() keepalive.ClientParameters {
	return keepalive.ClientParameters{
		Time:                configuration.ClientParameters.Time,
		Timeout:             configuration.ClientParameters.Timeout,
		PermitWithoutStream: configuration.ClientParameters.PermitWithoutStream,
	}
}

// Parse parses an input configuration yaml document into a Configuration struct
//
// Environment variables may be used to override configuration parameters other than version,
// following the scheme below:
// Configuration.Abc may be replaced by the value of SEATA_ABC,
// Configuration.Abc.Xyz may be replaced by the value of SEATA_ABC_XYZ, and so forth
func parse(rd io.Reader) (*Configuration, error) {
	in, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	p := parser.NewParser("seata")

	config := new(Configuration)
	err = p.Parse(in, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// InitConfiguration init configuration from a file path
func InitConfiguration(configurationPath string) *Configuration {
	fp, err := os.Open(configurationPath)
	if err != nil {
		log.Fatalf("open configuration file fail, %v", err)
	}

	defer fp.Close()

	config, err := parse(fp)
	if err != nil {
		log.Fatalf("error parsing %s: %v", configurationPath, err)
	}

	configuration = config
	return configuration
}