package rm

import (
	"flag"
	"time"
)

type LockConfig struct {
	RetryInterval                       int           `yaml:"retry-interval" json:"retry-interval,omitempty" koanf:"retry-interval"`
	RetryTimes                          time.Duration `yaml:"retry-times" json:"retry-times,omitempty" koanf:"retry-times"`
	RetryPolicyBranchRollbackOnConflict bool          `yaml:"retry-policy-branch-rollback-on-conflict" json:"retry-policy-branch-rollback-on-conflict,omitempty" koanf:"retry-policy-branch-rollback-on-conflict"`
}

func (cfg *LockConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.IntVar(&cfg.RetryInterval, prefix+".retry-interval", 10, "The maximum number of retries when lock fail.")
	f.DurationVar(&cfg.RetryTimes, prefix+".retry-times", 30*time.Second, "The duration allowed for lock retrying.")
	f.BoolVar(&cfg.RetryPolicyBranchRollbackOnConflict, prefix+".retry-policy-branch-rollback-on-conflict", true, "The switch for lock conflict.")
}
