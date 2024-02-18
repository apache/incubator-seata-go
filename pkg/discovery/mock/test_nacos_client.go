package mock

import "github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

//go:generate mockgen -destination=mock_nacos_client.go -package=mock . NacosClient
type NacosClient interface {
	naming_client.INamingClient
}
