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

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/discovery"
	grpc2 "seata.apache.org/seata-go/pkg/integration/grpc"
	"seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/pkg/remoting/loadbalance"
	"seata.apache.org/seata-go/pkg/util/log"

	"google.golang.org/grpc"
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
	allChannels sync.Map
	clientSize  int32
	config      *config.Config
}

func initChannelManager(config *config.Config) {
	if channelManager == nil {
		onceChannelManager.Do(func() {
			channelManager = &ChannelManager{
				config:         config,
				allChannels:    sync.Map{},
				serverChannels: sync.Map{},
			}
			channelManager.init()
		})
	}
}

func (g *ChannelManager) init() {
	addressList := g.getAvailServerList()
	if len(addressList) == 0 {
		log.Warn("no have valid seata server list")
	}
	for _, address := range addressList {
		addr := net.JoinHostPort(address.Addr, strconv.Itoa(address.Port))
		if conn, err := g.newConn(addr); err != nil {
			log.Errorf("failed to dial gRPC addr %s: %v", addr, err)
			continue
		} else {
			regLock := sync.Mutex{}
			registered := atomic.Bool{}
			// todo if read g.config.ConnectionNum, will cause the connect to fail
			for i := 1; i <= 1; i++ {
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
					continue
				}
				g.registerChannel(channel)
			}
			if err = g.registerTm(addr); err != nil {
				log.Errorf("%v", err)
				g.releaseChannelByAddr(addr)
			}
		}
	}
}

func (g *ChannelManager) getAvailServerList() []*discovery.ServiceInstance {
	registryService := discovery.GetRegistry()
	instances, err := registryService.Lookup(config.GetSeataConfig().TxServiceGroup)
	if err != nil {
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
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.WithStreamInterceptor(grpc2.ClientTransactionStreamInterceptor))
	grpc.InitialConnWindowSize(1 * 1024 * 1024)
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
	channel := loadbalance.Select(loadbalance.GetLoadBalanceConfig().Type, &g.allChannels, g.getXid(msg)).(*Channel)
	if channel != nil {
		return channel
	}

	if g.clientSize == 0 {
		ticker := time.NewTicker(time.Duration(checkAliveInternal) * time.Millisecond)
		defer ticker.Stop()
		for i := 0; i < maxCheckAliveRetry; i++ {
			<-ticker.C
			g.allChannels.Range(func(key, value interface{}) bool {
				channel = key.(*Channel)
				if channel.IsClosed() {
					g.releaseChannel(channel)
				} else {
					return false
				}
				return true
			})
			if channel != nil {
				return channel
			}
		}
	}
	return nil
}

func (g *ChannelManager) getXid(msg interface{}) string {
	var xid string
	if tmpMsg, ok := msg.(pb.AbstractGlobalEndRequestProto); ok {
		xid = tmpMsg.Xid
	} else if tmpMsg, ok := msg.(pb.GlobalBeginRequestProto); ok {
		xid = tmpMsg.TransactionName
	} else if tmpMsg, ok := msg.(pb.BranchRegisterRequestProto); ok {
		xid = tmpMsg.Xid
	} else if tmpMsg, ok := msg.(pb.BranchReportRequestProto); ok {
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

func (g *ChannelManager) releaseChannel(channel *Channel) {
	g.allChannels.Delete(channel)
	if !channel.IsClosed() {
		m, _ := g.serverChannels.LoadOrStore(channel.addr, &sync.Map{})
		sMap := m.(*sync.Map)
		sMap.Delete(channel)
		channel.close()
	}
	atomic.AddInt32(&g.clientSize, -1)
}

func (g *ChannelManager) registerChannel(channel *Channel) {
	g.allChannels.LoadOrStore(channel, true)
	m, _ := g.serverChannels.LoadOrStore(channel.addr, &sync.Map{})
	sMap := m.(*sync.Map)
	sMap.Store(channel, true)
	atomic.AddInt32(&g.clientSize, 1)
}

func (g *ChannelManager) releaseChannelByAddr(addr string) {
	g.allChannels.Range(func(key, value any) bool {
		ch := key.(*Channel)
		if ch.addr == addr {
			g.releaseChannel(ch)
		}
		return true
	})
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
