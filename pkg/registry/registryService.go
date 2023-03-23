package registry

type RegistryService interface {

	//Register
	RegisterServiceInstance(config Config)

	//DeRegister
	DeRegisterServiceInstance(config Config)
	//
	////BatchRegister
	//batchRegisterServiceInstance(config Config)

	//Get service with serviceName, groupName , clusters
	GetService(config Config)

	//SelectAllInstance
	//GroupName=DEFAULT_GROUP
	SelectAllInstances(config Config)

	//SelectInstances only return the instances of healthy=${HealthyOnly},enable=true and weight>0
	//ClusterName=DEFAULT,GroupName=DEFAULT_GROUP
	SelectInstances(config Config)

	//SelectOneHealthyInstance return one instance by WRR strategy for load balance
	//And the instance should be health=true,enable=true and weight>0
	//ClusterName=DEFAULT,GroupName=DEFAULT_GROUP
	SelectOneHealthyInstance(config Config)

	//Subscribe key=serviceName+groupName+cluster
	//Note:We call add multiple SubscribeCallback with the same key.
	Subscribe(config Config)

	UpdateServiceInstance(config Config)

	// UnSubscribe
	UnSubscribe(config Config)

	//GeAllService will get the list of service name
	//NameSpace default value is public.If the client set the namespaceId, NameSpace will use it.
	//GroupName default value is DEFAULT_GROUP
	GetAllService(config Config, pageInfo PageInfo)
}
type PageInfo struct {
	pageSize uint32
	pageNo   uint32
}
