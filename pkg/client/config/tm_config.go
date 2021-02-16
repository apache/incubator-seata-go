package config

type TMConfig struct {
	CommitRetryCount   int32 `default:"5" yaml:"commit_retry_count" json:"commit_retry_count,omitempty"`
	RollbackRetryCount int32 `default:"5" yaml:"rollback_retry_count" json:"rollback_retry_count,omitempty"`
}

func GetDefaultTmConfig() TMConfig {
	return TMConfig{
		CommitRetryCount:   5,
		RollbackRetryCount: 5,
	}
}
