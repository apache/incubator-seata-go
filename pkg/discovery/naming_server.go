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
	"errors"
	"fmt"
	gostnet "github.com/dubbogo/gost/net"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"seata.apache.org/seata-go/pkg/util/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	httpPrefix            = "http://"
	healthCheckThreshold  = 1
	longPollTimeoutPeriod = 28 * time.Second
	authorizationHeader   = "Authorization"
	contentTypeJSON       = "application/json"
)

type MetaResponse struct {
	Term        int64     `json:"term"`
	ClusterList []Cluster `json:"clusterList"`
}

type Cluster struct {
	ClusterName string `json:"clusterName"`
	ClusterType string `json:"clusterType"`
	UnitData    []Unit `json:"unitData"`
}

type Unit struct {
	UnitName           string             `json:"unitName"`
	NamingInstanceList []NamingServerNode `json:"namingInstanceList"`
}

type NamingServerNode struct {
	Role        ClusterRole            `json:"role"`
	Term        int64                  `json:"term"`
	Transaction Endpoint               `json:"transaction"`
	Control     Endpoint               `json:"control"`
	Internal    Endpoint               `json:"internal"`
	Group       string                 `json:"group"`
	Version     string                 `json:"version"`
	Metadata    map[string]interface{} `json:"metadata"`
	TimeStamp   int64                  `json:"timeStamp"`
	Weight      float64                `json:"weight"`
	Healthy     bool                   `json:"healthy"`
	Unit        string                 `json:"unit"`
}

type Endpoint struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type ExternalEndpoint struct {
	Host            string `json:"host"`
	ControlPort     int    `json:"controlPort"`
	TransactionPort int    `json:"transactionPort"`
}

type ClusterRole string

const (
	ClusterRoleLeader ClusterRole = "LEADER"
	ClusterRoleMember ClusterRole = "MEMBER"
)

type NamingServerClient struct {
	config          *NamingServerConfig
	logger          *zap.Logger
	mu              sync.Mutex
	instance        *NamingServerClient
	term            int64
	jwtToken        string
	tokenTimeStamp  int64
	isSubscribed    bool
	namingAddrCache string

	availableNamingMap sync.Map
	vgroupAddressMap   sync.Map
	listenerServiceMap sync.Map

	healthCheckTicker *time.Ticker
	closeChan         chan struct{}
	wg                sync.WaitGroup
}

type NamingListener interface {
	OnEvent(vGroup string) error
}

type NamingServerRegistryService struct {
	client *NamingServerClient
}

func (n *NamingServerRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	return n.client.Lookup(key)
}

func (n *NamingServerRegistryService) Close() {
	n.client.Close()
}

func newNamingServerRegistryService(_ *ServiceConfig, cfg *NamingServerConfig) RegistryService {
	client := GetInstance(cfg)
	return &NamingServerRegistryService{
		client: client,
	}
}

func GetInstance(config *NamingServerConfig) *NamingServerClient {
	var instance *NamingServerClient
	var once sync.Once
	once.Do(func() {
		instance = &NamingServerClient{
			config:            config,
			logger:            zap.L().Named("naming-server-client"),
			closeChan:         make(chan struct{}),
			healthCheckTicker: time.NewTicker(time.Duration(config.HeartbeatPeriod) * time.Millisecond),
		}
		instance.initHealthCheck()
	})
	return instance
}

func (c *NamingServerClient) initHealthCheck() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.healthCheckTicker.C:
				urlList := c.getNamingAddrs()
				c.checkAvailableNamingAddr(urlList)
			case <-c.closeChan:
				return
			}
		}
	}()
}

func (c *NamingServerClient) checkAvailableNamingAddr(urlList []string) {
	for _, addr := range urlList {
		isHealthy := c.doHealthCheck(addr)

		val, _ := c.availableNamingMap.LoadOrStore(addr, int32(0))
		failCount := val.(int32)

		if !isHealthy {
			atomic.AddInt32(&failCount, 1)
			c.availableNamingMap.Store(addr, failCount)
			if failCount >= healthCheckThreshold {
				c.logger.Error("naming server offline", zap.String("addr", addr))
			}
		} else {
			if failCount > 0 {
				c.availableNamingMap.Store(addr, int32(0))
				c.logger.Info("naming server online", zap.String("addr", addr))
			}
		}
	}
}

func (c *NamingServerClient) doHealthCheck(addr string) bool {
	checkURL := fmt.Sprintf("%s%s/naming/v1/health", httpPrefix, addr)
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		c.logger.Error("create health check request failed", zap.Error(err))
		return false
	}
	req.Header.Set("Content-Type", contentTypeJSON)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.logger.Error("health check failed", zap.String("addr", addr), zap.Error(err))
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *NamingServerClient) getNamingAddrs() []string {
	return strings.Split(c.config.ServerAddr, ",")
}

func (c *NamingServerClient) Lookup(vGroup string) ([]*ServiceInstance, error) {
	if !c.isSubscribed {
		if err := c.RefreshGroup(vGroup); err != nil {
			return nil, fmt.Errorf("refresh group failed: %w", err)
		}
		listener := &RefreshListener{client: c}
		if err := c.Subscribe(vGroup, listener); err != nil {
			return nil, fmt.Errorf("subscribe failed: %w", err)
		}
	}

	val, ok := c.vgroupAddressMap.Load(vGroup)
	if !ok {
		if err := c.RefreshGroup(vGroup); err != nil {
			return nil, fmt.Errorf("refresh group failed: %w", err)
		}
		val, ok = c.vgroupAddressMap.Load(vGroup)
		if !ok {
			return nil, errors.New("no nodes found for vgroup")
		}
	}

	nodes := val.([]NamingServerNode)

	var instances []*ServiceInstance
	for _, node := range nodes {
		if !node.Healthy {
			continue
		}

		if node.Transaction.Host == "" || node.Transaction.Port <= 0 || node.Transaction.Port > 65535 {
			c.logger.Warn("invalid node address", zap.String("host", node.Transaction.Host), zap.Int("port", node.Transaction.Port))
			continue
		}

		instances = append(instances, &ServiceInstance{
			Addr: node.Transaction.Host,
			Port: node.Transaction.Port,
		})
	}
	return instances, nil
}

func (c *NamingServerClient) RefreshGroup(vGroup string) error {
	namingAddr, err := c.getNamingAddr()
	if err != nil {
		return err
	}

	if c.isTokenExpired() {
		if err := c.RefreshToken(namingAddr); err != nil {
			return err
		}
	}

	params := url.Values{}
	params.Add("vGroup", vGroup)
	params.Add("namespace", c.config.Namespace)

	discoveryURL := fmt.Sprintf("%s%s/naming/v1/discovery?%s", httpPrefix, namingAddr, params.Encode())
	req, err := http.NewRequest(http.MethodGet, discoveryURL, nil)
	if err != nil {
		return err
	}
	if c.jwtToken != "" {
		req.Header.Set(authorizationHeader, c.jwtToken)
	}
	req.Header.Set("Content-Type", contentTypeJSON)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("discovery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discovery failed, status: %d", resp.StatusCode)
	}

	var metaResp MetaResponse
	if err := json.NewDecoder(resp.Body).Decode(&metaResp); err != nil {
		return fmt.Errorf("decode meta response failed: %w", err)
	}

	return c.handleMetadata(&metaResp, vGroup)
}

func (c *NamingServerClient) getNamingAddr() (string, error) {
	if c.namingAddrCache != "" {
		return c.namingAddrCache, nil
	}

	var availableAddrs []string
	c.availableNamingMap.Range(func(key, value interface{}) bool {
		addr := key.(string)
		failCount := value.(int32)
		if failCount < healthCheckThreshold {
			availableAddrs = append(availableAddrs, addr)
		}
		return true
	})

	if len(availableAddrs) == 0 {
		return "", errors.New("no available naming server")
	}

	addr := availableAddrs[rand.RandIntn(len(availableAddrs))]
	c.namingAddrCache = addr
	return addr, nil
}

func (c *NamingServerClient) isTokenExpired() bool {
	if c.config.Username == "" || c.config.Password == "" {
		return false
	}
	ts := atomic.LoadInt64(&c.tokenTimeStamp)
	if ts == 0 {
		return true
	}
	return time.Now().UnixMilli() >= ts+c.config.TokenValidityInMilliseconds
}

func (c *NamingServerClient) RefreshToken(addr string) error {
	if c.config.Username == "" || c.config.Password == "" {
		return nil
	}

	loginURL := fmt.Sprintf("%s%s/api/v1/auth/login", httpPrefix, addr)
	params := url.Values{}
	params.Add("username", c.config.Username)
	params.Add("password", c.config.Password)

	req, err := http.NewRequest(http.MethodPost, loginURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.URL.RawQuery = params.Encode()

	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("authentication failed: invalid credentials")
	}

	var loginResp struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("decode login response failed: %w", err)
	}
	if loginResp.Code != "200" {
		return errors.New("authentication failed: " + loginResp.Code)
	}

	c.jwtToken = loginResp.Data
	atomic.StoreInt64(&c.tokenTimeStamp, time.Now().UnixMilli())
	return nil
}

func (c *NamingServerClient) handleMetadata(metaResp *MetaResponse, vGroup string) error {
	if metaResp.Term > 0 {
		atomic.StoreInt64(&c.term, metaResp.Term)
	}

	var newNodes []NamingServerNode
	for _, cluster := range metaResp.ClusterList {
		for _, unit := range cluster.UnitData {
			for _, node := range unit.NamingInstanceList {
				if (node.Role == ClusterRoleLeader && node.Term >= atomic.LoadInt64(&c.term)) ||
					node.Role == ClusterRoleMember {
					newNodes = append(newNodes, node)
				}
			}
		}
	}

	c.vgroupAddressMap.Store(vGroup, newNodes)
	return nil
}

type RefreshListener struct {
	client *NamingServerClient
}

func (l *RefreshListener) OnEvent(vGroup string) error {
	return l.client.RefreshGroup(vGroup)
}

func (c *NamingServerClient) Subscribe(vGroup string, listener NamingListener) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, _ := c.listenerServiceMap.LoadOrStore(vGroup, []NamingListener{})
	listeners := append(val.([]NamingListener), listener)
	c.listenerServiceMap.Store(vGroup, listeners)

	if !c.isSubscribed {
		c.isSubscribed = true
		c.wg.Add(1)
		go c.watchLoop(vGroup)
	}
	return nil
}

func (c *NamingServerClient) watchLoop(vGroup string) {
	defer c.wg.Done()
	interval := time.Duration(c.config.MetadataMaxAgeMs) * time.Millisecond
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeChan:
			return
		case <-ticker.C:
			changed, err := c.Watch(vGroup)
			if err != nil {
				c.logger.Error("watch failed", zap.Error(err))
				continue
			}
			if changed {
				if err := c.RefreshGroup(vGroup); err != nil {
					c.logger.Error("refresh group failed in watch", zap.Error(err))
					continue
				}
				val, ok := c.listenerServiceMap.Load(vGroup)
				if !ok {
					continue
				}
				for _, listener := range val.([]NamingListener) {
					if err := listener.OnEvent(vGroup); err != nil {
						c.logger.Warn("listener callback failed", zap.Error(err))
					}
				}
			}
		}
	}
}

func (c *NamingServerClient) Watch(vGroup string) (bool, error) {
	namingAddr, err := c.getNamingAddr()
	if err != nil {
		return false, err
	}

	if c.isTokenExpired() {
		if err := c.RefreshToken(namingAddr); err != nil {
			return false, err
		}
	}

	clientIP, err := gostnet.GetLocalIP()
	if err != nil {
		return false, fmt.Errorf("failed to get local IP: %w", err)
	}

	params := url.Values{}
	params.Add("vGroup", vGroup)
	params.Add("clientTerm", strconv.FormatInt(atomic.LoadInt64(&c.term), 10))
	params.Add("timeout", strconv.FormatInt(longPollTimeoutPeriod.Milliseconds(), 10))
	params.Add("clientAddr", clientIP)

	watchURL := fmt.Sprintf("%s%s/naming/v1/watch?%s", httpPrefix, namingAddr, params.Encode())
	req, err := http.NewRequest(http.MethodPost, watchURL, nil)
	if err != nil {
		return false, err
	}
	if c.jwtToken != "" {
		req.Header.Set(authorizationHeader, c.jwtToken)
	}
	req.Header.Set("Content-Type", contentTypeJSON)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("watch request failed: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (c *NamingServerClient) Close() {
	close(c.closeChan)
	c.healthCheckTicker.Stop()
	c.wg.Wait()
	c.logger.Info("naming server client closed")
}
