package tm

import (
	"flag"
	"time"
)

type TmConf struct {
	CommitRetryCount                int           `yaml:"commit-retry-count" json:"commit-retry-count,omitempty" property:"commit-retry-count" koanf:"commit-retry-count"`
	RollbackRetryCount              int           `yaml:"rollback-retry-count" json:"rollback-retry-count,omitempty" property:"rollback-retry-count" koanf:"rollback-retry-count"`
	DefaultGlobalTransactionTimeout time.Duration `yaml:"default-global-transaction-timeout" json:"default-global-transaction-timeout,omitempty" property:"default-global-transaction-timeout" koanf:"default-global-transaction-timeout"`
	DegradeCheck                    bool          `yaml:"degrade-check" json:"degrade-check,omitempty" property:"degrade-check" koanf:"degrade-check"`
	DegradeCheckPeriod              int           `yaml:"degrade-check-period" json:"degrade-check-period,omitempty" property:"degrade-check-period" koanf:"degrade-check-period"`
	DegradeCheckAllowTimes          time.Duration `yaml:"degrade-check-allow-times" json:"degrade-check-allow-times,omitempty" property:"degrade-check-allow-times" koanf:"degrade-check-allow-times"`
	InterceptorOrder                int           `yaml:"interceptor-order" json:"interceptor-order,omitempty" property:"interceptor-order" koanf:"interceptor-order"`
}

func (cfg *TmConf) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.IntVar(&cfg.CommitRetryCount, prefix+".commit-retry-count", 5, "The maximum number of retries when commit global transaction.")
	f.IntVar(&cfg.RollbackRetryCount, prefix+".rollback-retry-count", 5, "The maximum number of retries when rollback global transaction.")
	f.DurationVar(&cfg.DefaultGlobalTransactionTimeout, prefix+".default-global-transaction-timeout", 10*time.Second, "The timeout for a global transaction.")
	f.BoolVar(&cfg.DegradeCheck, prefix+".degrade-check", false, "The switch for degrade check.")
	f.IntVar(&cfg.DegradeCheckPeriod, prefix+".degrade-check-period", 2000, "The period for degrade checking.")
	f.DurationVar(&cfg.DegradeCheckAllowTimes, prefix+".degrade-check-allow-times", 10*time.Second, "The duration allowed for degrade checking.")
	f.IntVar(&cfg.InterceptorOrder, prefix+".interceptor-order", -2147482648, "The order of interceptor.")
}
