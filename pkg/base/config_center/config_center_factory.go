package config_center

type DynamicConfigurationFactory interface {
	GetConfig() string                          //返回配置信息
	AddListener(listener ConfigurationListener) //添加配置监听

}
