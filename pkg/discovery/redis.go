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
	RedisFilekeyPrefix = "registry.redis."
)

type RedisRegistryService struct {
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
		cli: cli,
		ctx: context.Background(),
	}

	go redisRegistryService.watch()

	return redisRegistryService
}

func (s *RedisRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	insList, ok := s.serverMap.Load(key)
	if !ok {
		s.cli.SRandMember(s.ctx, key).Result()
	}
	return nil, nil
}

func (s *RedisRegistryService) watch() error {
	return nil
}

func (s *RedisRegistryService) getRedisRegistryKey() string {
	return fmt.Sprintf("%s%s", RedisFilekeyPrefix)
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

func (RedisRegistryService) Close() {
	//TODO implement me
	panic("implement me")
}
