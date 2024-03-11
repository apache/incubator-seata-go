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
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seata/seata-go/pkg/util/log"
)

const (
	RedisFileKeyPrefix = "registry.redis."

	// redis registry key live 5 seconds, auto refresh key every 2 seconds
	KeyTTL           = 5
	KeyRefreshPeriod = 2
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
		config: redisConfig,
		cli:    cli,
		ctx:    context.Background(),
	}

	go redisRegistryService.watch()

	return redisRegistryService
}

func (s *RedisRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	ins, ok := s.serverMap.Load(key)
	if !ok {
		insTmp, err := s.cli.Get(s.ctx, key).Result()
		if err != nil {
			return nil, err
		}

	}
	return nil, nil
}

func (s *RedisRegistryService) subscribe() error {
	s.cli.Subscribe()
	// 定时更新Map
	go func() {
		for range time.Tick(KeyRefreshPeriod * time.Millisecond) {
			func() {
				defer s.cli.Close()
				updateClusterAddressMap(redisRegistryKey, cluster)
			}()
		}
	}()

	// 定时订阅
	go func() {
		for range time.Tick(1 * time.Millisecond) {
			func() {
				defer s.cli.Close()
				s.cli.Subscribe(s.ctx, func() {
					notifySub := NotifySub{listeners: listenerServiceMap[cluster]}
					return notifySub
				}(), redisRegistryKey)
			}()
		}
	}()

	return nil
}

func (s *RedisRegistryService) getRedisRegistryKey() string {
	return fmt.Sprintf("%s%s", RedisFileKeyPrefix, s.config.Cluster)
}

func (s *RedisRegistryService) SetHeartBeat(key, addr string) error {
	akey := "alives." + addr

	aliveDuration, _ := time.ParseDuration("10s")
	s.cli.Set(s.ctx, akey, key, aliveDuration)

	select {
	case <-time.After(aliveDuration):
		go s.SetHeartBeat(key, addr)
	}

	return nil
}

func (s *RedisRegistryService) Close() {
	if s.cli != nil {
		s.cli.Close()
	}
}
