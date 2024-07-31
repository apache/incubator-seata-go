package discovery

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"log"
	"sync"
	"time"
)

type NacosRegistryService struct {
	client       naming_client.INamingClient
	serviceCache map[string][]*ServiceInstance
	rwLock       sync.RWMutex
	stopCh       chan struct{}
}

func NewNacosRegistryService(nacosConfig *NacosConfig) (*NacosRegistryService, error) {
	serverConfig := []constant.ServerConfig{
		*constant.NewServerConfig(nacosConfig.ServerAddr, 8848),
	}

	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(nacosConfig.Namespace),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
		constant.WithUsername(nacosConfig.Username),
		constant.WithPassword(nacosConfig.Password),
	)

	client, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfig,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Nacos client: %v", err)
	}

	nacosRegistryService := &NacosRegistryService{
		client:       client,
		serviceCache: make(map[string][]*ServiceInstance),
		stopCh:       make(chan struct{}),
	}

	// Register the service instances
	instances := []*ServiceInstance{
		{Addr: "127.0.0.1", Port: 8080},
	}
	err = nacosRegistryService.RegisterService(nacosConfig.Application, instances)
	if err != nil {
		return nil, fmt.Errorf("failed to register service instances: %v", err)
	}

	// Start watching for service changes
	go nacosRegistryService.watch(nacosConfig)

	return nacosRegistryService, nil
}

func (s *NacosRegistryService) watch(nacosConfig *NacosConfig) {
	for {
		select {
		case <-s.stopCh:
			log.Println("Stopping Nacos watcher")
			return
		default:
			err := s.client.Subscribe(&vo.SubscribeParam{
				ServiceName: nacosConfig.Application,
				GroupName:   nacosConfig.Group,
				SubscribeCallback: func(services []model.SubscribeService, err error) {
					if err != nil {
						log.Printf("Failed to subscribe: %v", err)
						return
					}

					log.Printf("Received services: %+v", services)
					s.rwLock.Lock()
					defer s.rwLock.Unlock()

					serviceInstances := make([]*ServiceInstance, len(services))
					for i, service := range services {
						serviceInstances[i] = &ServiceInstance{
							Addr: service.Ip,
							Port: int(service.Port),
						}
					}
					s.serviceCache[nacosConfig.Application] = serviceInstances
					log.Printf("Service cache updated: %+v", s.serviceCache)
				},
			})
			if err != nil {
				log.Printf("Failed to subscribe to service: %v", err)
			}
			time.Sleep(10 * time.Second) // Wait before retrying
		}
	}
}

func (s *NacosRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	s.rwLock.RLock()
	defer s.rwLock.RUnlock()
	log.Printf("Service cache: %+v", s.serviceCache) // 增加日志输出
	if instances, ok := s.serviceCache[key]; ok {
		log.Printf("Found instances for key %s: %+v", key, instances)
		return instances, nil
	}
	log.Printf("No instances found for key %s", key)
	return nil, fmt.Errorf("no instances found for service: %s", key)
}

func (s *NacosRegistryService) RegisterService(serviceName string, instances []*ServiceInstance) error {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()

	// Check if instance is already registered
	existingInstances, exists := s.serviceCache[serviceName]
	if exists {
		for _, newInstance := range instances {
			for _, existingInstance := range existingInstances {
				if newInstance.Addr == existingInstance.Addr && newInstance.Port == existingInstance.Port {
					log.Printf("Instance %v is already registered", newInstance)
					return nil
				}
			}
		}
	}

	for _, instance := range instances {
		param := vo.RegisterInstanceParam{
			Ip:          instance.Addr,
			Port:        uint64(instance.Port),
			ServiceName: serviceName,
			GroupName:   "DEFAULT_GROUP",
			Weight:      1.0,
			Enable:      true,
			Healthy:     true,
			Ephemeral:   true,
		}
		_, err := s.client.RegisterInstance(param)
		if err != nil {
			log.Printf("Failed to register instance %v: %v", instance, err)
			return err
		}
		log.Printf("Successfully registered instance %v", instance)
	}

	// Update the service cache
	s.serviceCache[serviceName] = instances
	log.Printf("Service cache updated: %+v", s.serviceCache)
	return nil
}

func (s *NacosRegistryService) DeregisterService(serviceName string, instances []*ServiceInstance) error {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()

	// Check if instance is already deregistered
	existingInstances, exists := s.serviceCache[serviceName]
	if !exists {
		log.Printf("No instances registered for service %s", serviceName)
		return nil
	}

	for _, instance := range instances {
		param := vo.DeregisterInstanceParam{
			Ip:          instance.Addr,
			Port:        uint64(instance.Port),
			ServiceName: serviceName,
			GroupName:   "DEFAULT_GROUP",
		}
		_, err := s.client.DeregisterInstance(param)
		if err != nil {
			log.Printf("Failed to deregister instance %v: %v", instance, err)
			return err
		}
		log.Printf("Successfully deregistered instance %v", instance)
	}

	// Remove instances from cache
	newInstances := []*ServiceInstance{}
	for _, existingInstance := range existingInstances {
		shouldDeregister := false
		for _, instance := range instances {
			if existingInstance.Addr == instance.Addr && existingInstance.Port == instance.Port {
				shouldDeregister = true
				break
			}
		}
		if !shouldDeregister {
			newInstances = append(newInstances, existingInstance)
		}
	}

	s.serviceCache[serviceName] = newInstances
	if len(newInstances) == 0 {
		delete(s.serviceCache, serviceName)
	}

	return nil
}

func (s *NacosRegistryService) Close() {
	if s.stopCh != nil {
		close(s.stopCh)
	}
	if s.client != nil {
		for serviceName, instances := range s.serviceCache {
			for _, instance := range instances {
				param := vo.DeregisterInstanceParam{
					Ip:          instance.Addr,
					Port:        uint64(instance.Port),
					ServiceName: serviceName,
					GroupName:   "DEFAULT_GROUP",
				}
				_, err := s.client.DeregisterInstance(param)
				if err != nil {
					log.Printf("Failed to deregister instance %v: %v", instance, err)
				} else {
					log.Printf("Successfully deregistered instance %v", instance)
				}
			}
		}
	}
}
