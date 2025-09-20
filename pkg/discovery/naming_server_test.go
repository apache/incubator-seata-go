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
	"flag"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// fake config for tests
var testConfig = &NamingServerConfig{
	ServerAddr:                  "127.0.0.1:8080,localhost:9090",
	HeartbeatPeriod:             1000,
	Namespace:                   "testns",
	Username:                    "user",
	Password:                    "pass",
	TokenValidityInMilliseconds: 100,
	MetadataMaxAgeMs:            1000,
}

// Test getNamingAddrs splits config.ServerAddr
func TestGetNamingAddrs(t *testing.T) {
	client := &NamingServerClient{
		config: testConfig,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		longPollClient: &http.Client{Timeout: 30 * time.Second},
	}
	addrs := client.getNamingAddrs()
	if len(addrs) != 2 {
		t.Errorf("expected 2 addresses, got %d", len(addrs))
	}
}

// Test isTokenExpired behavior
func TestIsTokenExpired_NoCredentials(t *testing.T) {
	cfg := *testConfig
	cfg.Username = ""
	cfg.Password = ""
	client := &NamingServerClient{
		config: &cfg,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		longPollClient: &http.Client{Timeout: 30 * time.Second},
	}
	// without credentials, token should never be considered expired
	atomic.StoreInt64(&client.tokenTimeStamp, 0)
	if client.isTokenExpired() {
		t.Error("expected token not expired when no credentials")
	}
}

func TestIsTokenExpired_WithCredentials(t *testing.T) {
	client := &NamingServerClient{
		config: testConfig,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		longPollClient: &http.Client{Timeout: 30 * time.Second},
	}
	// timestamp zero => expired
	atomic.StoreInt64(&client.tokenTimeStamp, 0)
	if !client.isTokenExpired() {
		t.Error("expected token expired when timestamp zero and credentials present")
	}
	// future timestamp => not expired
	atomic.StoreInt64(&client.tokenTimeStamp, time.Now().UnixMilli())
	if client.isTokenExpired() {
		t.Error("expected token not expired when timestamp fresh")
	}
}

// Test doHealthCheck with a local httptest server
func TestDoHealthCheck(t *testing.T) {
	// healthy server
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()
	addr := healthy.Listener.Addr().String()
	client := &NamingServerClient{
		logger: zap.NewNop(),
		httpClient: &http.Client{Timeout: 3 * time.Second},
		longPollClient: &http.Client{Timeout: 30 * time.Second},
	}
	if !client.doHealthCheck(addr) {
		t.Error("expected healthy server to return true")
	}

	// unhealthy server
	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer unhealthy.Close()
	addr2 := unhealthy.Listener.Addr().String()
	if client.doHealthCheck(addr2) {
		t.Error("expected unhealthy server to return false")
	}
}

func TestHealthCheckThreshold(t *testing.T) {
	resetInstance()
	client := &NamingServerClient{
		config: testConfig,
		logger: zap.NewNop(),
		httpClient: &http.Client{Timeout: 3 * time.Second},
		longPollClient: &http.Client{Timeout: 30 * time.Second},
	}
	
	addr := "test-addr"
	
	client.availableNamingMap.Store(addr, int32(0))
	val, _ := client.availableNamingMap.Load(addr)
	if val.(int32) != 0 {
		t.Error("initial fail count should be 0")
	}
	
	client.checkAvailableNamingAddr([]string{addr})
	val, _ = client.availableNamingMap.Load(addr)
	if val.(int32) != 1 {
		t.Error("fail count should be 1 after first failure")
	}
	
	client.checkAvailableNamingAddr([]string{addr})
	val, _ = client.availableNamingMap.Load(addr)
	if val.(int32) != 2 {
		t.Error("fail count should be 2 after second failure")
	}
	
	client.checkAvailableNamingAddr([]string{addr})
	val, _ = client.availableNamingMap.Load(addr)
	if val.(int32) != 3 {
		t.Error("fail count should be 3 after third failure")
	}
}

// Test handleMetadata filters nodes correctly
func TestHandleMetadata(t *testing.T) {
	client := &NamingServerClient{
		httpClient: &http.Client{Timeout: 3 * time.Second},
		longPollClient: &http.Client{Timeout: 30 * time.Second},
	}
	meta := &MetaResponse{
		Term: 2,
		ClusterList: []Cluster{
			{
				ClusterName: "c1",
				ClusterType: "t1",
				UnitData: []Unit{
					{
						UnitName: "u1",
						NamingInstanceList: []NamingServerNode{
							{Role: ClusterRoleLeader, Term: 2, Healthy: true},
							{Role: ClusterRoleLeader, Term: 1, Healthy: true},
							{Role: ClusterRoleMember, Term: 1, Healthy: false},
						},
					},
				},
			},
		},
	}
	err := client.handleMetadata(meta, "vg1")
	if err != nil {
		t.Fatalf("handleMetadata returned error: %v", err)
	}
	val, ok := client.vgroupAddressMap.Load("vg1")
	if !ok {
		t.Fatal("vgroupAddressMap missing key vg1")
	}
	nodes := val.([]NamingServerNode)
	// expect only leader with term>=2 and all members
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

// Additional tests for getNamingAddr when availableNamingMap populated
func TestGetNamingAddr_NoAvailable(t *testing.T) {
	client := &NamingServerClient{
		config: testConfig,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		longPollClient: &http.Client{Timeout: 30 * time.Second},
	}
	// empty map
	_, err := client.getNamingAddr()
	if err == nil {
		t.Error("expected error when no available naming server")
	}
}

func TestGetNamingAddr_OneAvailable(t *testing.T) {
	client := &NamingServerClient{
		config: testConfig,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		longPollClient: &http.Client{Timeout: 30 * time.Second},
	}
	client.availableNamingMap.Store("host1", int32(0))
	addr, err := client.getNamingAddr()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "host1" {
		t.Errorf("expected host1, got %s", addr)
	}
}

func TestGetNamingAddr_CacheExpiry(t *testing.T) {
	resetInstance()
	client := &NamingServerClient{
		config: testConfig,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		longPollClient: &http.Client{Timeout: 30 * time.Second},
	}
	client.availableNamingMap.Store("host1", int32(0))
	client.availableNamingMap.Store("host2", int32(0))
	
	_, err := client.getNamingAddr()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	client.namingAddrCacheTimestamp = time.Now().UnixMilli() - 31000
	
	_, err = client.getNamingAddr()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if client.namingAddrCacheTimestamp <= time.Now().UnixMilli()-30000 {
		t.Error("cache timestamp should be updated after expiry")
	}
}

func TestLookup_FirstCall(t *testing.T) {
	// Reset naming server singleton instance
	resetInstance()
	// Create mock naming server with necessary endpoints (login, discovery, health)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Received request: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/api/v1/auth/login":
			// Handle login request, return valid JWT token
			w.Header().Set("Content-Type", contentTypeJSON)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code": "200",
				"data": "mock-jwt-token",
			})

		case "/naming/v1/discovery":
			// Verify auth token
			if r.Header.Get(authorizationHeader) != "mock-jwt-token" {
				t.Errorf("Missing valid token: %s", r.Header.Get(authorizationHeader))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Construct mock response with 4 nodes (2 valid, 2 invalid)
			resp := MetaResponse{
				Term: 1,
				ClusterList: []Cluster{{
					ClusterName: "testCluster",
					ClusterType: "SEATA",
					UnitData: []Unit{{
						UnitName: "testUnit",
						NamingInstanceList: []NamingServerNode{
							{Role: ClusterRoleLeader, Term: 1, Healthy: true, Transaction: Endpoint{Host: "10.0.0.1", Port: 8091, Protocol: "tcp"}, Version: "1.0.0"},
							{Role: ClusterRoleMember, Term: 1, Healthy: true, Transaction: Endpoint{Host: "10.0.0.2", Port: 8092, Protocol: "tcp"}, Version: "1.0.0"},
							{Healthy: false, Transaction: Endpoint{Host: "10.0.0.3", Port: 8093}},
							{Healthy: true, Transaction: Endpoint{Host: "10.0.0.4", Port: 70000}},
						},
					}},
				}},
			}

			w.Header().Set("Content-Type", contentTypeJSON)
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}

		case "/naming/v1/health":
			w.WriteHeader(http.StatusOK)

		default:
			t.Errorf("Unknown path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Configure client to use mock server
	mockAddr := mockServer.Listener.Addr().String()
	testConfig.ServerAddr = mockAddr
	client := GetInstance(testConfig)
	client.logger = zap.NewNop()
	client.availableNamingMap.Store(mockAddr, int32(0))
	client.namingAddrCache = mockAddr

	// Call Lookup and check for errors
	instances, err := client.Lookup("testGroup")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	// Verify cached data exists
	val, ok := client.vgroupAddressMap.Load("testGroup")
	if !ok {
		t.Fatal("No testGroup data in cache")
	}
	nodes := val.([]NamingServerNode)
	t.Logf("Fetched %d nodes from server", len(nodes))
	for i, node := range nodes {
		t.Logf("Node %d: Healthy=%v, Host=%s, Port=%d", i, node.Healthy, node.Transaction.Host, node.Transaction.Port)
	}

	// Verify correct number of valid instances (2 expected)
	if len(instances) != 2 {
		t.Fatalf("Expected 2 instances, got %d", len(instances))
	}
	expected := map[string]int{"10.0.0.1": 8091, "10.0.0.2": 8092}
	for _, inst := range instances {
		if port, ok := expected[inst.Addr]; !ok || inst.Port != port {
			t.Errorf("Mismatched instance: %s:%d", inst.Addr, inst.Port)
		}
	}
}

func TestRefreshToken_Success(t *testing.T) {
	// Reset naming server singleton instance
	resetInstance()
	// Mock login endpoint returning valid token
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" || r.Method != http.MethodPost {
			t.Errorf("Invalid request: %s %s", r.Method, r.URL.Path)
		}
		// Verify credentials
		if r.URL.Query().Get("username") != testConfig.Username || r.URL.Query().Get("password") != testConfig.Password {
			t.Error("Invalid username/password in request")
		}
		// Return success response with token
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code": "200",
			"data": "mock-jwt-token",
		})
	}))
	defer mockServer.Close()

	client := GetInstance(testConfig)
	client.logger = zap.NewNop()
	err := client.RefreshToken(mockServer.Listener.Addr().String())
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if client.jwtToken != "mock-jwt-token" {
		t.Error("jwtToken not updated after login")
	}
}

func TestRefreshToken_RetryMechanism(t *testing.T) {
	resetInstance()
	attemptCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code": "200",
			"data": "retry-success-token",
		})
	}))
	defer mockServer.Close()

	client := GetInstance(testConfig)
	client.logger = zap.NewNop()
	err := client.RefreshToken(mockServer.Listener.Addr().String())
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
	if client.jwtToken != "retry-success-token" {
		t.Error("jwtToken not updated after retry success")
	}
}

func TestWatch(t *testing.T) {
	// Reset naming server singleton instance
	resetInstance()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			// Handle login request
			if r.Method != http.MethodPost {
				t.Errorf("login: Expected POST, got %s", r.Method)
			}
			if r.URL.Query().Get("username") != testConfig.Username || r.URL.Query().Get("password") != testConfig.Password {
				t.Error("login: Invalid username/password")
			}
			w.Header().Set("Content-Type", contentTypeJSON)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code": "200",
				"data": "mock-jwt-token",
			})

		case "/naming/v1/watch":
			// Verify auth token
			if r.Header.Get(authorizationHeader) != "mock-jwt-token" {
				t.Errorf("watch: Missing valid token, got %s", r.Header.Get(authorizationHeader))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// Verify request method and parameters
			if r.Method != http.MethodPost {
				t.Errorf("watch: Expected POST, got %s", r.Method)
			}
			params := r.URL.Query()
			if params.Get("vGroup") != "testGroup" {
				t.Errorf("watch: Invalid vGroup, got %s", params.Get("vGroup"))
			}
			if params.Get("clientTerm") != "0" {
				t.Errorf("watch: Invalid clientTerm, got %s", params.Get("clientTerm"))
			}
			w.WriteHeader(http.StatusOK)

		default:
			t.Errorf("Unknown path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Configure client
	mockAddr := mockServer.Listener.Addr().String()
	client := GetInstance(testConfig)
	client.logger = zap.NewNop()
	client.availableNamingMap.Store(mockAddr, int32(0))
	client.namingAddrCache = mockAddr

	// Call Watch and verify result
	changed, err := client.Watch("testGroup")
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	if !changed {
		t.Error("Expected change detected (changed=true), got false")
	}
}

// Test Watch handling non-200 responses
func TestWatch_ErrorResponse(t *testing.T) {
	// Reset naming server singleton instance
	resetInstance()
	// Mock server returning error status
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	client := GetInstance(testConfig)
	client.logger = zap.NewNop()
	mockAddr := mockServer.Listener.Addr().String()
	client.availableNamingMap.Store(mockAddr, int32(0))
	client.namingAddrCache = mockAddr

	// Verify Watch returns error for non-200 status
	_, err := client.Watch("testGroup")
	if err == nil {
		t.Error("Expected error from Watch, got nil")
	}
}

func TestNamingServerConfig_LoadFromYAML(t *testing.T) {
	// Prepare test YAML content with custom configurations
	yamlContent := `
naming-server:
  cluster: "test-cluster"
  server-addr: "192.168.1.100:8888"
  namespace: "test-ns"
  heartbeat-period: 3000
  metadata-max-age-ms: 60000
  username: "test-user"
  password: "test-pass"
  token-validity-in-milliseconds: 3600000
`

	// Create temporary YAML file
	tmpFile, err := os.CreateTemp("", "naming-server-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write YAML: %v", err)
	}
	tmpFile.Close()

	// Parse YAML into config structure
	var registryConfig struct {
		NamingServer NamingServerConfig `yaml:"naming-server"`
	}
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	if err := yaml.Unmarshal(data, &registryConfig); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Verify parsed values match YAML content
	nsConfig := registryConfig.NamingServer
	if nsConfig.Cluster != "test-cluster" {
		t.Errorf("Cluster: expected 'test-cluster', got %s", nsConfig.Cluster)
	}
	if nsConfig.ServerAddr != "192.168.1.100:8888" {
		t.Errorf("ServerAddr: expected '192.168.1.100:8888', got %s", nsConfig.ServerAddr)
	}
	if nsConfig.Namespace != "test-ns" {
		t.Errorf("Namespace: expected 'test-ns', got %s", nsConfig.Namespace)
	}
	if nsConfig.HeartbeatPeriod != 3000 {
		t.Errorf("HeartbeatPeriod: expected 3000, got %d", nsConfig.HeartbeatPeriod)
	}
	if nsConfig.Username != "test-user" {
		t.Errorf("Username: expected 'test-user', got %s", nsConfig.Username)
	}
	if nsConfig.Password != "test-pass" {
		t.Errorf("Password: expected 'test-pass', got %s", nsConfig.Password)
	}
	if nsConfig.TokenValidityInMilliseconds != 3600000 {
		t.Errorf("TokenValidity: expected 3600000, got %d", nsConfig.TokenValidityInMilliseconds)
	}
}

func TestNamingServerConfig_DefaultValues(t *testing.T) {
	// Initialize default values using flag registration
	var nsConfig NamingServerConfig
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	nsConfig.RegisterFlagsWithPrefix("naming-server", fs)
	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	// Parse empty YAML configuration
	yamlContent := `naming-server: {}`
	tmpFile, err := os.CreateTemp("", "naming-server-default-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write YAML: %v", err)
	}
	tmpFile.Close()

	// Load empty config from YAML
	var tmpConfig struct {
		NamingServer NamingServerConfig `yaml:"naming-server"`
	}
	data, _ := os.ReadFile(tmpFile.Name())
	if err := yaml.Unmarshal(data, &tmpConfig); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Merge YAML values with defaults (YAML takes precedence)
	if tmpConfig.NamingServer.ServerAddr != "" {
		nsConfig.ServerAddr = tmpConfig.NamingServer.ServerAddr
	}
	if tmpConfig.NamingServer.Namespace != "" {
		nsConfig.Namespace = tmpConfig.NamingServer.Namespace
	}
	if tmpConfig.NamingServer.HeartbeatPeriod != 0 {
		nsConfig.HeartbeatPeriod = tmpConfig.NamingServer.HeartbeatPeriod
	}
	if tmpConfig.NamingServer.MetadataMaxAgeMs != 0 {
		nsConfig.MetadataMaxAgeMs = tmpConfig.NamingServer.MetadataMaxAgeMs
	}
	if tmpConfig.NamingServer.Username != "" {
		nsConfig.Username = tmpConfig.NamingServer.Username
	}
	if tmpConfig.NamingServer.TokenValidityInMilliseconds != 0 {
		nsConfig.TokenValidityInMilliseconds = tmpConfig.NamingServer.TokenValidityInMilliseconds
	}

	// Verify default values match expected
	if nsConfig.ServerAddr != "127.0.0.1:8081" {
		t.Errorf("ServerAddr default: expected '127.0.0.1:8081', got %s", nsConfig.ServerAddr)
	}
	if nsConfig.Namespace != "public" {
		t.Errorf("Namespace default: expected 'public', got %s", nsConfig.Namespace)
	}
	if nsConfig.HeartbeatPeriod != 5000 {
		t.Errorf("HeartbeatPeriod default: expected 5000, got %d", nsConfig.HeartbeatPeriod)
	}
	if nsConfig.MetadataMaxAgeMs != 30000 {
		t.Errorf("MetadataMaxAgeMs default: expected 30000, got %d", nsConfig.MetadataMaxAgeMs)
	}
	if nsConfig.Username != "" {
		t.Errorf("Username default: expected empty, got %s", nsConfig.Username)
	}
	if nsConfig.TokenValidityInMilliseconds != 29*60*1000 {
		t.Errorf("TokenValidity default: expected 1740000, got %d", nsConfig.TokenValidityInMilliseconds)
	}
}

func TestInitRegistry_WithNamingServerConfig(t *testing.T) {
	// Reset global registry instance
	registryServiceInstance = nil
	// Reset naming server singleton instance
	resetInstance()

	// Create custom configuration
	customConfig := &RegistryConfig{
		Type: NAMINGSERVER,
		NamingServer: NamingServerConfig{
			ServerAddr:                  "custom-server:8080",
			Username:                    "init-test-user",
			Password:                    "init-test-pass",
			HeartbeatPeriod:             2000,
			TokenValidityInMilliseconds: 600000,
		},
	}

	// Initialize registry with custom config
	InitRegistry(nil, customConfig)

	// Verify registry initialization
	registry := GetRegistry()
	if registry == nil {
		t.Fatal("InitRegistry failed: registry is nil")
	}

	// Check registry type
	namingRegistry, ok := registry.(*NamingServerRegistryService)
	if !ok {
		t.Fatalf("Expected NamingServerRegistryService, got %T", registry)
	}

	// Verify client uses custom config
	client := namingRegistry.client
	if client.config.ServerAddr != "custom-server:8080" {
		t.Errorf("Client ServerAddr: expected 'custom-server:8080', got %s", client.config.ServerAddr)
	}
	if client.config.Username != "init-test-user" {
		t.Errorf("Client Username: expected 'init-test-user', got %s", client.config.Username)
	}
	if client.config.Password != "init-test-pass" {
		t.Errorf("Client Password: expected 'init-test-pass', got %s", client.config.Password)
	}
	if client.config.HeartbeatPeriod != 2000 {
		t.Errorf("Client HeartbeatPeriod: expected 2000, got %d", client.config.HeartbeatPeriod)
	}
	if client.config.TokenValidityInMilliseconds != 600000 {
		t.Errorf("Client TokenValidity: expected 600000, got %d", client.config.TokenValidityInMilliseconds)
	}
}
