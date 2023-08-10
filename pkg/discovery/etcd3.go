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
	"github.com/seata/seata-go/pkg/util/log"
	etcd3 "go.etcd.io/etcd/client/v3"
)

type EtcdRegistryService struct {
	client        *etcd3.Client
	cfg           etcd3.Config
	vgroupMapping map[string]string
	grouplist     map[string]*ServiceConfig

	stopCh chan struct{}
}

func newEtcdRegistryService(etcd3Config *Etcd3Config) RegistryService {

	if etcd3Config == nil {
		log.Fatalf("etcd config is nil")
		panic("etcd config is nil")
	}

	cfg := etcd3.Config{
		Endpoints: []string{etcd3Config.ServerAddr},
	}
	cli, err := etcd3.New(cfg)
	if err != nil {
		log.Fatalf("failed to create etcd3 client")
		panic("failed to create etcd3 client")
	}

	vgroupMapping := make(map[string]string, 0)
	grouplist := make(map[string]*ServiceConfig, 0)

	etcdRegistryService := &EtcdRegistryService{
		client:        cli,
		cfg:           cfg,
		vgroupMapping: vgroupMapping,
		grouplist:     grouplist,
		stopCh:        make(chan struct{}),
	}
	go etcdRegistryService.watch("seata-server")

	return etcdRegistryService
}

func (s *EtcdRegistryService) watch(key string) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监视指定 key 的变化
	watchCh := s.client.Watch(ctx, key, etcd3.WithPrefix())

	fmt.Println("yep I begin to watch")
	// 处理监视事件
	for {
		select {
		case watchResp, ok := <-watchCh:
			if !ok {
				fmt.Println("Watch channel closed")
				return
			}
			for _, event := range watchResp.Events {
				// 处理事件
				switch event.Type {
				case etcd3.EventTypePut:
					fmt.Printf("Key %s updated. New value: %s\n", event.Kv.Key, event.Kv.Value)
					// 进行你想要的操作
				case etcd3.EventTypeDelete:
					fmt.Printf("Key %s deleted.\n", event.Kv.Key)
					// 进行你想要的操作
				}
			}
		case <-s.stopCh: // stop信号
			fmt.Println("stop the watch")
			return
		}
	}
}

func (s *EtcdRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	//TODO implement me
	get, err := s.client.Get(context.Background(), key)
	if err != nil {
		return nil, err
	}
	kvs := get.Kvs
	fmt.Println(kvs)
	return nil, err
}

func (s *EtcdRegistryService) Close() {
	s.stopCh <- struct{}{}

}
