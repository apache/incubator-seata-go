package registry

type Address struct {
	IP   string
	Port uint64
}

// Registry Extension - Registry
type Registry interface {
	//注册服务
	Register(addr *Address) error
	//取消注册
	UnRegister(addr *Address) error
	//查询服务地址
	Lookup() ([]string, error)
	//订阅
	Subscribe(EventListener) error
	//取消订阅
	UnSubscribe(EventListener) error

	Stop()
}

//订阅获取到的服务信息
type Service struct {
	EventType uint32 // EventType: 0 => PUT, 1 => DELETE
	IP        string
	Port      uint64
	Name      string
}

type EventListener interface {
	OnEvent(service []*Service) error
}
