package registry

import (
	"net"
)

// RegistryService registers
type RegistryService interface {

	// RegisterServiceInstance register new service to nacos
	RegisterServiceInstance(address net.TCPAddr)

	// DeRegisterServiceInstance deRegister new service to nacos
	DeRegisterServiceInstance(address net.TCPAddr)

	// GetService Get service with serviceName, groupName , clusters
	GetService(cluster string, groupName string)

	// Subscribe key=serviceName+groupName+cluster
	// Note:We call add multiple SubscribeCallback with the same key.
	Subscribe(cluster string, groupName string)

	// UnSubscribe key=serviceName+groupName+cluster
	UnSubscribe(cluster string, groupName string)
}
