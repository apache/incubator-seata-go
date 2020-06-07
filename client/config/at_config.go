package config

import (
	"github.com/pkg/errors"
	"time"
)

type ATConfig struct {
	DSN                 string `yaml:"dsn" json:"dsn,omitempty"`
	Active              int    `yaml:"active" json:"active,omitempty"`
	Idle                int    `yaml:"idle" json:"idle,omitempty"`
	IdleTmt             string `yaml:"idle_timeout" json:"idle_timeout,omitempty"`
	IdleTimeout         time.Duration
	ReportRetryCount    int  `default:"5" yaml:"report_retry_count" json:"report_retry_count,omitempty"`
	ReportSuccessEnable bool `default:"false" yaml:"report_success_enable" json:"report_success_enable,omitempty"`
}

func (c *ATConfig) CheckValidity() error {
	var err error

	if c.IdleTimeout, err = time.ParseDuration(c.IdleTmt); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(IdleTimeOut{%#v})", c.IdleTmt)
	}
	return err
}