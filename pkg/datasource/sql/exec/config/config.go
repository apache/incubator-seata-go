package config

import (
	"flag"

	"github.com/seata/seata-go/pkg/datasource/sql/exec/at"
)

func Init(config Config) {
	at.ATConfig = config.ATExecutor
}

type Config struct {
	ATExecutor at.Config `yaml:"at" json:"at" koanf:"at"`
}

func (c *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	c.ATExecutor.RegisterFlagsWithPrefix(prefix+".at", f)
}
