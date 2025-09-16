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
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"seata.apache.org/seata-go/pkg/discovery/metadata"
	"testing"
)

var (
	serviceConfig = &ServiceConfig{VgroupMapping: map[string]string{"default_tx_group": "default"}}
	raftConfig    = RaftConfig{
		MetadataMaxAgeMs:            int64(30000),
		ServerAddr:                  "",
		TokenValidityInMilliseconds: int64(1740000),
	}
	registryConfig = &RegistryConfig{
		Type:             "raft",
		NamingserverAddr: "",
		Username:         "seata",
		Password:         "seata",
		Raft:             raftConfig,
	}
)

func TestMetadataHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Received request: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/api/v1/auth/login":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code": "200",
				"data": "mock-jwt-token",
			})

		case "/metadata/v1/cluster":
			if r.Header.Get("Authorization") != "mock-jwt-token" {
				t.Errorf("Missing valid token: %s", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			resp := metadata.MetadataResponse{
				Term: 1,
				Nodes: []*metadata.Node{
					{
						Control:     &metadata.Endpoint{Host: "127.0.0.1", Port: 7001},
						Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 7001},
						Internal:    &metadata.Endpoint{Host: "127.0.0.1", Port: 9001},
						Group:       "default",
						Role:        metadata.LEADER,
						Version:     "2.5.0",
						Metadata:    map[string]interface{}{},
					},
					{
						Control:     &metadata.Endpoint{Host: "127.0.0.1", Port: 7002},
						Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 7002},
						Internal:    &metadata.Endpoint{Host: "127.0.0.1", Port: 9002},
						Group:       "default",
						Role:        metadata.FOLLOWER,
						Version:     "2.5.0",
						Metadata:    map[string]interface{}{},
					},
					{
						Control:     &metadata.Endpoint{Host: "127.0.0.1", Port: 7003},
						Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 7003},
						Internal:    &metadata.Endpoint{Host: "127.0.0.1", Port: 9003},
						Group:       "default",
						Role:        metadata.FOLLOWER,
						Version:     "2.5.0",
						Metadata:    map[string]interface{}{},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}

		default:
			t.Errorf("Unknown path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()
	mockAddr := mockServer.Listener.Addr().String()
	registryConfig.Raft.ServerAddr = mockAddr
	registryConfig.NamingserverAddr = mockAddr
	service := NewRaftRegistryService(serviceConfig, registryConfig)
	instances, err := service.Lookup("default_tx_group")
	if err != nil {
		t.Errorf("Failed to lookup service: %v", err)
	}
	serviceInstances := []*ServiceInstance{
		{
			Addr: "127.0.0.1",
			Port: 7001,
		}, {
			Addr: "127.0.0.1",
			Port: 7002,
		}, {
			Addr: "127.0.0.1",
			Port: 7003,
		},
	}
	assert.Equal(t, serviceInstances, instances)
}
