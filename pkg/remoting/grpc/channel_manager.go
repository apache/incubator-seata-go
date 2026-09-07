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

package grpc

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/discovery"
	grpc2 "seata.apache.org/seata-go/v2/pkg/integration/grpc"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/remoting/loadbalance"
	"seata.apache.org/seata-go/v2/pkg/util/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	defaultSendChBuffer    = 1000
	maxCheckAliveRetry     = 600
	checkAliveInternal     = 100
	maxHeartBeatRetryTimes = 3
)

var (
	onceChannelManager sync.Once
	channelManager     *ChannelManager
	//msgId:error
	msgSendTrackers sync.Map
)

type ChannelManager struct {
	//addr:map[Channel]bool
	serverChannels sync.Map
	//stream:bool
	allChannels           sync.Map
	clientSize            int32
	config                *config.Config
	seataConfig           *config.SeataConfig
	registrySubscription  discovery.RegistrySubscription
	startedAddresses      sync.Map
	serverAddressMu       sync.RWMutex
	serverAddressSnapshot map[string]struct{}
	serverAddressReady    bool
	startChannel          func(*discovery.ServiceInstance)
}

type channelStartEntry struct{}

func initChannelManager(grpcConfig *config.Config) {
	if channelManager == nil {
		onceChannelManager.Do(func() {
			channelManager = &ChannelManager{
				config:         grpcConfig,
				seataConfig:    config.GetSeataConfig(),
				allChannels:    sync.Map{},
				serverChannels: sync.Map{},
			}
			channelManager.init()
		})
	}
}

func (g *ChannelManager) init() {
	g.initWithRegistry(discovery.GetRegistry())
}

func (g *ChannelManager) initWithRegistry(registryService discovery.RegistryService) {
	if registryService == nil {
		log.Warn("registry service not initialized")
		return
	}
	if g.seataConfig == nil || g.seataConfig.TxServiceGroup == "" {
		log.Warn("transaction service group not initialized")
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

func (g *ChannelManager) subscribeRegistry(registryService discovery.RegistryService) bool {
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

func (g *ChannelManager) refreshServerList(instances []*discovery.ServiceInstance) {
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
		g.releaseChannelByAddr(address)
	}

	for _, instance := range servers {
		g.ensureChannel(instance)
	}
}

func (g *ChannelManager) replaceServerAddressSnapshot(servers map[string]*discovery.ServiceInstance) []string {
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

func (g *ChannelManager) ensureChannel(instance *discovery.ServiceInstance) {
	address := serverAddress(instance)
	entry := &channelStartEntry{}
	if _, loaded := g.startedAddresses.LoadOrStore(address, entry); loaded {
		return
	}
	if g.startChannel != nil {
		g.startChannel(instance)
		return
	}
	go g.startGrpcChannel(instance, entry)
}

func (g *ChannelManager) startGrpcChannel(instance *discovery.ServiceInstance, entry *channelStartEntry) {
	addr := serverAddress(instance)
	if !g.isServerAddressAvailable(addr) || !g.isChannelStartCurrent(addr, entry) {
		g.deleteChannelStart(addr, entry)
		return
	}

	conn, err := g.newConn(addr)
	if err != nil {
		log.Errorf("failed to dial gRPC addr %s: %v", addr, err)
		g.deleteChannelStart(addr, entry)
		return
	}

	regLock := sync.Mutex{}
	registered := atomic.Bool{}
	// todo if read g.config.ConnectionNum, will cause the connect to fail
	channel := &Channel{
		addr:       addr,
		conn:       conn,
		sendCh:     make(chan *pb.GrpcMessageProto, defaultSendChBuffer),
		closeCh:    make(chan struct{}),
		wg:         sync.WaitGroup{},
		mu:         sync.Mutex{},
		regLock:    &regLock,
		registered: &registered,
	}

	channel, err = g.initChannel(channel)
	if err != nil {
		log.Errorf("failed to create gRPC stream error: %v", err)
		_ = conn.Close()
		g.deleteChannelStart(addr, entry)
		return
	}
	if !g.isServerAddressAvailable(addr) || !g.isChannelStartCurrent(addr, entry) {
		channel.close()
		g.deleteChannelStart(addr, entry)
		return
	}
	if !g.registerChannel(channel) {
		_ = conn.Close()
		g.deleteChannelStart(addr, entry)
		return
	}
	if err = g.registerTm(addr); err != nil {
		log.Errorf("%v", err)
		g.releaseChannelByAddr(addr)
	}
}

func (g *ChannelManager) isChannelStartCurrent(address string, entry *channelStartEntry) bool {
	current, ok := g.startedAddresses.Load(address)
	return ok && current == entry
}

func (g *ChannelManager) deleteChannelStart(address string, entry *channelStartEntry) {
	g.startedAddresses.CompareAndDelete(address, entry)
}

func (g *ChannelManager) getAvailServerList(registryService discovery.RegistryService) []*discovery.ServiceInstance {
	instances, err := registryService.Lookup(g.seataConfig.TxServiceGroup)
	if err != nil {
		log.Warnf("lookup seata server list failed: %v", err)
		return nil
	}
	return instances
}

func (g *ChannelManager) newConn(addr string) (conn *grpc.ClientConn, err error) {
	var opts []grpc.DialOption

	opts = append(opts,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(g.config.StreamConfig.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(g.config.StreamConfig.MaxSendMsgSize),
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), g.config.StreamConfig.DialTimeout)
	defer cancel()

	kaParams := keepalive.ClientParameters{
		Time:                g.config.StreamConfig.KeepAliveTime,
		Timeout:             g.config.StreamConfig.KeepAliveTimeout,
		PermitWithoutStream: g.config.StreamConfig.PermitWithoutStream,
	}
	opts = append(opts, grpc.WithKeepaliveParams(kaParams))
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.WithStreamInterceptor(grpc2.ClientTransactionStreamInterceptor))
	opts = append(opts, grpc.WithInitialConnWindowSize(1*1024*1024))
	return grpc.DialContext(ctx, addr, opts...)
}

func (g *ChannelManager) initChannel(channel *Channel) (*Channel, error) {
	ctx := context.Background()
	client := pb.NewSeataServiceClient(channel.conn)
	stream, err := client.SendRequest(ctx)
	if err != nil {
		return nil, err
	}
	channel.client = client
	channel.stream = stream

	channel.wg.Add(2)

	go channel.sendProcessor()
	go GetGrpcClientHandlerInstance().monitorStreamHealth(context.Background(), channel, g.config.StreamConfig.HeartbeatInterval)
	go GetGrpcClientHandlerInstance().StartReceiveLoop(ctx, channel)

	return channel, nil
}

func (g *ChannelManager) registerTm(addr string) error {
	conf := config.GetSeataConfig()

	request := &pb.RegisterTMRequestProto{
		AbstractIdentifyRequest: &pb.AbstractIdentifyRequestProto{
			Version:                 constant.SeataVersion,
			ApplicationId:           conf.ApplicationID,
			TransactionServiceGroup: conf.TxServiceGroup,
		},
	}
	err := GetGrpcRemotingClient().SendAsyncRequest(request)
	if err != nil {
		g.releaseChannelByAddr(addr)
		return fmt.Errorf("register TM error: {%#v}", err.Error())
	}
	return nil
}

func (g *ChannelManager) selectChannel(msg interface{}) *Channel {
	channels := g.selectableChannels()
	channel := g.selectAvailableChannel(channels, msg)
	if channel != nil {
		return channel
	}

	if selectableConnectionCount(channels) == 0 {
		ticker := time.NewTicker(time.Duration(checkAliveInternal) * time.Millisecond)
		defer ticker.Stop()
		for i := 0; i < maxCheckAliveRetry; i++ {
			<-ticker.C
			channels = g.selectableChannels()
			channel = g.selectAvailableChannel(channels, msg)
			if channel != nil {
				return channel
			}
		}
	}
	return nil
}

func (g *ChannelManager) selectAvailableChannel(channels *sync.Map, msg interface{}) *Channel {
	selected := loadbalance.Select(loadbalance.GetLoadBalanceConfig().Type, channels, g.getXid(msg))
	channel, ok := selected.(*Channel)
	if ok && g.isChannelSelectable(channels, channel) {
		return channel
	}

	// Some load balancers cache their result. Re-select from the current
	// snapshot if a cached result points to a removed server address.
	selected = loadbalance.Select("RandomLoadBalance", channels, g.getXid(msg))
	channel, ok = selected.(*Channel)
	if ok && g.isChannelSelectable(channels, channel) {
		return channel
	}
	return nil
}

func (g *ChannelManager) isChannelSelectable(channels *sync.Map, channel *Channel) bool {
	if channel == nil || channel.IsClosed() || !g.isServerAddressAvailable(channel.addr) {
		return false
	}
	_, ok := channels.Load(channel)
	return ok
}

func (g *ChannelManager) getXid(msg interface{}) string {
	switch tmpMsg := msg.(type) {
	case *pb.AbstractGlobalEndRequestProto:
		return tmpMsg.Xid
	case *pb.GlobalBeginRequestProto:
		return tmpMsg.TransactionName
	case *pb.BranchRegisterRequestProto:
		return tmpMsg.Xid
	case *pb.BranchReportRequestProto:
		return tmpMsg.Xid
	}

	msgValue := reflect.ValueOf(msg)
	if !msgValue.IsValid() {
		return ""
	}
	if msgValue.Kind() == reflect.Ptr {
		if msgValue.IsNil() {
			return ""
		}
		msgValue = msgValue.Elem()
	}
	if msgValue.Kind() != reflect.Struct {
		return ""
	}
	if field := msgValue.FieldByName("Xid"); field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	if field := msgValue.FieldByName("TransactionName"); field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	return ""
}

func (g *ChannelManager) releaseChannel(channel *Channel) {
	if _, loaded := g.allChannels.LoadAndDelete(channel); !loaded {
		return
	}
	if m, ok := g.serverChannels.Load(channel.addr); ok {
		sMap := m.(*sync.Map)
		sMap.Delete(channel)
	}
	if !channel.IsClosed() {
		channel.close()
	}
	atomic.AddInt32(&g.clientSize, -1)
	if g.getAllChannelIsClosedByAddr(channel.addr) {
		g.startedAddresses.Delete(channel.addr)
	}
}

func (g *ChannelManager) registerChannel(channel *Channel) bool {
	if !g.isServerAddressAvailable(channel.addr) {
		log.Warnf("skip channel for removed server address: %s", channel.addr)
		if _, loaded := g.allChannels.Load(channel); loaded {
			g.releaseChannel(channel)
		} else if !channel.IsClosed() {
			channel.close()
		}
		return false
	}
	if _, loaded := g.allChannels.LoadOrStore(channel, true); loaded {
		return true
	}
	m, _ := g.serverChannels.LoadOrStore(channel.addr, &sync.Map{})
	sMap := m.(*sync.Map)
	sMap.Store(channel, true)
	if _, loaded := g.startedAddresses.Load(channel.addr); !loaded {
		g.startedAddresses.Store(channel.addr, &channelStartEntry{})
	}
	atomic.AddInt32(&g.clientSize, 1)
	return true
}

func (g *ChannelManager) releaseChannelByAddr(addr string) {
	g.allChannels.Range(func(key, value any) bool {
		ch := key.(*Channel)
		if ch.addr == addr {
			g.releaseChannel(ch)
		}
		return true
	})
	g.serverChannels.Delete(addr)
	g.startedAddresses.Delete(addr)
}

func (g *ChannelManager) getAllChannelIsClosedByAddr(addr string) bool {
	flag := true
	g.allChannels.Range(func(key, value any) bool {
		ch := key.(*Channel)
		if ch.addr == addr {
			if !ch.IsClosed() {
				flag = false
				return false
			}
		}
		return true
	})
	return flag
}

func (g *ChannelManager) selectableChannels() *sync.Map {
	channels := &sync.Map{}
	g.allChannels.Range(func(key, value interface{}) bool {
		channel, ok := key.(*Channel)
		if !ok {
			return true
		}
		if channel.IsClosed() {
			g.releaseChannel(channel)
			return true
		}
		if g.isServerAddressAvailable(channel.addr) {
			channels.Store(channel, value)
		}
		return true
	})
	return channels
}

func selectableConnectionCount(connections *sync.Map) int {
	count := 0
	connections.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (g *ChannelManager) isServerAddressAvailable(address string) bool {
	g.serverAddressMu.RLock()
	defer g.serverAddressMu.RUnlock()

	if !g.serverAddressReady {
		return true
	}
	_, ok := g.serverAddressSnapshot[address]
	return ok
}

func serverAddress(instance *discovery.ServiceInstance) string {
	return net.JoinHostPort(instance.Addr, strconv.Itoa(instance.Port))
}
