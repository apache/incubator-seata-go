package registry //nolint:typecheck

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"net"
	"strconv"
)

type NacosRegistryService struct {
	client naming_client.INamingClient
	config NacosConfig
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
	return &NacosRegistryService{client: client, config: config}
}
func (n *NacosRegistryService) RegisterServiceInstance(address net.TCPAddr) {
	param := vo.RegisterInstanceParam{
		Ip:          address.IP.String(),
		Port:        uint64(address.Port),
		ServiceName: n.config.ServiceName,
		Weight:      n.config.Weight,
		Enable:      n.config.Enable,
		Healthy:     n.config.Healthy,
		Ephemeral:   n.config.Ephemeral,
		Metadata:    n.config.MetaData,
		//ClusterName: n.config.Clusters[0], // default value is DEFAULT
		GroupName: n.config.GroupName, // default value is DEFAULT_GROUP
	}
	success, err := n.client.RegisterInstance(param)
	if !success || err != nil {
		panic("RegisterServiceInstance failed!" + err.Error())
	}
	fmt.Printf("RegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)
}

func (n *NacosRegistryService) DeRegisterServiceInstance(address net.TCPAddr) {
	param := vo.DeregisterInstanceParam{
		Ip:          address.IP.String(),
		Port:        uint64(address.Port),
		ServiceName: n.config.ServiceName,
		GroupName:   n.config.GroupName,
		Cluster:     n.config.Clusters[0],
		Ephemeral:   n.config.Ephemeral, //it must be true
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

func (n *NacosRegistryService) GetService(cluster string, groupName string) {
	param := vo.GetServiceParam{
		ServiceName: n.config.ServiceName,
		GroupName:   groupName,
		Clusters:    []string{cluster},
	}
	service, err := n.client.GetService(param)
	if err != nil {
		panic("GetService failed!" + err.Error())
	}
	fmt.Printf("GetService,param:%+v, result:%+v \n\n", param, service)

}

func (n *NacosRegistryService) SelectAllInstances(cluster string, groupName string) {
	param := vo.SelectAllInstancesParam{
		ServiceName: n.config.ServiceName,
		GroupName:   groupName,
		Clusters:    []string{cluster},
	}
	instances, err := n.client.SelectAllInstances(param)
	if err != nil {
		panic("SelectAllInstances failed!" + err.Error())
	}
	fmt.Printf("SelectAllInstance,param:%+v, result:%+v \n\n", param, instances)

}

func (n *NacosRegistryService) SelectInstances(cluster string, groupName string) {
	param := vo.SelectInstancesParam{
		ServiceName: n.config.ServiceName,
		GroupName:   groupName,
		Clusters:    []string{cluster},
		HealthyOnly: n.config.HealthyOnly,
	}
	instances, err := n.client.SelectInstances(param)
	if err != nil {
		panic("SelectInstances failed!" + err.Error())
	}
	fmt.Printf("SelectInstances,param:%+v, result:%+v \n\n", param, instances)

}

func (n *NacosRegistryService) SelectOneHealthyInstance(cluster string, groupName string) {
	param := vo.SelectOneHealthInstanceParam{
		ServiceName: n.config.ServiceName,
		GroupName:   groupName,
		Clusters:    []string{cluster},
	}
	instances, err := n.client.SelectOneHealthyInstance(param)
	if err != nil {
		panic("SelectOneHealthyInstance failed!")
	}
	fmt.Printf("SelectOneHealthyInstance,param:%+v, result:%+v \n\n", param, instances)

}

func (n *NacosRegistryService) Subscribe(cluster string, groupName string) {
	param := &vo.SubscribeParam{
		ServiceName: n.config.ServiceName,
		GroupName:   groupName,
		Clusters:    []string{cluster},
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Printf("callback return services:%s \n\n", util.ToJsonString(services))
		},
	}
	err := n.client.Subscribe(param)
	if err != nil {
		panic("Subscribe failed!")
	}

}

func (n *NacosRegistryService) UnSubscribe(cluster string, groupName string) {
	param := &vo.SubscribeParam{
		ServiceName: n.config.ServiceName,
		GroupName:   groupName,
		Clusters:    []string{cluster},
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Printf("callback return services:%s \n\n", util.ToJsonString(services))
		},
	}
	err := n.client.Unsubscribe(param)
	if err != nil {
		panic("Unsubscribe failed!" + err.Error())
	}

}

func (n *NacosRegistryService) UpdateServiceInstance(address net.TCPAddr, cluster string, groupName string) {
	param := vo.UpdateInstanceParam{
		Ip:          address.IP.String(), //update ip
		Port:        uint64(address.Port),
		ServiceName: n.config.ServiceName,
		GroupName:   groupName,
		ClusterName: cluster,
		Weight:      n.config.Weight,
		Enable:      n.config.Enable,
		Ephemeral:   n.config.Ephemeral,
		Metadata:    n.config.MetaData, //update metadata
	}
	success, err := n.client.UpdateInstance(param)
	if !success || err != nil {
		panic("UpdateInstance failed!" + err.Error())
	}
	fmt.Printf("UpdateServiceInstance,param:%+v,result:%+v \n\n", param, success)
}

func (n *NacosRegistryService) GetAllService(groupName string, pageInfo PageInfo) {
	param := vo.GetAllServiceInfoParam{
		GroupName: groupName,
		PageNo:    pageInfo.pageNo,
		PageSize:  pageInfo.pageSize,
	}
	service, err := n.client.GetAllServicesInfo(param)
	if err != nil {
		panic("GetAllService failed!")
	}
	fmt.Printf("GetAllService,param:%+v, result:%+v \n\n", param, service)
}
