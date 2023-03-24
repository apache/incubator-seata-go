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
	"fmt"
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
	param := n.createRegistryParam(address)
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
		ServiceName: n.config.GroupName + "@@" + n.config.ServiceName,
		GroupName:   n.config.GroupName,
		Cluster:     n.config.Cluster,
		Ephemeral:   true, //it must be true
	}

	success, err := n.client.DeregisterInstance(param)
	if !success || err != nil {
		panic("DeRegisterServiceInstance failed!" + err.Error())
	}
	fmt.Printf("DeRegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)

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
	fmt.Printf("GetService,param:%+v, result:%+v \n\n", param, service)

}

func (n *NacosRegistryService) Subscribe(cluster string, groupName string) {
	param := &vo.SubscribeParam{
		ServiceName: n.config.ServiceName,
		GroupName:   groupName,
		Clusters:    []string{cluster},
		SubscribeCallback: func(services []model.Instance, err error) {
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
		SubscribeCallback: func(services []model.Instance, err error) {
			fmt.Printf("callback return services:%s \n\n", util.ToJsonString(services))
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
