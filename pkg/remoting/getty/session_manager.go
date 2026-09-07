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

package getty

import (
	"crypto/tls"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	getty "github.com/apache/dubbo-getty"
	gxsync "github.com/dubbogo/gost/sync"

	"seata.apache.org/seata-go/v2/pkg/discovery"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	"seata.apache.org/seata-go/v2/pkg/remoting/loadbalance"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

const (
	maxCheckAliveRetry     = 600
	checkAliveInternal     = 100
	heartBeatRetryTimesKey = "heartbeat-retry-times"
	maxHeartBeatRetryTimes = 3
)

var (
	sessionManager     *SessionManager
	onceSessionManager = &sync.Once{}
)

type SessionManager struct {
	// serverAddress -> rpc_client.Session -> bool
	serverSessions        sync.Map
	allSessions           sync.Map
	sessionSize           int32
	gettyConf             *config.Config
	seataConfig           *config.SeataConfig
	registrySubscription  discovery.RegistrySubscription
	serverClients         sync.Map
	serverAddressMu       sync.RWMutex
	serverAddressSnapshot map[string]struct{}
	serverAddressReady    bool
	startClient           func(*discovery.ServiceInstance) closeableClient
}

type closeableClient interface {
	Close()
}

type serverClientEntry struct {
	mu     sync.Mutex
	client closeableClient
	closed bool
}

func (e *serverClientEntry) setClient(client closeableClient) bool {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		client.Close()
		return false
	}
	e.client = client
	e.mu.Unlock()
	return true
}

func (e *serverClientEntry) Close() {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	e.closed = true
	client := e.client
	e.client = nil
	e.mu.Unlock()

	if client != nil {
		client.Close()
	}
}

func initSessionManager(gettyConfig *config.Config, seataConfig *config.SeataConfig) {
	if sessionManager == nil {
		onceSessionManager.Do(func() {
			sessionManager = &SessionManager{
				allSessions:    sync.Map{},
				serverSessions: sync.Map{},
				gettyConf:      gettyConfig,
				seataConfig:    seataConfig,
			}
			sessionManager.init()
		})
	}
}

func (g *SessionManager) init() {
	g.initWithRegistry(discovery.GetRegistry())
}

func (g *SessionManager) initWithRegistry(registryService discovery.RegistryService) {
	if registryService == nil {
		log.Warn("registry service not initialized")
		return
	}
	if g.subscribeRegistry(registryService) {
		return
	}

	addressList := g.getAvailServerList(registryService)
	if len(addressList) == 0 {
		log.Warn("no have valid seata server list")
	}
	g.refreshServerList(addressList)
}

func (g *SessionManager) subscribeRegistry(registryService discovery.RegistryService) bool {
	subscriber, ok := registryService.(discovery.RegistrySubscriber)
	if !ok {
		return false
	}
	subscription, err := subscriber.Subscribe(g.seataConfig.TxServiceGroup, func(event discovery.RegistryChangeEvent) {
		if event.Key != g.seataConfig.TxServiceGroup {
			return
		}
		g.refreshServerList(event.Instances)
	})
	if err != nil {
		log.Warnf("subscribe registry changes failed: %v", err)
		return false
	}
	g.registrySubscription = subscription
	return true
}

func (g *SessionManager) getAvailServerList(registryService discovery.RegistryService) []*discovery.ServiceInstance {
	instances, err := registryService.Lookup(g.seataConfig.TxServiceGroup)
	if err != nil {
		log.Warnf("lookup seata server list failed: %v", err)
		return nil
	}
	return instances
}

func (g *SessionManager) refreshServerList(instances []*discovery.ServiceInstance) {
	servers := make(map[string]*discovery.ServiceInstance, len(instances))
	for _, instance := range instances {
		if instance == nil || instance.Addr == "" || instance.Port <= 0 {
			continue
		}
		clone := &discovery.ServiceInstance{Addr: instance.Addr, Port: instance.Port}
		servers[serverAddress(clone)] = clone
	}

	removedAddresses := g.replaceServerAddressSnapshot(servers)
	for _, address := range removedAddresses {
		g.releaseServerAddress(address)
	}

	for _, instance := range servers {
		g.ensureServerClient(instance)
	}
}

func (g *SessionManager) replaceServerAddressSnapshot(servers map[string]*discovery.ServiceInstance) []string {
	g.serverAddressMu.Lock()
	var removedAddresses []string
	for address := range g.serverAddressSnapshot {
		if _, ok := servers[address]; !ok {
			removedAddresses = append(removedAddresses, address)
		}
	}
	g.serverAddressSnapshot = make(map[string]struct{}, len(servers))
	for address := range servers {
		g.serverAddressSnapshot[address] = struct{}{}
	}
	g.serverAddressReady = true
	g.serverAddressMu.Unlock()
	return removedAddresses
}

func (g *SessionManager) ensureServerClient(instance *discovery.ServiceInstance) {
	address := serverAddress(instance)
	entry := &serverClientEntry{}
	if _, loaded := g.serverClients.LoadOrStore(address, entry); loaded {
		return
	}
	var client closeableClient
	if g.startClient != nil {
		client = g.startClient(instance)
	} else {
		client = g.startGettyClient(instance)
	}
	if client == nil {
		g.serverClients.CompareAndDelete(address, entry)
		return
	}
	if !entry.setClient(client) {
		return
	}
	if !g.isServerAddressAvailable(address) || !g.isServerClientEntryCurrent(address, entry) {
		g.releaseServerClientEntry(address, entry)
		return
	}
}

func (g *SessionManager) isServerClientEntryCurrent(address string, entry *serverClientEntry) bool {
	current, ok := g.serverClients.Load(address)
	return ok && current == entry
}

func (g *SessionManager) releaseServerClientEntry(address string, entry *serverClientEntry) {
	g.serverClients.CompareAndDelete(address, entry)
	entry.Close()
}

func (g *SessionManager) startGettyClient(instance *discovery.ServiceInstance) getty.Client {
	gettyClient := getty.NewTCPClient(
		getty.WithServerAddress(serverAddress(instance)),
		// todo if read c.gettyConf.ConnectionNum, will cause the connect to fail
		getty.WithConnectionNumber(1),
		getty.WithReconnectInterval(g.gettyConf.ReconnectInterval),
		getty.WithClientTaskPool(gxsync.NewTaskPoolSimple(0)),
	)
	go gettyClient.RunEventLoop(g.newSession)
	return gettyClient
}

func (g *SessionManager) releaseServerAddress(address string) {
	if clientAny, loaded := g.serverClients.LoadAndDelete(address); loaded {
		if client, ok := clientAny.(closeableClient); ok && client != nil {
			client.Close()
		}
	}
	if sessionsAny, ok := g.serverSessions.LoadAndDelete(address); ok {
		sessions := sessionsAny.(*sync.Map)
		sessions.Range(func(key, _ interface{}) bool {
			if session, ok := key.(getty.Session); ok {
				g.releaseSession(session)
			}
			return true
		})
	}
}

func serverAddress(instance *discovery.ServiceInstance) string {
	return net.JoinHostPort(instance.Addr, strconv.Itoa(instance.Port))
}

func (g *SessionManager) setSessionConfig(session getty.Session) {
	session.SetName(g.gettyConf.SessionConfig.SessionName)
	session.SetMaxMsgLen(g.gettyConf.SessionConfig.MaxMsgLen)
	session.SetPkgHandler(rpcPkgHandler)
	session.SetEventListener(GetGettyClientHandlerInstance())
	session.SetReadTimeout(g.gettyConf.SessionConfig.TCPReadTimeout)
	session.SetWriteTimeout(g.gettyConf.SessionConfig.TCPWriteTimeout)
	session.SetCronPeriod((int)(g.gettyConf.SessionConfig.CronPeriod.Milliseconds()))
	session.SetWaitTime(g.gettyConf.SessionConfig.WaitTimeout)
	session.SetAttribute(heartBeatRetryTimesKey, 0)
}

func (g *SessionManager) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
		err     error
	)

	if g.gettyConf.SessionConfig.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}
	if _, ok = session.Conn().(*tls.Conn); ok {
		g.setSessionConfig(session)
		log.Debugf("server accepts new tls session:%s\n", session.Stat())
		return nil
	}
	if _, ok = session.Conn().(*net.TCPConn); !ok {
		panic(fmt.Sprintf("%s, session.conn{%#v} is not a tcp connection\n", session.Stat(), session.Conn()))
	}

	if _, ok = session.Conn().(*tls.Conn); !ok {
		if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
			return fmt.Errorf("%s, session.conn{%#v} is not tcp connection", session.Stat(), session.Conn())
		}

		if err = tcpConn.SetNoDelay(g.gettyConf.SessionConfig.TCPNoDelay); err != nil {
			return err
		}
		if err = tcpConn.SetKeepAlive(g.gettyConf.SessionConfig.TCPKeepAlive); err != nil {
			return err
		}
		if g.gettyConf.SessionConfig.TCPKeepAlive {
			if err = tcpConn.SetKeepAlivePeriod(g.gettyConf.SessionConfig.KeepAlivePeriod); err != nil {
				return err
			}
		}
		if err = tcpConn.SetReadBuffer(g.gettyConf.SessionConfig.TCPRBufSize); err != nil {
			return err
		}
		if err = tcpConn.SetWriteBuffer(g.gettyConf.SessionConfig.TCPWBufSize); err != nil {
			return err
		}
	}

	g.setSessionConfig(session)
	log.Debugf("rpc_client new session:%s\n", session.Stat())

	return nil
}

func (g *SessionManager) selectSession(msg interface{}) getty.Session {
	sessions := g.selectableSessions()
	selected := loadbalance.Select(loadbalance.GetLoadBalanceConfig().Type, sessions, g.getXid(msg))
	session, ok := selected.(getty.Session)
	if ok && session != nil {
		return session
	}

	if selectableConnectionCount(sessions) == 0 {
		ticker := time.NewTicker(time.Duration(checkAliveInternal) * time.Millisecond)
		defer ticker.Stop()
		for i := 0; i < maxCheckAliveRetry; i++ {
			<-ticker.C
			sessions = g.selectableSessions()
			selected = loadbalance.Select(loadbalance.GetLoadBalanceConfig().Type, sessions, g.getXid(msg))
			session, ok = selected.(getty.Session)
			if ok && session != nil {
				return session
			}
		}
	}
	return nil
}

func (g *SessionManager) selectableSessions() *sync.Map {
	sessions := &sync.Map{}
	g.allSessions.Range(func(key, value interface{}) bool {
		session, ok := key.(getty.Session)
		if !ok {
			return true
		}
		if session.IsClosed() {
			g.releaseSession(session)
			return true
		}
		if g.isServerAddressAvailable(session.RemoteAddr()) {
			sessions.Store(session, value)
		}
		return true
	})
	return sessions
}

func (g *SessionManager) isServerAddressAvailable(address string) bool {
	g.serverAddressMu.RLock()
	defer g.serverAddressMu.RUnlock()

	if !g.serverAddressReady {
		return true
	}
	_, ok := g.serverAddressSnapshot[address]
	return ok
}

func (g *SessionManager) getXid(msg interface{}) string {
	var xid string
	if tmpMsg, ok := msg.(message.AbstractGlobalEndRequest); ok {
		xid = tmpMsg.Xid
	} else if tmpMsg, ok := msg.(message.GlobalBeginRequest); ok {
		xid = tmpMsg.TransactionName
	} else if tmpMsg, ok := msg.(message.BranchRegisterRequest); ok {
		xid = tmpMsg.Xid
	} else if tmpMsg, ok := msg.(message.BranchReportRequest); ok {
		xid = tmpMsg.Xid
	} else {
		msgType := reflect.TypeOf(msg)
		msgValue := reflect.ValueOf(msg)
		if msgType.Kind() == reflect.Ptr {
			msgValue = msgValue.Elem()
		}
		xid = msgValue.FieldByName("Xid").String()
	}
	return xid
}

func (g *SessionManager) releaseSession(session getty.Session) {
	if session == nil {
		return
	}
	if _, loaded := g.allSessions.LoadAndDelete(session); !loaded {
		return
	}
	if m, ok := g.serverSessions.Load(session.RemoteAddr()); ok {
		sMap := m.(*sync.Map)
		sMap.Delete(session)
	}
	if !session.IsClosed() {
		session.Close()
	}
	atomic.AddInt32(&g.sessionSize, -1)
}

func (g *SessionManager) registerSession(session getty.Session) {
	if !g.isServerAddressAvailable(session.RemoteAddr()) {
		log.Warnf("skip session for removed server address: %s", session.RemoteAddr())
		session.Close()
		return
	}
	if _, loaded := g.allSessions.LoadOrStore(session, true); loaded {
		return
	}
	m, _ := g.serverSessions.LoadOrStore(session.RemoteAddr(), &sync.Map{})
	sMap := m.(*sync.Map)
	sMap.Store(session, true)
	atomic.AddInt32(&g.sessionSize, 1)
}

func selectableConnectionCount(connections *sync.Map) int {
	count := 0
	connections.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
