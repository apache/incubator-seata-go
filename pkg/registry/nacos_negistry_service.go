package registry //nolint:typecheck

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"strconv"
)

type NacosRegistryService struct {
	client naming_client.INamingClient
}

func NewNacosRegistryService(config NacosConfig) *NacosRegistryService {
	//create ServerConfig
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(config.ServerAddr, config.Port, constant.WithContextPath("/nacos")),
	}

	//create ClientConfig
	cc := *constant.NewClientConfig(
		constant.WithNamespaceId(config.NamespaceId),
		constant.WithTimeoutMs(config.TimeoutMs),
		constant.WithNotLoadCacheAtStart(config.NotLoadCacheAtStart),
		constant.WithLogDir(config.LogDir),
		constant.WithCacheDir(config.CacheDir),
		constant.WithLogLevel(config.LogLevel),
		constant.WithAccessKey(config.AccessKey),
		constant.WithSecretKey(config.SecretKey),
		constant.WithEndpoint(config.Endpoint+strconv.FormatUint(config.Port, 10)),
	)

	// create naming client
	client, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)
	if err != nil {
		panic("NewNacosRegistryService create failed")
	}
	return &NacosRegistryService{client: client}
}
func (n *NacosRegistryService) RegisterServiceInstance(config Config) {
	param := vo.RegisterInstanceParam{
		Ip:          config.NacosConfig.ServerAddr,
		Port:        config.NacosConfig.Port,
		ServiceName: config.NacosConfig.ServiceName,
		Weight:      config.NacosConfig.Weight,
		Enable:      config.NacosConfig.Enable,
		Healthy:     config.NacosConfig.Healthy,
		Ephemeral:   config.NacosConfig.Ephemeral,
		Metadata:    config.NacosConfig.MetaData,
		//ClusterName: config.NacosConfig.Clusters[0], // default value is DEFAULT
		GroupName: config.NacosConfig.GroupName, // default value is DEFAULT_GROUP
	}
	success, err := n.client.RegisterInstance(param)
	if !success || err != nil {
		panic("RegisterServiceInstance failed!" + err.Error())
	}
	fmt.Printf("RegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)
}

func (n *NacosRegistryService) DeRegisterServiceInstance(config Config) {
	param := vo.DeregisterInstanceParam{
		Ip:          config.NacosConfig.ServerAddr,
		Port:        config.NacosConfig.Port,
		ServiceName: config.NacosConfig.ServiceName,
		GroupName:   config.NacosConfig.GroupName,
		Cluster:     config.NacosConfig.Clusters[0],
		Ephemeral:   config.NacosConfig.Ephemeral, //it must be true
	}

	success, err := n.client.DeregisterInstance(param)
	if !success || err != nil {
		panic("DeRegisterServiceInstance failed!" + err.Error())
	}
	fmt.Printf("DeRegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)

}

//func (n *NacosRegistryService) BatchRegisterServiceInstance( param vo.RegisterInstanceParam) {
//	success, err :=n. client.BatchRegisterInstance(param)
//	if !success || err != nil {
//		panic("BatchRegisterServiceInstance failed!" + err.Error())
//	}
//	fmt.Printf("BatchRegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)
//}

func (n *NacosRegistryService) GetService(config Config) {
	param := vo.GetServiceParam{
		ServiceName: config.NacosConfig.ServiceName,
		GroupName:   config.NacosConfig.GroupName,
		Clusters:    config.NacosConfig.Clusters,
	}
	service, err := n.client.GetService(param)
	if err != nil {
		panic("GetService failed!" + err.Error())
	}
	fmt.Printf("GetService,param:%+v, result:%+v \n\n", param, service)

}

func (n *NacosRegistryService) SelectAllInstances(config Config) {
	param := vo.SelectAllInstancesParam{
		ServiceName: config.NacosConfig.ServiceName,
		GroupName:   config.NacosConfig.GroupName,
		Clusters:    config.NacosConfig.Clusters,
	}
	instances, err := n.client.SelectAllInstances(param)
	if err != nil {
		panic("SelectAllInstances failed!" + err.Error())
	}
	fmt.Printf("SelectAllInstance,param:%+v, result:%+v \n\n", param, instances)

}

func (n *NacosRegistryService) SelectInstances(config Config) {
	param := vo.SelectInstancesParam{
		ServiceName: config.NacosConfig.ServiceName,
		GroupName:   config.NacosConfig.GroupName,
		Clusters:    config.NacosConfig.Clusters,
		HealthyOnly: config.NacosConfig.HealthyOnly,
	}
	instances, err := n.client.SelectInstances(param)
	if err != nil {
		panic("SelectInstances failed!" + err.Error())
	}
	fmt.Printf("SelectInstances,param:%+v, result:%+v \n\n", param, instances)

}

func (n *NacosRegistryService) SelectOneHealthyInstance(config Config) {
	param := vo.SelectOneHealthInstanceParam{
		ServiceName: config.NacosConfig.ServiceName,
		GroupName:   config.NacosConfig.GroupName,
		Clusters:    config.NacosConfig.Clusters,
	}
	instances, err := n.client.SelectOneHealthyInstance(param)
	if err != nil {
		panic("SelectOneHealthyInstance failed!")
	}
	fmt.Printf("SelectOneHealthyInstance,param:%+v, result:%+v \n\n", param, instances)

}

func (n *NacosRegistryService) Subscribe(config Config) {
	param := &vo.SubscribeParam{
		ServiceName: config.NacosConfig.ServiceName,
		GroupName:   config.NacosConfig.GroupName,
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Printf("callback return services:%s \n\n", util.ToJsonString(services))
		},
	}
	err := n.client.Subscribe(param)
	if err != nil {
		panic("Subscribe failed!")
	}

}

func (n *NacosRegistryService) UpdateServiceInstance(config Config) {
	param := vo.UpdateInstanceParam{
		Ip:          config.NacosConfig.ServerAddr, //update ip
		Port:        config.NacosConfig.Port,
		ServiceName: config.NacosConfig.ServiceName,
		GroupName:   config.NacosConfig.GroupName,
		ClusterName: config.NacosConfig.Clusters[0],
		Weight:      10,
		Enable:      true,
		Ephemeral:   true,
		Metadata:    map[string]string{"idc": "beijing1"}, //update metadata
	}
	success, err := n.client.UpdateInstance(param)
	if !success || err != nil {
		panic("UpdateInstance failed!" + err.Error())
	}
	fmt.Printf("UpdateServiceInstance,param:%+v,result:%+v \n\n", param, success)

}

func (n *NacosRegistryService) UnSubscribe(config Config) {
	param := &vo.SubscribeParam{
		ServiceName: config.NacosConfig.ServiceName,
		GroupName:   config.NacosConfig.GroupName,
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Printf("callback return services:%s \n\n", util.ToJsonString(services))
		},
	}
	err := n.client.Unsubscribe(param)
	if err != nil {
		panic("Unsubscribe failed!" + err.Error())
	}

}

func (n *NacosRegistryService) GetAllService(config Config, pageInfo PageInfo) {
	param := vo.GetAllServiceInfoParam{
		GroupName: config.NacosConfig.GroupName,
		PageNo:    pageInfo.pageNo,
		PageSize:  pageInfo.pageSize,
	}
	service, err := n.client.GetAllServicesInfo(param)
	if err != nil {
		panic("GetAllService failed!")
	}
	fmt.Printf("GetAllService,param:%+v, result:%+v \n\n", param, service)

}
