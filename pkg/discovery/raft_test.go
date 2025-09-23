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
	"math/rand"
	"net/http"
	"net/http/httptest"
	"seata.apache.org/seata-go/pkg/discovery/metadata"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

func TestMultiClusterWatch(t *testing.T) {
	clusterRequests := make(map[string]int)
	var requestMu sync.Mutex
	
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMu.Lock()
		clusterRequests[r.URL.Path]++
		requestMu.Unlock()
		
		switch r.URL.Path {
		case "/api/v1/auth/login":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code": "200",
				"data": "mock-jwt-token",
			})
			
		case "/metadata/v1/cluster":
			resp := metadata.MetadataResponse{
				Term: 1,
				Nodes: []*metadata.Node{
					{
						Control:     &metadata.Endpoint{Host: "127.0.0.1", Port: 7001},
						Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 7001},
						Internal:    &metadata.Endpoint{Host: "127.0.0.1", Port: 9001},
						Group:       "",
						Role:        metadata.LEADER,
						Version:     "2.5.0",
						Metadata:    map[string]interface{}{},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()
	
	multiServiceConfig := &ServiceConfig{
		VgroupMapping: map[string]string{
			"tx_group_1": "cluster1", 
			"tx_group_2": "cluster2",
		},
	}
	
	mockAddr := mockServer.Listener.Addr().String()
	multiRegistryConfig := &RegistryConfig{
		Type:             "raft",
		NamingserverAddr: mockAddr,
		Username:         "seata",
		Password:         "seata",
		Raft: RaftConfig{
			MetadataMaxAgeMs:            int64(30000),
			ServerAddr:                  mockAddr + "," + mockAddr,
			TokenValidityInMilliseconds: int64(1740000),
		},
	}
	
	service := NewRaftRegistryService(multiServiceConfig, multiRegistryConfig)
	defer service.Close()
	
	_, err1 := service.Lookup("tx_group_1")
	assert.NoError(t, err1)
	
	_, err2 := service.Lookup("tx_group_2") 
	assert.NoError(t, err2)
	
	requestMu.Lock()
	clusterCount := clusterRequests["/metadata/v1/cluster"]
	requestMu.Unlock()
	
	assert.True(t, clusterCount >= 2, "Multiple clusters should trigger multiple cluster metadata requests")
}

func TestConcurrentAccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code": "200",
				"data": "mock-jwt-token",
			})
			
		case "/metadata/v1/cluster":
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
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			
		case "/metadata/v1/watch":
			w.WriteHeader(http.StatusOK)
			
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()
	
	mockAddr := mockServer.Listener.Addr().String()
	testRegistryConfig := &RegistryConfig{
		Type:             "raft",
		NamingserverAddr: mockAddr,
		Username:         "seata",
		Password:         "seata",
		Raft: RaftConfig{
			MetadataMaxAgeMs:            int64(30000),
			ServerAddr:                  mockAddr,
			TokenValidityInMilliseconds: int64(1740000),
		},
	}
	
	service := NewRaftRegistryService(serviceConfig, testRegistryConfig)
	defer service.Close()
	
	_, err := service.Lookup("default_tx_group")
	assert.NoError(t, err)
	
	var wg sync.WaitGroup
	successCount := int32(0)
	
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			_, err := service.Lookup("default_tx_group")
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}
	
	wg.Wait()
	
	assert.True(t, atomic.LoadInt32(&successCount) > 0, "Some concurrent lookups should succeed")
}

func TestWatchErrorRecovery(t *testing.T) {
	service := &RaftRegistryService{
		metadata: metadata.NewMetadata(),
		vgroupMapping: map[string]string{"test_group": "test_cluster"},
		currentTransactionClusterName: "test_cluster",
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
		initAddresses: sync.Map{},
		cfg: &RaftConfig{TokenValidityInMilliseconds: 1740000},
		tokenTimestamp: time.Now().UnixMilli(),
		httpClient: &http.Client{},
	}
	
	service.metadata.RefreshMetadata("test_cluster", metadata.MetadataResponse{
		Term: 1,
		Nodes: []*metadata.Node{
			{
				Control:     &metadata.Endpoint{Host: "127.0.0.1", Port: 7001},
				Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 8001},
				Internal:    &metadata.Endpoint{Host: "127.0.0.1", Port: 9001},
				Group:       "",
				Role:        metadata.LEADER,
				Version:     "2.5.0",
				Metadata:    map[string]interface{}{},
			},
		},
	})
	
	service.initAddresses.Store("test_cluster", []*ServiceInstance{
		{Addr: "127.0.0.1", Port: 7001},
	})
	
	result, err := service.watch()
	assert.Error(t, err)
	assert.False(t, result)
	
	address, err := service.queryHttpAddress("test_cluster", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, address)
}

func TestRefreshAliveLookup(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code": "200",
				"data": "mock-jwt-token",
			})
			
		case "/metadata/v1/cluster":
			resp := metadata.MetadataResponse{
				Term: 1,
				Nodes: []*metadata.Node{
					{
						Control:     &metadata.Endpoint{Host: "127.0.0.1", Port: 7001},
						Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 8001},
						Internal:    &metadata.Endpoint{Host: "127.0.0.1", Port: 9001},
						Group:       "default",
						Role:        metadata.LEADER,
						Version:     "2.5.0",
						Metadata:    map[string]interface{}{},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()
	
	mockAddr := mockServer.Listener.Addr().String()
	testRegistryConfig := &RegistryConfig{
		Type:             "raft",
		NamingserverAddr: mockAddr,
		Username:         "seata",
		Password:         "seata",
		Raft: RaftConfig{
			MetadataMaxAgeMs:            int64(30000),
			ServerAddr:                  mockAddr,
			TokenValidityInMilliseconds: int64(1740000),
		},
	}
	
	service := NewRaftRegistryService(serviceConfig, testRegistryConfig)
	defer service.Close()
	
	_, err := service.Lookup("default_tx_group")
	assert.NoError(t, err)
	
	aliveInstances := []*ServiceInstance{
		{Addr: "127.0.0.1", Port: 8001},
		{Addr: "127.0.0.1", Port: 8002},
		{Addr: "127.0.0.1", Port: 8003},
	}
	
	result, err := service.RefreshAliveLookup("default_tx_group", aliveInstances)
	assert.NoError(t, err)
	
	expectedResult := []*ServiceInstance{
		{Addr: "127.0.0.1", Port: 8002},
		{Addr: "127.0.0.1", Port: 8003},
	}
	assert.Equal(t, expectedResult, result)
	
	emptyResult, err := service.RefreshAliveLookup("default_tx_group", []*ServiceInstance{})
	assert.NoError(t, err)
	assert.Empty(t, emptyResult)
	
	_, err = service.RefreshAliveLookup("non_existent_group", aliveInstances)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster not found")
}

func TestTypeAssertionSafety(t *testing.T) {
	service := &RaftRegistryService{
		aliveNodes: sync.Map{},
		vgroupMapping: map[string]string{"test_group": "test_cluster"},
		metadata: metadata.NewMetadata(),
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	
	service.metadata.RefreshMetadata("test_cluster", metadata.MetadataResponse{
		Term: 1,
		Nodes: []*metadata.Node{
			{
				Control:     &metadata.Endpoint{Host: "127.0.0.1", Port: 7001},
				Transaction: &metadata.Endpoint{Host: "127.0.0.1", Port: 8001},
				Internal:    &metadata.Endpoint{Host: "127.0.0.1", Port: 9001},
				Group:       "",
				Role:        metadata.LEADER,
				Version:     "2.5.0",
				Metadata:    map[string]interface{}{},
			},
		},
	})
	
	service.aliveNodes.Store("test_group", "invalid_type_string")
	service.currentTransactionServiceGroup = "test_group"
	
	_, err := service.queryHttpAddress("test_cluster", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid alive nodes type")
	
	service.aliveNodes.Store("test_group", []*ServiceInstance{
		{Addr: "127.0.0.1", Port: 8001},
	})
	
	address, err := service.queryHttpAddress("test_cluster", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, address)
}
