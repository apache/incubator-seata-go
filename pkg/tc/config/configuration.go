package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/opentrx/seata-golang/v2/pkg/util/log"
	"github.com/opentrx/seata-golang/v2/pkg/util/parser"
)

// Configuration is a versioned registry configuration, intended to be provided by a yaml file, and
// optionally modified by environment variables.
//
// Note that yaml field names should never include _ characters, since this is the separator used
// in environment variable names.
type Configuration struct {
	Server struct {
		Port                             int   `yaml:"port"`
		MaxCommitRetryTimeout            int64 `yaml:"maxCommitRetryTimeout"`
		MaxRollbackRetryTimeout          int64 `yaml:"maxRollbackRetryTimeout"`
		RollbackRetryTimeoutUnlockEnable bool  `yaml:"rollbackRetryTimeoutUnlockEnable"`

		AsyncCommittingRetryPeriod time.Duration `yaml:"asyncCommittingRetryPeriod"`
		CommittingRetryPeriod      time.Duration `yaml:"committingRetryPeriod"`
		RollingBackRetryPeriod     time.Duration `yaml:"rollingBackRetryPeriod"`
		TimeoutRetryPeriod         time.Duration `yaml:"timeoutRetryPeriod"`
	} `yaml:"server"`

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

	// Storage is the configuration for the storage driver
	Storage Storage `yaml:"storage"`

	Log struct {
		LogPath  string       `yaml:"logPath"`
		LogLevel log.LogLevel `yaml:"logLevel"`
	} `yaml:"log"`
}

func (configuration *Configuration) GetEnforcementPolicy() keepalive.EnforcementPolicy {
	return keepalive.EnforcementPolicy{
		MinTime:             configuration.EnforcementPolicy.MinTime,
		PermitWithoutStream: configuration.EnforcementPolicy.PermitWithoutStream,
	}
}

func (configuration *Configuration) GetServerParameters() keepalive.ServerParameters {
	return keepalive.ServerParameters{
		MaxConnectionIdle:     configuration.ServerParameters.MaxConnectionIdle,
		MaxConnectionAge:      configuration.ServerParameters.MaxConnectionAge,
		MaxConnectionAgeGrace: configuration.ServerParameters.MaxConnectionAgeGrace,
		Time:                  configuration.ServerParameters.Time,
		Timeout:               configuration.ServerParameters.Timeout,
	}
}

// Parameters defines a key-value parameters mapping
type Parameters map[string]interface{}

// Storage defines the configuration for registry object storage
type Storage map[string]Parameters

// Type returns the storage driver type, such as filesystem or s3
func (storage Storage) Type() string {
	var storageType []string

	// Return only key in this map
	for k := range storage {
		storageType = append(storageType, k)
	}
	if len(storageType) > 1 {
		panic("multiple storage drivers specified in configuration or environment: " + strings.Join(storageType, ", "))
	}
	if len(storageType) == 1 {
		return storageType[0]
	}
	return ""
}

// Parameters returns the Parameters map for a Storage configuration
func (storage Storage) Parameters() Parameters {
	return storage[storage.Type()]
}

// setParameter changes the parameter at the provided key to the new value
func (storage Storage) setParameter(key string, value interface{}) {
	storage[storage.Type()][key] = value
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
// Unmarshals a single item map into a Storage or a string into a Storage type with no parameters
func (storage *Storage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var storageMap map[string]Parameters
	err := unmarshal(&storageMap)
	if err == nil {
		if len(storageMap) > 1 {
			types := make([]string, 0, len(storageMap))
			for k := range storageMap {
				types = append(types, k)

			}

			if len(types) > 1 {
				return fmt.Errorf("must provide exactly one storage type. Provided: %v", types)
			}
		}
		*storage = storageMap
		return nil
	}

	var storageType string
	err = unmarshal(&storageType)
	if err == nil {
		*storage = Storage{storageType: Parameters{}}
		return nil
	}

	return err
}

// MarshalYAML implements the yaml.Marshaler interface
func (storage Storage) MarshalYAML() (interface{}, error) {
	if storage.Parameters() == nil {
		return storage.Type(), nil
	}
	return map[string]Parameters(storage), nil
}

// Parse parses an input configuration yaml document into a Configuration struct
//
// Environment variables may be used to override configuration parameters other than version,
// following the scheme below:
// Configuration.Abc may be replaced by the value of SEATA_ABC,
// Configuration.Abc.Xyz may be replaced by the value of SEATA_ABC_XYZ, and so forth
func Parse(rd io.Reader) (*Configuration, error) {
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
