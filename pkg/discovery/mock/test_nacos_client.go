package mock

import "github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

type NacosClient interface {
	naming_client.INamingClient
}
