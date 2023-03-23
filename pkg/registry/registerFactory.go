package registry

// getRegister
func GetRegister(config *Config) (RegistryService, error) {
	switch config.Type {
	case "file":
	case "nacos":
		return NewNacosRegistryService(config.NacosConfig), nil
	case "etcd":
		panic("not implemented etcd register center")
	}
	return nil, nil
}
