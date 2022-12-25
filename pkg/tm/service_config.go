package tm

import (
	"flag"
)

var (
	serviceConfig *ServiceConfig
)

type ServiceConfig struct {
	VgroupMapping            map[string]string `yaml:"vgroup-mapping" json:"vgroup-mapping" koanf:"vgroup-mapping"`
	Grouplist                map[string]string `yaml:"grouplist" json:"grouplist" koanf:"grouplist"`
	EnableDegrade            bool              `yaml:"enable-degrade" json:"enable-degrade" koanf:"enable-degrade"`
	DisableGlobalTransaction bool              `yaml:"disable-global-transaction" json:"disable-global-transaction" koanf:"disable-global-transaction"`
}

func (cfg *ServiceConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.EnableDegrade, prefix+".enable-degrade", false, "degrade current not support.")
	f.BoolVar(&cfg.DisableGlobalTransaction, prefix+".disable-global-transaction", false, "disable globalTransaction.")
}

func NewServiceConfig(cfg *ServiceConfig) {
	serviceConfig = cfg
}
func GetServiceConfig() (cfg *ServiceConfig) {
	return serviceConfig
}
