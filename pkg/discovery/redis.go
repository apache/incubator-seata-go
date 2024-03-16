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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/seata/seata-go/pkg/util/log"
)

const (
	redisFileKeyPrefix   = "registry.redis."
	redisRegisterChannel = "redis_registry_channel"

	keyRefreshPeriod = 2
)

type RedisRegistryService struct {
	// the config about redis
	config *RedisConfig

	// client for redis
	cli *redis.Client

	// serverMap the map of discovery server
	// key: server name value: server address
	serverMap *sync.Map

	ctx context.Context
}

type NotifyMessage struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

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

	redisRegistryService := &RedisRegistryService{
		config:    redisConfig,
		cli:       cli,
		ctx:       context.Background(),
		serverMap: new(sync.Map),
	}

	go redisRegistryService.load()
	go redisRegistryService.subscribe()
	go redisRegistryService.flush()

	return redisRegistryService
}

func (s *RedisRegistryService) Lookup(key string) (r []*ServiceInstance, err error) {
	r = make([]*ServiceInstance, 0)
	value, ok := s.serverMap.Load(key)
	if !ok {
		return
	}
	val, ok := value.(string)
	if !ok || val == "" {
		return
	}
	addrList := strings.Split(val, ":")
	if len(addrList) < 2 {
		return
	}
	addr := addrList[0]
	port, _err := strconv.Atoi(addrList[1])
	if _err != nil {
		return
	}
	r = append(r, &ServiceInstance{
		Addr: addr,
		Port: port,
	})

	return
}

func (s *RedisRegistryService) flush() {
	// regular update all keys each 2 second
	ticker := time.NewTicker(keyRefreshPeriod * time.Second)
	defer ticker.Stop()
	func(t *time.Ticker) {
		// find all key and update value
		for {
			<-t.C
			s.load()
		}
	}(ticker)
}

func (s *RedisRegistryService) load() {
	keys, _, err := s.cli.Scan(s.ctx, 0, fmt.Sprintf("%s*", redisFileKeyPrefix), 0).Result()
	if err != nil {
		log.Errorf("RedisRegistryService-Scan-Key-Error:%s", err)
		return
	}
	for _, key := range keys {
		val, err := s.cli.Get(s.ctx, key).Result()
		if err != nil {
			log.Errorf("RedisRegistryService-Get-Key:%s, Err:%s", key, err)
			continue
		}
		s.serverMap.Store(key, val)
	}
}

func (s *RedisRegistryService) subscribe() {
	// real time subscripting
	go func() {
		msgs := s.cli.Subscribe(s.ctx, redisRegisterChannel).Channel()
		for msg := range msgs {
			var data *NotifyMessage
			err := json.Unmarshal([]byte(msg.Payload), &data)
			if err != nil {
				log.Errorf("RedisRegistryService-subscribe-Subscribe:%+v", err)
				continue
			}
			s.serverMap.Store(data.Key, data.Value)
		}
	}()

	return
}

func (s *RedisRegistryService) getRedisRegistryKey() string {
	return fmt.Sprintf("%s%s", redisFileKeyPrefix, s.config.Cluster)
}

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
