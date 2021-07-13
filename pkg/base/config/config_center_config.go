package config

type ConfigCenterConfig struct {
	Mode        string            `yaml:"type" json:"type,omitempty"` //类型
	NacosConfig NacosConfigCenter `yaml:"nacos" json:"nacos,omitempty"`
}

type NacosConfigCenter struct {
	ServerAddr string `yaml:"server_addr" json:"server_addr,omitempty"`
	Group      string `default:"SEATA_GROUP" yaml:"group" json:"group,omitempty"`
	Namespace  string `yaml:"namespace" json:"namespace,omitempty"`
	Cluster    string `yaml:"cluster" json:"cluster,omitempty"`
	UserName   string `yaml:"username" json:"username,omitempty"`
	Password   string `yaml:"password" json:"password,omitempty"`
	DataId     string `default:"seata" yaml:"data_id" json:"data_id,omitempty"`
}
