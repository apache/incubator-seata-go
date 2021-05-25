package registry

type Address struct {
	IP   string
	Port int
}

// Registry Extension - Registry
type Registry interface {
	//注册服务
	Register(addr *Address) error
	//取消注册
	UnRegister(addr *Address) error
}
