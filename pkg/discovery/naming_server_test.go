package discovery

import (
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
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
	client := &NamingServerClient{config: testConfig}
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
	client := &NamingServerClient{config: &cfg}
	// without credentials, token should never be considered expired
	atomic.StoreInt64(&client.tokenTimeStamp, 0)
	if client.isTokenExpired() {
		t.Error("expected token not expired when no credentials")
	}
}

func TestIsTokenExpired_WithCredentials(t *testing.T) {
	client := &NamingServerClient{config: testConfig}
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
	client := &NamingServerClient{logger: zap.NewNop()}
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

// Test handleMetadata filters nodes correctly
func TestHandleMetadata(t *testing.T) {
	client := &NamingServerClient{}
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
	client := &NamingServerClient{config: testConfig}
	// empty map
	_, err := client.getNamingAddr()
	if err == nil {
		t.Error("expected error when no available naming server")
	}
}

func TestGetNamingAddr_OneAvailable(t *testing.T) {
	client := &NamingServerClient{config: testConfig}
	client.availableNamingMap.Store("host1", int32(0))
	addr, err := client.getNamingAddr()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "host1" {
		t.Errorf("expected host1, got %s", addr)
	}
}

func TestLookup_FirstCall(t *testing.T) {
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

func TestWatch(t *testing.T) {
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
