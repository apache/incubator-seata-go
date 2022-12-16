package transport

import (
	"flag"
	"time"
)

type ShutdownConfig struct {
	Wait time.Duration `yaml:"wait" json:"wait" konaf:"wait"`
}

func (cfg *ShutdownConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.Wait, prefix+".wait", 3*time.Second, "Shutdown wait time.")
}
