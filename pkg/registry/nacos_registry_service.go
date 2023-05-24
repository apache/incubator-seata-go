/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package registry

import (
	"github.com/seata/seata-go/pkg/util/log"
	"net"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type NacosRegistryService struct {
	client naming_client.INamingClient
	config NacosConfig
}

func NewNacosRegistryService(config NacosConfig) *NacosRegistryService {
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(config.ServerAddr, config.Port, constant.WithContextPath("/nacos")),
	}

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
	param := n.createRegistryParam(address)
	success, err := n.client.RegisterInstance(param)
	if !success || err != nil {
		panic("RegisterServiceInstance failed!" + err.Error())
	}
	log.Infof("RegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)
}

func (n *NacosRegistryService) DeRegisterServiceInstance(address net.TCPAddr) {
	param := vo.DeregisterInstanceParam{
		Ip:          address.IP.String(),
		Port:        uint64(address.Port),
		ServiceName: n.config.GroupName + "@@" + n.config.ServiceName,
		GroupName:   n.config.GroupName,
		Cluster:     n.config.Cluster,
		Ephemeral:   true, //it must be true
	}

	success, err := n.client.DeregisterInstance(param)
	if !success || err != nil {
		panic("DeRegisterServiceInstance failed!" + err.Error())
	}
	log.Infof("DeRegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)

}

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
	log.Infof("GetService,param:%+v, result:%+v \n\n", param, service)

}

func (n *NacosRegistryService) Subscribe(cluster string, groupName string) {
	param := &vo.SubscribeParam{
		ServiceName: n.config.ServiceName,
		GroupName:   groupName,
		Clusters:    []string{cluster},
		SubscribeCallback: func(services []model.Instance, err error) {
			log.Infof("callback return services:%s \n\n", util.ToJsonString(services))
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
		SubscribeCallback: func(services []model.Instance, err error) {
			log.Infof("callback return services:%s \n\n", util.ToJsonString(services))
		},
	}
	err := n.client.Unsubscribe(param)
	if err != nil {
		panic("Unsubscribe failed!" + err.Error())
	}

}

func (n *NacosRegistryService) createRegistryParam(address net.TCPAddr) vo.RegisterInstanceParam {
	param := vo.RegisterInstanceParam{
		Ip:          address.IP.String(),
		Port:        uint64(address.Port),
		ServiceName: n.config.ServiceName,
		Weight:      n.config.Weight,
		Enable:      n.config.Enable,
		Healthy:     n.config.Healthy,
		Ephemeral:   n.config.Ephemeral,
		Metadata:    n.config.MetaData,
		ClusterName: n.config.Cluster,
		GroupName:   n.config.GroupName, // default value is DEFAULT_GROUP
	}
	return param
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
	log.Infof("SelectAllInstance,param:%+v, result:%+v \n\n", param, instances)

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
	log.Infof("SelectInstances,param:%+v, result:%+v \n\n", param, instances)

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
	log.Infof("SelectOneHealthyInstance,param:%+v, result:%+v \n\n", param, instances)

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
	log.Infof("UpdateServiceInstance,param:%+v,result:%+v \n\n", param, success)
}

type PageInfo struct {
	pageNo   uint32
	pageSize uint32
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
	log.Infof("GetAllService,param:%+v, result:%+v \n\n", param, service)
}
