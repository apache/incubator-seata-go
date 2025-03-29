package rocketmq

type Config struct {
	Addr      string
	Namespace string
	GroupName string
}

type ConfigBuilder struct {
	config *Config
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &Config{},
	}
}

func (b *ConfigBuilder) WithAddr(addr string) *ConfigBuilder {
	b.config.Addr = addr
	return b
}

func (b *ConfigBuilder) WithNamespace(namespace string) *ConfigBuilder {
	b.config.Namespace = namespace
	return b
}

func (b *ConfigBuilder) WithGroupName(groupName string) *ConfigBuilder {
	b.config.GroupName = groupName
	return b
}

func (b *ConfigBuilder) Build() Config {
	return *b.config
}
