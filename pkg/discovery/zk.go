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
package discovery

import (
	"fmt"
	"github.com/go-zookeeper/zk"
	"seata.apache.org/seata-go/pkg/discovery/mock"
	"seata.apache.org/seata-go/pkg/util/log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// zkConnAdapter wraps a real *zk.Conn to implement ZkConnInterface.
type zkConnAdapter struct {
	conn *zk.Conn
}

func (a *zkConnAdapter) GetW(path string) ([]byte, *zk.Stat, <-chan zk.Event, error) {
	return a.conn.GetW(path)
}

func (a *zkConnAdapter) Get(path string) ([]byte, *zk.Stat, error) {
	return a.conn.Get(path)
}

func (a *zkConnAdapter) Close() error {
	a.conn.Close()
	return nil
}

const (
	zookeeperClusterPrefix = "/registry-seata"
)

type ZookeeperRegistryService struct {
	conn          mock.ZkConnInterface
	vgroupMapping map[string]string
	grouplist     map[string][]*ServiceInstance
	rwLock        sync.RWMutex
	stopCh        chan struct{}
}

func newZookeeperRegistryService(config *ServiceConfig, zkConfig *ZookeeperConfig) RegistryService {
	if zkConfig == nil {
		log.Fatalf("zookeeper config is nil")
		panic("zookeeper config is nil")
	}

	// Connect to the actual *zk.Conn
	conn, _, err := zk.Connect([]string{zkConfig.ServerAddr}, zkConfig.SessionTimeout*time.Second)
	if err != nil {
		log.Fatalf("failed to create zookeeper client")
		panic("failed to create zookeeper client")
	}

	// Wrap it into an adapter
	adapter := &zkConnAdapter{conn: conn}

	vgroupMapping := config.VgroupMapping
	grouplist := make(map[string][]*ServiceInstance, 0)

	// init groplist
	grouplist, err = initFromServiceConfig(config)
	if err != nil {
		log.Errorf("Error initializing service config: %v", err)
		return nil
	}

	zkRegistryService := &ZookeeperRegistryService{
		conn:          adapter,
		vgroupMapping: vgroupMapping,
		grouplist:     grouplist,
		stopCh:        make(chan struct{}),
	}

	//go zkRegistryService.watch(zookeeperClusterPrefix)
	go zkRegistryService.watch(zkConfig.NodePath)

	return zkRegistryService
}

func (s *ZookeeperRegistryService) watch(path string) {
	for {
		// Get initial data and set the watch
		data, _, events, err := s.conn.GetW(path)
		if err != nil {
			log.Infof("Failed to get server instances from Zookeeper: %v", err)
			return
		}

		// Handle initial data
		s.handleZookeeperData(path, data)

		// Listen for changes to Zookeeper nodes
		for {
			select {
			case event := <-events:
				// Re-establish the watch to continue listening for changes
				data, _, events, err = s.conn.GetW(path)
				if err != nil {
					log.Errorf("Failed to set watch on Zookeeper node: %v", err)
					return
				}
				switch event.Type {
				case zk.EventNodeCreated, zk.EventNodeDataChanged:
					log.Infof("Node updated: %s", event.Path)
					s.handleZookeeperData(event.Path, data)

				case zk.EventNodeDeleted:
					log.Infof("Node deleted: %s", event.Path)
					s.removeServiceInstance(event.Path)
				}

			case <-s.stopCh:
				log.Warn("Received stop signal, stopping watch.")
				return
			}
		}
	}
}

func (s *ZookeeperRegistryService) handleZookeeperData(path string, data []byte) {
	clusterName, serverInstance, err := parseZookeeperData(path, data)
	if err != nil {
		log.Errorf("Zookeeper data error: %s", err)
		return
	}

	s.rwLock.Lock()
	if s.grouplist[clusterName] == nil {
		s.grouplist[clusterName] = []*ServiceInstance{serverInstance}
	} else {
		s.grouplist[clusterName] = append(s.grouplist[clusterName], serverInstance)
	}
	s.rwLock.Unlock()
}

func (s *ZookeeperRegistryService) removeServiceInstance(path string) {
	clusterName, ip, port, err := parseClusterAndAddress(path)
	if err != nil {
		log.Errorf("Zookeeper path error: %s", err)
		return
	}

	s.rwLock.Lock()
	serviceInstances := s.grouplist[clusterName]
	if serviceInstances == nil {
		log.Warnf("Zookeeper doesn't exist cluster: %s", clusterName)
		s.rwLock.Unlock()
		return
	}
	s.grouplist[clusterName] = removeValueFromList(serviceInstances, ip, port)
	s.rwLock.Unlock()
}

func parseZookeeperData(path string, data []byte) (string, *ServiceInstance, error) {
	parts := strings.Split(string(data), addressSplitChar)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("Zookeeper data has incorrect format: %s", string(data))
	}
	ip := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", nil, fmt.Errorf("Invalid port in Zookeeper data: %w", err)
	}
	clusterName := strings.TrimPrefix(path, zookeeperClusterPrefix+"/")
	serverInstance := &ServiceInstance{
		Addr: ip,
		Port: port,
	}
	return clusterName, serverInstance, nil
}

func parseClusterAndAddress(path string) (string, string, int, error) {
	// Extract cluster and address information from path
	// Example: /registry-seata/<cluster>/<ip>:<port>
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return "", "", 0, fmt.Errorf("Zookeeper path format is incorrect: %s", path)
	}
	cluster := parts[2]
	addressParts := strings.Split(parts[3], addressSplitChar)
	if len(addressParts) != 2 {
		return "", "", 0, fmt.Errorf("Zookeeper address format is incorrect: %s", parts[3])
	}
	ip := addressParts[0]
	port, err := strconv.Atoi(addressParts[1])
	if err != nil {
		return "", "", 0, fmt.Errorf("Invalid port in Zookeeper address: %w", err)
	}
	return cluster, ip, port, nil
}

func (s *ZookeeperRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	s.rwLock.RLock()
	defer s.rwLock.RUnlock()
	cluster := s.vgroupMapping[key]
	if cluster == "" {
		return nil, fmt.Errorf("cluster doesn't exist")
	}
	list := s.grouplist[cluster]
	return list, nil
}

func (s *ZookeeperRegistryService) Close() {
	close(s.stopCh)
	if err := s.conn.Close(); err != nil {
		log.Errorf("zk closes incorrectly %v", err)
	}
}

// initFromServiceConfig initializes the service instances from the ServiceConfig.
func initFromServiceConfig(serviceConfig *ServiceConfig) (map[string][]*ServiceInstance, error) {
	grouplist := make(map[string][]*ServiceInstance)

	for _, groupName := range serviceConfig.VgroupMapping {
		addrStr, ok := serviceConfig.Grouplist[groupName]
		if !ok || addrStr == "" {
			return nil, fmt.Errorf("endpoint is empty for group: %s", groupName)
		}

		addrs := strings.Split(addrStr, endPointSplitChar)
		instances := make([]*ServiceInstance, 0)

		for _, addr := range addrs {
			ipPort := strings.Split(addr, ipPortSplitChar)
			if len(ipPort) != 2 {
				return nil, fmt.Errorf("endpoint format should be like ip:port. endpoint: %s", addr)
			}
			ip := ipPort[0]
			port, err := strconv.Atoi(ipPort[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port format: %v", err)
			}
			instances = append(instances, &ServiceInstance{
				Addr: ip,
				Port: port,
			})
		}

		grouplist[groupName] = instances
	}

	return grouplist, nil
}
