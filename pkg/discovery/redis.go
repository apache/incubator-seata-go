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
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/seata/seata-go/pkg/util/log"
)

const (
	// key = registry.redis.${cluster}_ip:port

	// redisFileKeyPrefix the redis register key prefix
	redisFileKeyPrefix = "registry.redis."

	// redisAddressSplitChar A notation for split ip addresses and ports
	redisAddressSplitChar = ":"

	// redisRegisterChannel the channel for redis to publish/subscript key&value
	redisRegisterChannel = "redis_registry_channel"

	// defaultCluster default cluster name
	defaultCluster = "default"

	// keyRefreshPeriod frequency of refreshing each key
	keyRefreshPeriod = 2

	// regexClusterName the regular expression for find the cluster name
	regexClusterName = `registry\.redis\.(\w+)_`
)

type RedisRegistryService struct {
	// the config about redis
	config *RedisConfig

	// client for redis
	cli *redis.Client

	// rwLock lock groupList when update service instance
	rwLock *sync.RWMutex

	// vgroupMapping to store the cluster group
	// eg: map[cluster_name_key]cluster_name
	vgroupMapping map[string]string

	// groupList store all addresses under this cluster
	// eg: map[cluster_name][]{service_instance1,service_instance2...}
	groupList map[string][]*ServiceInstance

	ctx context.Context
}

// NotifyMessage redis subscript structure
type NotifyMessage struct {
	// key = registry.redis.${cluster}_ip:port
	Key   string `json:"key"`
	Value string `json:"value"`
}

// newRedisRegisterService init the redis register service
func newRedisRegisterService(config *ServiceConfig, redisConfig *RedisConfig) RegistryService {
	if redisConfig == nil {
		log.Fatalf("redis config is nil")
		panic("redis config is nil")
	}

	cfg := &redis.Options{
		Addr:     redisConfig.ServerAddr,
		Username: redisConfig.Username,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	}
	cli := redis.NewClient(cfg)

	vgroupMapping := config.VgroupMapping
	groupList := make(map[string][]*ServiceInstance)

	redisRegistryService := &RedisRegistryService{
		config:        redisConfig,
		cli:           cli,
		ctx:           context.Background(),
		rwLock:        &sync.RWMutex{},
		vgroupMapping: vgroupMapping,
		groupList:     groupList,
	}

	// loading all server at init time
	redisRegistryService.load()
	// subscribe at real time
	go redisRegistryService.subscribe()
	// flushing all server at regular time
	// go redisRegistryService.flush()

	return redisRegistryService
}

func (s *RedisRegistryService) Lookup(key string) (r []*ServiceInstance, err error) {
	s.rwLock.RLock()
	defer s.rwLock.RUnlock()
	cluster := s.vgroupMapping[key]
	if cluster == "" {
		err = fmt.Errorf("cluster doesnt exit")
		return
	}

	r = s.groupList[cluster]

	return
}

// flush regular update all keys each 2 seconds
func (s *RedisRegistryService) flush() {
	ticker := time.NewTicker(keyRefreshPeriod * time.Second)
	defer ticker.Stop()
	func(t *time.Ticker) {
		// find all key and update value
		for {
			s.load()

			<-t.C
		}
	}(ticker)
}

// load loading all key & value into map
func (s *RedisRegistryService) load() {
	// find all the server list redis register by redisFileKeyPrefix
	keys, _, err := s.cli.Scan(s.ctx, 0, fmt.Sprintf("%s*", redisFileKeyPrefix), 0).Result()
	if err != nil {
		log.Errorf("RedisRegistryService-Scan-Key-Error:%s", err)
		return
	}
	for _, key := range keys {
		clusterName := s.getClusterNameByKey(key)
		val, err := s.cli.Get(s.ctx, key).Result()
		if err != nil {
			log.Errorf("RedisRegistryService-Get-Key:%s, Err:%s", key, err)
			continue
		}
		ins, err := s.getServerInstance(val)
		if err != nil {
			log.Errorf("RedisRegistryService-getServerInstance-val:%s, Err:%s", val, err)
			continue
		}
		// put server instance list in group list
		s.rwLock.Lock()
		if s.groupList[clusterName] == nil {
			s.groupList[clusterName] = make([]*ServiceInstance, 0)
		}
		s.groupList[clusterName] = append(s.groupList[clusterName], ins)
		s.rwLock.Unlock()
	}
}

// subscribe real time subscripting the change of data
func (s *RedisRegistryService) subscribe() {
	go func() {
		msgs := s.cli.Subscribe(s.ctx, redisRegisterChannel).Channel()
		for msg := range msgs {
			var data *NotifyMessage
			err := json.Unmarshal([]byte(msg.Payload), &data)
			if err != nil {
				log.Errorf("RedisRegistryService-subscribe-Subscribe-Err:%+v", err)
				continue
			}
			// get cluster name by key
			clusterName := s.getClusterNameByKey(data.Key)
			ins, err := s.getServerInstance(data.Value)
			if err != nil {
				log.Errorf("RedisRegistryService-subscribe-getServerInstance-value:%s, Err:%s", data.Value, err)
				continue
			}
			s.rwLock.Lock()
			if s.groupList[clusterName] == nil {
				s.groupList[clusterName] = make([]*ServiceInstance, 0)
			}
			s.groupList[clusterName] = append(s.groupList[clusterName], ins)
			s.rwLock.Unlock()
		}
	}()

	return
}

// getClusterNameByKey get the cluster name by key
func (s *RedisRegistryService) getClusterNameByKey(key string) string {
	// key = registry.redis.${cluster}_ip:port
	re := regexp.MustCompile(regexClusterName)
	match := re.FindStringSubmatch(key)
	// if we find the match , use the match result
	if len(match) > 1 {
		return match[1]
	}
	// if not find , return default cluster name for underwriting
	return s.getDefaultClusterName()
}

// getDefaultClusterName get default cluster name
func (s *RedisRegistryService) getDefaultClusterName() string {
	if s.config != nil && s.config.Cluster != "" {
		return s.config.Cluster
	}
	return defaultCluster
}

// getServerInstance parse ip address and port from value
func (s *RedisRegistryService) getServerInstance(value string) (ins *ServiceInstance, err error) {
	valueSplit := strings.Split(value, redisAddressSplitChar)
	if len(valueSplit) != 2 {
		err = fmt.Errorf("redis value has an incorrect format. value: %s", value)
		return
	}
	ip := valueSplit[0]
	port, err := strconv.Atoi(valueSplit[1])
	if err != nil {
		err = fmt.Errorf("redis port has an incorrect format. err: %w", err)
		return
	}
	ins = &ServiceInstance{
		Addr: ip,
		Port: port,
	}

	return
}

// getKey
// @param: addr the register service ip & port
// eg: localhost:7455
func (s *RedisRegistryService) getKey(addr string) string {
	return fmt.Sprintf("%s_%s", s.getRedisRegistryKey(), addr)
}

func (s *RedisRegistryService) getRedisRegistryKey() string {
	if s.config != nil && s.config.Cluster != "" {
		return fmt.Sprintf("%s%s", redisFileKeyPrefix, s.config.Cluster)
	}

	return fmt.Sprintf("%s%s", redisFileKeyPrefix, defaultCluster)
}

// register register to redis
func (s *RedisRegistryService) register(key, value string) (err error) {
	_, err = s.cli.HSet(s.ctx, key, value).Result()
	if err != nil {
		return
	}

	msg := &NotifyMessage{
		Key:   key,
		Value: value,
	}

	s.cli.Publish(s.ctx, redisRegisterChannel, msg)

	go func() {
		s.keepAlive(s.ctx, key)
	}()

	return
}

func (s *RedisRegistryService) Close() {
	if s.cli != nil {
		_ = s.cli.Close()
	}
}

// keepAlive keep every key alive
func (s *RedisRegistryService) keepAlive(ctx context.Context, key string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.cli.Expire(ctx, key, 2*time.Second)
		case <-ctx.Done():
			break
		}
	}
}
