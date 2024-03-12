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
	 "strconv"
	 "strings"
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
 
 type RedisConfig struct {
	 Cluster    string `yaml:"cluster" json:"cluster" koanf:"cluster"`
	 ServerAddr string `yaml:"server-addr" json:"server-addr" koanf:"server-addr"`
	 Username   string `yaml:"username" json:"username" koanf:"username"`
	 Password   string `yaml:"password" json:"password" koanf:"password"`
	 DB         int    `yaml:"db" json:"db" koanf:"db"`
 }
 
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
 
	 go redisRegistryService.subscribe()
 
	 return redisRegistryService
 }
 
 func (s *RedisRegistryService) Lookup(key string) (r []*ServiceInstance, err error) {
	 r = make([]*ServiceInstance, 0)
	 ins, ok := s.serverMap.Load(key)
	 if !ok {
		 list, err := s.cli.HGetAll(s.ctx, key).Result()
		 if err != nil {
			 return nil, err
		 }
		 for _, v := range list {
			 addrList := strings.Split(v, ":")
			 if len(addrList) < 2 {
				 continue
			 }
			 addr := addrList[0]
			 port, _err := strconv.Atoi(addrList[1])
			 if _err != nil {
				 continue
			 }
			 r = append(r, &ServiceInstance{
				 Addr: addr,
				 Port: port,
			 })
		 }
		 return
	 }
 
	 r = append(r, ins.([]*ServiceInstance)...)
	 return
 }
 
 func (s *RedisRegistryService) subscribe() error {
	 // 定时更新Map
	 go func() {
		 for range time.Tick(KeyRefreshPeriod * time.Millisecond) {
			 allServiceList, err := s.cli.HGetAll(s.ctx, "*").Result()
			 if err != nil {
				 return
			 }
 
			 for k, v := range allServiceList {
				 addrList := strings.Split(v, ":")
				 if len(addrList) < 2 {
					 continue
				 }
				 addr := addrList[0]
				 port, _err := strconv.Atoi(addrList[1])
				 if _err != nil {
					 continue
				 }
				 r = append(r, &ServiceInstance{
					 Addr: addr,
					 Port: port,
				 })
			 }
		 }
	 }()
 
	 // // 定时订阅
	 // go func() {
	 // 	for range time.Tick(1 * time.Millisecond) {
	 // 		func() {
	 // 			defer s.cli.Close()
	 // 			s.cli.Subscribe(s.ctx, func() {
	 // 				notifySub := NotifySub{listeners: listenerServiceMap[cluster]}
	 // 				return notifySub
	 // 			}())
	 // 		}()
	 // 	}
	 // }()
 
	 return nil
 }
 
 func (s *RedisRegistryService) getRedisRegistryKey() string {
	 return fmt.Sprintf("%s%s", RedisFileKeyPrefix, s.config.Cluster)
 }
 
 func (s *RedisRegistryService) register(key, value string) (err error) {
	 _, err = s.cli.HSet(s.ctx, key, value).Result()
	 if err != nil {
		 return
	 }
 
	 go func() {
		 s.keepAlive(s.ctx, key)
	 }()
 
	 return
 }
 
 func (s *RedisRegistryService) Close() {
	 if s.cli != nil {
		 s.cli.Close()
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
 