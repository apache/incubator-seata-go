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
	"sync"
	"time"

	"seata.apache.org/seata-go/pkg/protocol/codec"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/pkg/remoting/processor"
	"seata.apache.org/seata-go/pkg/util/log"

	"go.uber.org/atomic"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/protobuf/proto"
)

var (
	clientHandler     *grpcClientHandler
	onceClientHandler = &sync.Once{}
)

type grpcClientHandler struct {
	idGenerator  *atomic.Uint32
	processorMap map[message.MessageType]processor.RemotingProcessor
	typeMap      map[string]message.MessageType
}

func GetGrpcClientHandlerInstance() *grpcClientHandler {
	if clientHandler == nil {
		onceClientHandler.Do(func() {
			clientHandler = &grpcClientHandler{
				idGenerator:  &atomic.Uint32{},
				processorMap: make(map[message.MessageType]processor.RemotingProcessor, 0),
				typeMap:      make(map[string]message.MessageType, 0),
			}
		})
	}
	return clientHandler
}

func (g *grpcClientHandler) monitorStreamHealth(ctx context.Context, channel *Channel, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	heartBeatRetryTimes := 0

	for {
		select {
		case <-ticker.C:
			err := g.transferHeartBeat(channel, &pb.HeartbeatMessageProto{Ping: true})
			if err != nil {
				heartBeatRetryTimes++
				log.Warnf("failed to send heart beat: {%#v}", err.Error())
				if heartBeatRetryTimes >= maxHeartBeatRetryTimes {
					log.Warnf("heartbeat retry times exceed default max retry times{%d}", maxHeartBeatRetryTimes)
					channel.close()
					if channelManager.getAllChannelIsClosedByAddr(channel.addr) {
						channel.registered.Store(false)
					}

					flag := false
					var reconnectErr error
					for !flag {
						state := channel.conn.GetState()
						switch state {
						case connectivity.Shutdown:
							reconnectErr = channel.hardReconnect()
							if reconnectErr != nil {
								log.Errorf("reconnect error cause {%v}", reconnectErr)
								channelManager.releaseChannelByAddr(channel.addr)
							}
						case connectivity.Ready:
							reconnectErr = channel.softReconnect()
							if reconnectErr != nil {
								log.Errorf("reconnect error cause {%v}", reconnectErr)
							}
						default:
							channel.conn.Connect()
							channel.conn.WaitForStateChange(ctx, state)
							continue
						}
						channelManager.registerChannel(channel)

						flag = true

					}
				}
			} else {
				heartBeatRetryTimes = 0
			}
		}
	}
}

func (g *grpcClientHandler) StartReceiveLoop(ctx context.Context, c *Channel) {
	defer c.wg.Done()

	for {
		select {
		case <-c.closeCh:
			log.Errorf("stream closed")
			return
		default:
			msg, err := c.stream.Recv()
			if err != nil {
				go c.close()
				return
			}

			go func() {
				rpcMessage, err := Decode(msg)
				if err != nil {
					log.Errorf("Decode failed cause %v", err)
					return
				}

				body := rpcMessage.Body.(proto.Message)
				if t, ok := g.typeMap[string(proto.MessageName(body).Name())]; ok {
					processor := g.processorMap[t]
					if processor != nil {
						processor.Process(ctx, rpcMessage)
					} else {
						log.Errorf("This format type %v has no processor.", t)
					}
				} else {
					log.Errorf("This rpcMessage body %#v is not in typeMap.", body)
				}
			}()
		}
	}
}

func (g *grpcClientHandler) transferHeartBeat(channel *Channel, msg *pb.HeartbeatMessageProto) error {
	rpcMessage := message.RpcMessage{
		ID:         int32(g.idGenerator.Inc()),
		Type:       message.RequestTypeHeartbeatRequest,
		Codec:      byte(codec.CodecTypeGRPC),
		Compressor: 0,
		HeadMap:    make(map[string]string),
		Body:       msg,
	}
	return GetGrpcRemotingClient().grpcRemoting.SendAsync(rpcMessage, channel, nil)
}

func (g *grpcClientHandler) RegisterProcessor(msgType message.MessageType, processor processor.RemotingProcessor) {
	if nil != processor {
		g.processorMap[msgType] = processor
	}
}

func (g *grpcClientHandler) RegisterType(messageProto string, msgType message.MessageType) {
	if messageProto != "" {
		g.typeMap[messageProto] = msgType
	}
}
