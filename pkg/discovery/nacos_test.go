package discovery

import (
	"log"
	"testing"
	"time"
)

func TestNacosRegistryService(t *testing.T) {

	// 配置Nacos配置
	nacosConfig := &NacosConfig{
		ServerAddr:  "172.168.1.71",
		Namespace:   "public",
		Username:    "nacos",
		Password:    "hssw@nacos2024",
		Application: "test-service",
		Group:       "DEFAULT_GROUP",
	}

	registryService, err := NewNacosRegistryService(nacosConfig)
	if err != nil {
		t.Fatalf("Failed to create NacosRegistryService: %v", err)
	}

	instances := []*ServiceInstance{
		{Addr: "127.0.0.1", Port: 8080},
	}

	err = registryService.RegisterService(nacosConfig.Application, instances)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	foundInstances, err := registryService.Lookup(nacosConfig.Application)
	if err != nil {
		t.Fatalf("Failed to lookup service: %v", err)
	}

	if len(foundInstances) == 0 {
		t.Fatalf("No instances found for service: %s", nacosConfig.Application)
	}

	log.Printf("Found instances: %+v", foundInstances)

	time.Sleep(5 * time.Second)

	err = registryService.DeregisterService(nacosConfig.Application, instances)
	if err != nil {
		t.Fatalf("Failed to deregister service: %v", err)
	}

	registryService.Close()
}
