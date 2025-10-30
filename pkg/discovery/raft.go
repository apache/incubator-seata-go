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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"seata.apache.org/seata-go/pkg/discovery/metadata"
	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	controlEndpoint     = "control"
	transactionEndpoint = "transaction"
)

type RaftRegistryService struct {
	cfg                            *RaftConfig
	metadata                       *metadata.Metadata
	initAddresses                  sync.Map // clusterName -> []*ServiceInstance
	aliveNodes                     sync.Map // transactionServiceGroup -> []*ServiceInstance
	vgroupMapping                  map[string]string
	namingserverAddress            string
	username                       string
	password                       string
	jwtToken                       string
	tokenTimestamp                 int64
	currentTransactionServiceGroup string
	currentTransactionClusterName  string
	mu                             sync.RWMutex
	stopCh                         chan struct{}
	refreshOnce                    sync.Once
	httpClient                     *http.Client
	random                         *rand.Rand
}

func NewRaftRegistryService(config *ServiceConfig, raftConfig *RegistryConfig) *RaftRegistryService {
	vgroupMapping := config.VgroupMapping

	r := &RaftRegistryService{
		cfg:                 &raftConfig.Raft,
		metadata:            metadata.NewMetadata(),
		initAddresses:       sync.Map{},
		aliveNodes:          sync.Map{},
		vgroupMapping:       vgroupMapping,
		namingserverAddress: raftConfig.NamingserverAddr,
		username:            raftConfig.Username,
		password:            raftConfig.Password,
		stopCh:              make(chan struct{}),
		httpClient:          &http.Client{},
		tokenTimestamp:      -1,
		random:              rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return r
}

func (r *RaftRegistryService) Lookup(key string) ([]*ServiceInstance, error) {
	clusterName := r.vgroupMapping[key]
	if clusterName == "" {
		return nil, fmt.Errorf("cluster doesnt exist")
	}
	r.currentTransactionServiceGroup = key
	r.currentTransactionClusterName = clusterName

	if !r.metadata.ContainsGroup(clusterName) {
		if _, ok := r.loadInitAddresses(clusterName); !ok && r.cfg.ServerAddr != "" {
			addrs := strings.Split(r.cfg.ServerAddr, ",")
			list := make([]*ServiceInstance, 0, len(addrs))
			for _, addr := range addrs {
				h, p, err := net.SplitHostPort(strings.TrimSpace(addr))
				if err != nil {
					log.Infof("invalid init server addr: %s, err: %v", addr, err)
					continue
				}
				port, err := strconv.Atoi(p)
				if err != nil {
					log.Errorf("invalid port: %s", p)
					continue
				}
				list = append(list, &ServiceInstance{Addr: h, Port: port})
			}
			if len(list) == 0 {
				return nil, fmt.Errorf("invalid service group/key: %s", key)
			}
			r.initAddresses.Store(clusterName, list)

			if err := r.refreshToken(); err != nil {
				return nil, err
			}

			err := r.acquireClusterMetaData(clusterName, "")
			if err != nil {
				return nil, err
			}
			r.startQueryMetadata()
		}
	}
	return r.getServiceInstances(clusterName, "")
}

func (r *RaftRegistryService) aliveLookup(transactionServiceGroup string) ([]*ServiceInstance, error) {
	clusterName := r.vgroupMapping[transactionServiceGroup]
	if clusterName == "" {
		return nil, fmt.Errorf("cluster doesnt exist")
	}
	leader := r.metadata.GetLeader(clusterName)
	if leader != nil {
		endpoint, err := r.selectEndpoint(transactionEndpoint, leader)
		if err != nil {
			return nil, err
		}
		return []*ServiceInstance{endpoint}, nil
	}
	return r.getServiceInstances(clusterName, "")
}

func (r *RaftRegistryService) getServiceInstances(clusterName, group string) ([]*ServiceInstance, error) {
	nodes := r.metadata.GetNodes(clusterName, group)
	if len(nodes) > 0 {
		instances := make([]*ServiceInstance, 0, len(nodes))
		for _, n := range nodes {
			inst, _ := r.selectEndpoint(transactionEndpoint, n)
			if inst != nil {
				instances = append(instances, inst)
			}
		}
		return instances, nil
	}
	return nil, nil
}

func (r *RaftRegistryService) RefreshAliveLookup(transactionServiceGroup string, aliveAddress []*ServiceInstance) ([]*ServiceInstance, error) {
	clusterName := r.vgroupMapping[transactionServiceGroup]
	if clusterName == "" {
		return nil, fmt.Errorf("cluster not found for serviceGroup=%s", transactionServiceGroup)
	}

	leader := r.metadata.GetLeader(clusterName)
	if leader == nil {
		return nil, fmt.Errorf("leader not found for cluster=%s", clusterName)
	}

	leaderEndpoint, err := r.selectEndpoint(transactionEndpoint, leader)
	if err != nil {
		return nil, err
	}

	var result []*ServiceInstance
	for _, addr := range aliveAddress {
		if addr.Port != leaderEndpoint.Port || addr.Addr != leaderEndpoint.Addr {
			result = append(result, addr)
		}
	}

	r.aliveNodes.Store(transactionServiceGroup, result)
	return result, nil
}

func (r *RaftRegistryService) Close() {
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
}

func (r *RaftRegistryService) selectEndpoint(t string, n *metadata.Node) (*ServiceInstance, error) {
	switch t {
	case controlEndpoint:
		return &ServiceInstance{
			Addr: n.Control.Host,
			Port: n.Control.Port,
		}, nil
	case transactionEndpoint:
		return &ServiceInstance{
			Addr: n.Transaction.Host,
			Port: n.Transaction.Port,
		}, nil
	default:
		return nil, fmt.Errorf("SelectEndpoint is not support type: %s", t)
	}
}

func (r *RaftRegistryService) startQueryMetadata() {
	r.refreshOnce.Do(func() {
		go func() {
			metadataMaxAge := int64(30000)
			if r.cfg.MetadataMaxAgeMs > 0 {
				metadataMaxAge = r.cfg.MetadataMaxAgeMs
			}
			currentTime := time.Now().UnixMilli()
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-r.stopCh:
					log.Info("raft registry service stopped")
					return
				case <-ticker.C:
					func() {
						shouldFetch := time.Now().UnixMilli()-currentTime > metadataMaxAge
						if !shouldFetch {
							ok, err := r.watch()
							if err != nil {
								log.Errorf("watch error: %v", err)
								shouldFetch = true
							} else {
								shouldFetch = ok
							}
						}

						if shouldFetch {
							clusterName := r.currentTransactionClusterName
							groups := r.metadata.Groups(clusterName)
							if len(groups) == 0 {
								groups = append(groups, "")
							}
							for _, g := range groups {
								err := r.acquireClusterMetaData(clusterName, g)
								if err != nil {
									log.Errorf("acquire cluster metadata failed: cluster=%s group=%s err=%v", clusterName, g, err)
								}
							}

							currentTime = time.Now().UnixMilli()
						}
					}()
				}
			}
		}()
	})
}

func (r *RaftRegistryService) watch() (bool, error) {
	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	clusterNames := r.clusterNamesFromInit()
	for _, clusterName := range clusterNames {
		groupTerms := r.metadata.GetClusterTerm(clusterName)
		if groupTerms == nil {
			groupTerms = map[string]int64{"": 0}
		}
		for group := range groupTerms {
			tcAddress, err := r.queryHttpAddress(clusterName, group)
			if err != nil {
				log.Infof("no tc address to watch for cluster %s: %v", clusterName, err)
				continue
			}
			if r.isTokenExpired() {
				if err = r.refreshToken(); err != nil {
					return false, err
				}
			}
			if r.jwtToken != "" {
				header["Authorization"] = r.jwtToken
			}

			form := url.Values{}
			for k, v := range groupTerms {
				form.Set(k, strconv.FormatInt(v, 10))
			}

			endpoint := fmt.Sprintf("http://%s/metadata/v1/watch", tcAddress)
			req, err := http.NewRequest("POST", endpoint, strings.NewReader(form.Encode()))
			if err != nil {
				return false, err
			}
			for hk, hv := range header {
				req.Header.Set(hk, hv)
			}
			resp, err := r.doRequest(req, 30*time.Second)
			if err != nil {
				log.Errorf("watch cluster node: %s, fail: %v", tcAddress, err)
				return false, err
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusUnauthorized {
				return false, errors.New("authentication failed: missing username/password")
			}
			return resp.StatusCode == http.StatusOK, nil
		}
	}
	return false, nil
}

func (r *RaftRegistryService) acquireClusterMetaData(clusterName, group string) error {
	tcAddress, err := r.queryHttpAddress(clusterName, group)
	if err != nil {
		return err
	}
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	if r.isTokenExpired() {
		if err = r.refreshToken(); err != nil {
			return err
		}
	}
	if r.jwtToken != "" {
		headers["Authorization"] = r.jwtToken
	}
	u := fmt.Sprintf("http://%s/metadata/v1/cluster", tcAddress)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("group", group)
	req.URL.RawQuery = q.Encode()
	for hk, hv := range headers {
		req.Header.Set(hk, hv)
	}

	resp, err := r.doRequest(req, 1*time.Second)
	if err != nil {
		return fmt.Errorf("http get cluster failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var mr metadata.MetadataResponse
		if err = json.Unmarshal(body, &mr); err != nil {
			return fmt.Errorf("unmarshal metadataResponse failed: %w", err)
		}
		r.metadata.RefreshMetadata(clusterName, mr)
		return nil
	} else if resp.StatusCode == http.StatusUnauthorized {
		if err = r.refreshToken(); err != nil {
			return err
		}
		return fmt.Errorf("authentication failed! you should configure the correct username and password")
	}
	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

/* -------------------- Token management -------------------- */

func (r *RaftRegistryService) isTokenExpired() bool {
	r.mu.RLock()
	ts := r.tokenTimestamp
	r.mu.RUnlock()
	if ts == -1 {
		return true
	}
	valid := int64(29 * 60 * 1000)
	if r.cfg.TokenValidityInMilliseconds > 0 {
		valid = r.cfg.TokenValidityInMilliseconds
	}
	expireTime := ts + valid
	return time.Now().UnixMilli() >= expireTime
}

func (r *RaftRegistryService) refreshToken() error {
	address := r.namingserverAddress
	body, _ := json.Marshal(map[string]string{
		"username": r.username,
		"password": r.password,
	})
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/api/v1/auth/login", address), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.doRequest(req, 1*time.Second)
	if err != nil {
		return fmt.Errorf("refresh token failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("authentication failed when refresh token")
	}
	respBody, _ := io.ReadAll(resp.Body)
	var node map[string]interface{}
	if err = json.Unmarshal(respBody, &node); err != nil {
		return fmt.Errorf("invalid auth response: %w", err)
	}
	code, _ := node["code"].(string)
	if code != "" && code != "200" {
		return errors.New("authentication failed! you should configure the correct username and password")
	}
	dataVal := node["data"].(string)
	r.mu.Lock()
	r.jwtToken = dataVal
	r.tokenTimestamp = time.Now().UnixMilli()
	r.mu.Unlock()
	return nil
}

func (r *RaftRegistryService) queryHttpAddress(clusterName, group string) (string, error) {
	var addressList []string

	nodes := r.metadata.GetNodes(clusterName, group)

	formatAddr := func(addr string, port int) string {
		return fmt.Sprintf("%s:%d", addr, port)
	}

	appendControlAddr := func(node *metadata.Node) error {
		endpoint, err := r.selectEndpoint(controlEndpoint, node)
		if err != nil {
			return fmt.Errorf("failed to select control endpoint: %v", err)
		}
		addressList = append(addressList, formatAddr(endpoint.Addr, endpoint.Port))
		return nil
	}

	if len(nodes) > 0 {
		currentServiceGroup := r.currentTransactionServiceGroup
		if aliveAny, ok := r.aliveNodes.Load(currentServiceGroup); ok {
			aliveNodes, ok := aliveAny.([]*ServiceInstance)
			if !ok {
				log.Errorf("invalid type for alive nodes, expected []*ServiceInstance")
				return "", fmt.Errorf("invalid alive nodes type for service group %s", currentServiceGroup)
			}

			if len(aliveNodes) == 0 {
				for _, node := range nodes {
					if err := appendControlAddr(node); err != nil {
						return "", err
					}
				}
			} else {
				m := make(map[string]*metadata.Node)
				for _, node := range nodes {
					inetSocketAddress, _ := r.selectEndpoint(transactionEndpoint, node)
					m[formatAddr(inetSocketAddress.Addr, inetSocketAddress.Port)] = node
				}

				for _, alive := range aliveNodes {
					addrKey := formatAddr(alive.Addr, alive.Port)
					node, exists := m[addrKey]
					if !exists {
						continue
					}

					contrEndpoint, err := r.selectEndpoint(controlEndpoint, node)
					if err != nil {
						log.Errorf("failed to select control endpoint for node: %v", err)
						continue
					}
					addressList = append(addressList, formatAddr(contrEndpoint.Addr, contrEndpoint.Port))
				}
			}
		} else {
			for _, node := range nodes {
				if err := appendControlAddr(node); err != nil {
					return "", err
				}
			}
		}
	} else {
		if initAny, ok := r.initAddresses.Load(clusterName); ok {
			for _, inetSocketAddress := range initAny.([]*ServiceInstance) {
				addressList = append(addressList, formatAddr(inetSocketAddress.Addr, inetSocketAddress.Port))
			}
		}
	}

	if len(addressList) == 0 {
		return "", fmt.Errorf("no available address for cluster=%s group=%s", clusterName, group)
	}
	return addressList[r.random.Intn(len(addressList))], nil
}

func (r *RaftRegistryService) clusterNamesFromInit() []string {
	var names []string
	r.initAddresses.Range(func(key, _ any) bool {
		if k, ok := key.(string); ok {
			names = append(names, k)
		}
		return true
	})
	return names
}

func (r *RaftRegistryService) doRequest(req *http.Request, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), timeout)
	defer cancel()

	req = req.WithContext(ctx)
	return r.httpClient.Do(req)
}

func (r *RaftRegistryService) loadInitAddresses(clusterName string) ([]*ServiceInstance, bool) {
	if val, ok := r.initAddresses.Load(clusterName); ok {
		if list, ok := val.([]*ServiceInstance); ok {
			return list, true
		}
	}
	return nil, false
}
