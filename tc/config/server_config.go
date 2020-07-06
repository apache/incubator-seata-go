package config

import (
	"fmt"
	"github.com/go-xorm/xorm"
	"io/ioutil"
	"path"
	"time"
)

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	conf ServerConfig
)

func GetServerConfig() ServerConfig {
	return conf
}

func GetStoreConfig() StoreConfig {
	return conf.StoreConfig
}

type ServerConfig struct {
	Host                             string `default:"127.0.0.1" yaml:"host" json:"host,omitempty"`
	Port                             string `default:"8091" yaml:"port" json:"port,omitempty"`
	MaxRollbackRetryTimeout          int64  `default:"-1" yaml:"max_rollback_retry_timeout" json:"max_rollback_retry_timeout,omitempty"`
	RollbackRetryTimeoutUnlockEnable bool   `default:"false" yaml:"rollback_retry_timeout_unlock_enable" json:"rollback_retry_timeout_unlock_enable,omitempty"`
	MaxCommitRetryTimeout            int64  `default:"-1" yaml:"max_commit_retry_timeout" json:"max_commit_retry_timeout,omitempty"`
	TimeoutRetryPrd                  string `default:"1s" yaml:"timeout_retry_period" json:"timeout_retry_period,omitempty"`
	TimeoutRetryPeriod               time.Duration
	RollbackingRetryPrd              string `default:"1s" yaml:"rollbacking_retry_period" json:"rollbacking_retry_period,omitempty"`
	RollbackingRetryPeriod           time.Duration
	CommittingRetryPrd               string `default:"1s" yaml:"committing_retry_period" json:"committing_retry_period,omitempty"`
	CommittingRetryPeriod            time.Duration
	AsynCommittingRetryPrd           string `default:"1s" yaml:"aync_committing_retry_period" json:"aync_committing_retry_period,omitempty"`
	AsynCommittingRetryPeriod        time.Duration
	LogDeletePrd                     string `default:"24h" yaml:"log_delete_period" json:"log_delete_period,omitempty"`
	LogDeletePeriod                  time.Duration
	GettyConfig                      GettyConfig `required:"true" yaml:"getty_config" json:"getty_config,omitempty"`
	UndoConfig                       UndoConfig  `required:"true" yaml:"undo_config" json:"undo_config,omitempty"`
	StoreConfig                      StoreConfig `required:"true" yaml:"store_config" json:"store_config,omitempty"`
}

func (c *ServerConfig) CheckValidity() error {
	var err error
	if conf.TimeoutRetryPeriod, err = time.ParseDuration(conf.TimeoutRetryPrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(TimeoutRetryPrd{%#v})", conf.TimeoutRetryPrd)
	}

	if conf.RollbackingRetryPeriod, err = time.ParseDuration(conf.RollbackingRetryPrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(RollbackingRetryPrd{%#v})", conf.RollbackingRetryPrd)
	}

	if conf.CommittingRetryPeriod, err = time.ParseDuration(conf.CommittingRetryPrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(CommittingRetryPrd{%#v})", conf.CommittingRetryPrd)
	}

	if conf.AsynCommittingRetryPeriod, err = time.ParseDuration(conf.AsynCommittingRetryPrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(AsynCommittingRetryPrd{%#v})", conf.AsynCommittingRetryPrd)
	}

	if conf.LogDeletePeriod, err = time.ParseDuration(conf.LogDeletePrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(LogDeletePrd{%#v})", conf.LogDeletePrd)
	}

	return errors.WithStack(c.GettyConfig.CheckValidity())
}

func InitConf(confFile string) error {
	var err error

	if confFile == "" {
		return errors.WithMessagef(err, fmt.Sprintf("application configure file name is nil"))
	}
	if path.Ext(confFile) != ".yml" {
		return errors.WithMessagef(err, fmt.Sprintf("application configure file name{%v} suffix must be .yml", confFile))
	}

	conf = ServerConfig{}
	confFileStream, err := ioutil.ReadFile(confFile)
	if err != nil {
		return errors.WithMessagef(err, fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", confFile, err))
	}
	err = yaml.Unmarshal(confFileStream, &conf)
	if err != nil {
		return errors.WithMessagef(err, fmt.Sprintf("yaml.Unmarshal() = error:%s", err))
	}

	(&conf).CheckValidity()
	if conf.StoreConfig.StoreMode == "db" && conf.StoreConfig.DBStoreConfig.DSN != "" {
		engine, err := xorm.NewEngine("mysql", conf.StoreConfig.DBStoreConfig.DSN)
		if err != nil {
			panic(err)
		}
		conf.StoreConfig.DBStoreConfig.Engine = engine
	}

	return nil
}
