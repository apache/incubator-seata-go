package registry

import (
	"net"
)

type RegistryService interface {

	// RegisterServiceInstance register new service to nacos
	RegisterServiceInstance(address net.TCPAddr)

	// DeRegisterServiceInstance deRegister new service to nacos
	DeRegisterServiceInstance(address net.TCPAddr)

	// GetService Get service with serviceName, groupName , clusters
	GetService(cluster string, groupName string)

	// SelectAllInstances SelectAllInstance 	GroupName=DEFAULT_GROUP
	SelectAllInstances(cluster string, groupName string)

	// SelectInstances only return the instances of healthy=${HealthyOnly},enable=true and weight>0  ClusterName=DEFAULT,GroupName=DEFAULT_GROUP
	SelectInstances(cluster string, groupName string)

	// SelectOneHealthyInstance return one instance by WRR strategy for load balance And the instance should be health=true,enable=true and weight>0
	SelectOneHealthyInstance(cluster string, groupName string)

	// Subscribe key=serviceName+groupName+cluster
	// Note:We call add multiple SubscribeCallback with the same key.
	Subscribe(cluster string, groupName string)

	// UnSubscribe key=serviceName+groupName+cluster
	UnSubscribe(cluster string, groupName string)

	UpdateServiceInstance(address net.TCPAddr, cluster string, groupName string)

	// GetAllService will get the list of service name
	//NameSpace default value is public.If the client set the namespaceId, NameSpace will use it.
	GetAllService(groupName string, pageInfo PageInfo)
}
type PageInfo struct {
	pageSize uint32
	pageNo   uint32
}
