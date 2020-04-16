package config

type TMConfig struct {
	CommitRetryCount int32
	RollbackRetryCount int32
}

var tmConfig TMConfig

func GetTmConfig() TMConfig {
	return TMConfig{
		CommitRetryCount:   5,
		RollbackRetryCount: 5,
	}
}