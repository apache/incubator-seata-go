package config

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"
	"time"
)

var (
	conf ServerConfig
)

func GetServerConfig() ServerConfig {
	return conf
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


func InitConf(confFile string) error {
	var err error

	if confFile == "" {
		return errors.WithMessagef(err,fmt.Sprintf("application configure file name is nil"))
	}
	if path.Ext(confFile) != ".yml" {
		return errors.WithMessagef(err,fmt.Sprintf("application configure file name{%v} suffix must be .yml", confFile))
	}

	conf = ServerConfig{}
	confFileStream, err := ioutil.ReadFile(confFile)
	if err != nil {
		return errors.WithMessagef(err,fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", confFile, err))
	}
	err = yaml.Unmarshal(confFileStream, &conf)
	if err != nil {
		return errors.WithMessagef(err,fmt.Sprintf("yaml.Unmarshal() = error:%s", err))
	}

	if conf.TimeoutRetryPeriod, err = time.ParseDuration(conf.TimeoutRetryPrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(TimeoutRetryPrd{%#v})", conf.TimeoutRetryPrd)
	}

	conf.RollbackingRetryPeriod, err = time.ParseDuration(conf.RollbackingRetryPrd)
	if err != nil {
		return errors.WithMessagef(err,"time.ParseDuration(RollbackingRetryPrd{%#v})", conf.RollbackingRetryPrd)
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

	conf.GettyConfig.SessionTimeout, err = time.ParseDuration(conf.GettyConfig.SessionTmt)
	if err != nil {
		return errors.WithMessagef(err,fmt.Sprintf("time.ParseDuration(SessionTimeout{%#v}) = error{%v}", conf.GettyConfig.SessionTmt, err))
	}
	if conf.GettyConfig.GettySessionParam.KeepAlivePeriod, err = time.ParseDuration(conf.GettyConfig.GettySessionParam.KeepAlivePrd); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(KeepAlivePeriod{%#v})", conf.GettyConfig.GettySessionParam.KeepAlivePrd)
	}

	if conf.GettyConfig.GettySessionParam.TcpReadTimeout, err = time.ParseDuration(conf.GettyConfig.GettySessionParam.TcpReadTmt); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(TcpReadTimeout{%#v})", conf.GettyConfig.GettySessionParam.TcpReadTmt)
	}

	if conf.GettyConfig.GettySessionParam.TcpWriteTimeout, err = time.ParseDuration(conf.GettyConfig.GettySessionParam.TcpWriteTmt); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(TcpWriteTimeout{%#v})", conf.GettyConfig.GettySessionParam.TcpWriteTmt)
	}

	if conf.GettyConfig.GettySessionParam.WaitTimeout, err = time.ParseDuration(conf.GettyConfig.GettySessionParam.WaitTmt); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(WaitTimeout{%#v})", conf.GettyConfig.GettySessionParam.WaitTmt)
	}

	storeConfig = conf.StoreConfig
	return nil
}
