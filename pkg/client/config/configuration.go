package config

import (
	"io"
	"io/ioutil"
	"os"
	"time"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/opentrx/seata-golang/v2/pkg/util/log"
	"github.com/opentrx/seata-golang/v2/pkg/util/parser"
)

var configuration *Configuration

// Configuration client configuration
type Configuration struct {
	Addressing       string `yaml:"addressing" json:"addressing"`
	ServerAddressing string `yaml:"serverAddressing" json:"serverAddressing"`

	TMConfig TMConfig `yaml:"tm" json:"tm,omitempty"`

	ATConfig ATConfig `yaml:"at" json:"at,omitempty"`

	ClientParameters struct {
		Time                time.Duration `yaml:"time"`
		Timeout             time.Duration `yaml:"timeout"`
		PermitWithoutStream bool          `yaml:"permitWithoutStream"`
	} `yaml:"clientParameters"`

	ClientTLS struct {
		Enable       bool   `yaml:"enable"`
		CertFilePath string `yaml:"certFilePath"`
		ServerName   string `yaml:"serverName"`
	} `yaml:"clientTLS"`

	Log struct {
		LogPath  string    `yaml:"logPath"`
		LogLevel log.Level `yaml:"logLevel"`
	} `yaml:"log"`
}

// TMConfig
type TMConfig struct {
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

// GetClientParameters used to config grpc connection keep alive
func GetClientParameters() keepalive.ClientParameters {
	return configuration.GetClientParameters()
}

// GetClientParameters used to config grpc connection keep alive
func (configuration *Configuration) GetClientParameters() keepalive.ClientParameters {
	cp := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             time.Second,
		PermitWithoutStream: true,
	}
	if configuration.ClientParameters.Time > 0 {
		cp.Time = configuration.ClientParameters.Timeout
	}
	if configuration.ClientParameters.Timeout > 0 {
		cp.Timeout = configuration.ClientParameters.Timeout
	}
	cp.PermitWithoutStream = configuration.ClientParameters.PermitWithoutStream
	return cp
}

func (configuration *Configuration) GetClientTLS() credentials.TransportCredentials {
	if !configuration.ClientTLS.Enable {
		return nil
	}
	cred, err := credentials.NewClientTLSFromFile(configuration.ClientTLS.CertFilePath, configuration.ClientTLS.ServerName)
	if err != nil {
		log.Fatalf("%v using TLS failed", err)
	}
	return cred
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

	config, err := parse(fp)
	if err != nil {
		log.Fatalf("error parsing %s: %v", configurationPath, err)
	}

	defer func() {
		err = fp.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	configuration = config
	return configuration
}

func SetConfiguration(config *Configuration) {
	configuration = config
}
