package config

import (
	"time"
)

import (
	"github.com/pkg/errors"
)

type ATConfig struct {
	DSN                 string `yaml:"dsn" json:"dsn,omitempty"`
	ReportRetryCount    int    `default:"5" yaml:"report_retry_count" json:"report_retry_count,omitempty"`
	ReportSuccessEnable bool   `default:"false" yaml:"report_success_enable" json:"report_success_enable,omitempty"`
	LockRetryItv        string `default:"10ms" yaml:"lock_retry_interval" json:"lock_retry_interval,omitempty"`
	LockRetryInterval   time.Duration
	LockRetryTimes      int `default:"30" yaml:"lock_retry_times" json:"lock_retry_times,omitempty"`
}

func (c *ATConfig) CheckValidity() error {
	var err error

	if c.LockRetryInterval, err = time.ParseDuration(c.LockRetryItv); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(LockRetryInterval{%#v})", c.LockRetryItv)
	}
	return nil
}
