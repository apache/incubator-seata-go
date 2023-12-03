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
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

func TestConsulRegistryService_Lookup(t *testing.T) {
	type fields struct {
		config    *ConsulConfig
		client    *api.Client
		serverMap *sync.Map
		stopCh    chan struct{}
		watchers  map[string]*watch.Plan
		RWMutex   *sync.RWMutex
		watchType string
	}
	type args struct {
		key string
	}
	config := &ConsulConfig{
		Cluster:    "default",
		ServerAddr: "localhost:8500",
	}
	cfg := api.DefaultConfig()
	cli, _ := api.NewClient(cfg)
	serviceWithSyncMap := new(sync.Map)
	serviceWithSyncMap.Store("mysql_service", []*ServiceInstance{
		{
			Addr: "localhost",
			Port: 3306,
		},
		{
			Addr: "localhost",
			Port: 3306,
		},
	})
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantServiceIns []*ServiceInstance
		wantErr        bool
	}{
		{
			name: "mysql_service_without_sync_map",
			fields: fields{
				config:    config,
				client:    cli,
				serverMap: new(sync.Map),
				RWMutex:   new(sync.RWMutex),
				watchType: "service",
			},
			args: args{key: "mysql_service"},
			wantServiceIns: []*ServiceInstance{
				{
					Addr: "localhost",
					Port: 3306,
				},
				{
					Addr: "localhost",
					Port: 3306,
				},
			},
			wantErr: false,
		},
		{
			name: "mysql_service_with_sync_map",
			fields: fields{
				config:    config,
				client:    cli,
				serverMap: serviceWithSyncMap,
				RWMutex:   new(sync.RWMutex),
				watchType: "service",
			},
			args: args{key: "mysql_service"},
			wantServiceIns: []*ServiceInstance{
				{
					Addr: "localhost",
					Port: 3306,
				},
				{
					Addr: "localhost",
					Port: 3306,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ConsulRegistryService{
				config:    tt.fields.config,
				client:    tt.fields.client,
				serverMap: tt.fields.serverMap,
				stopCh:    tt.fields.stopCh,
				watchers:  tt.fields.watchers,
				RWMutex:   tt.fields.RWMutex,
				watchType: tt.fields.watchType,
			}
			gotServiceIns, err := s.Lookup(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lookup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotServiceIns, tt.wantServiceIns) {
				t.Errorf("Lookup() gotServiceIns = %v, want %v", gotServiceIns, tt.wantServiceIns)
			}
		})
	}
}
