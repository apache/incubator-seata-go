package config

type TMConfig struct {
	CommitRetryCount                uint16 `default:"5" yaml:"commit_retry_count" json:"commit_retry_count,omitempty"`
	RollbackRetryCount              uint16 `default:"5" yaml:"rollback_retry_count" json:"rollback_retry_count,omitempty"`
	DefaultGlobalTransactionTimeout uint16 `default:"60000" yaml:"default_global_transaction_timeout" json:"default_global_transaction_timeout,omitempty"`
	DegradeCheck                    bool   `default:"false" yaml:"degrade_check" json:"degrade_check,omitempty"`
	DegradeCheckAllowTimes          uint16 `default:"10" yaml:"degrade_check_allow_times" json:"degrade_check_allow_times,omitempty"`
	DegradeCheckPeriod              uint16 `default:"2000" yaml:"degrade_check_period" json:"degrade_check_period,omitempty"`
}

func GetDefaultTmConfig() TMConfig {
	return TMConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 60000,
		DegradeCheck:                    false,
		DegradeCheckAllowTimes:          10,
		DegradeCheckPeriod:              2000,
	}
}
