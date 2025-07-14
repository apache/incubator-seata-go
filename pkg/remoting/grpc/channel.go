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
	"sync"
	"sync/atomic"
	"time"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/pkg/util/log"

	"google.golang.org/grpc"
)

type Channel struct {
	addr   string
	conn   *grpc.ClientConn
	client pb.SeataServiceClient
	stream pb.SeataService_SendRequestClient

	sendCh  chan *pb.GrpcMessageProto
	closeCh chan struct{}

	wg sync.WaitGroup
	mu sync.Mutex

	regLock    *sync.Mutex
	registered *atomic.Bool
}

func (c *Channel) RemoteAddr() string {
	return c.addr
}

func (c *Channel) IsClosed() bool {
	select {
	case <-c.closeCh:
		return true
	default:
		return false
	}
}

func (c *Channel) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.IsClosed() {
		close(c.closeCh)
		if err := c.stream.CloseSend(); err != nil {
			log.Debugf("CloseSend error: %v", err)
		}
		c.wg.Wait()
	}
}

func (c *Channel) softReconnect() error {
	return c.reconnect(c.client)
}

func (c *Channel) hardReconnect() error {
	c.registered.Store(false)
	conn, err := channelManager.newConn(c.addr)
	if err != nil {
		return err
	}
	client := pb.NewSeataServiceClient(conn)
	err = c.reconnect(client)
	if err != nil {
		return err
	}
	c.conn = conn
	c.client = client

	return nil
}

func (c *Channel) reconnect(client pb.SeataServiceClient) error {
	ctx := context.Background()
	stream, err := client.SendRequest(ctx)
	if err != nil {
		return err
	}
	c.stream = stream

	c.closeCh = make(chan struct{})
	c.sendCh = make(chan *pb.GrpcMessageProto, defaultSendChBuffer)
	c.wg.Add(2)

	instance := GetGrpcClientHandlerInstance()

	go c.sendProcessor()
	go instance.StartReceiveLoop(ctx, c)

	// ensure register once
	if !c.registered.Load() {
		c.regLock.Lock()
		defer c.regLock.Unlock()
		if !c.registered.Load() {
			if err = channelManager.registerTm(c.addr); err != nil {
				return err
			}
			c.registered.Store(true)
		}
	}

	return nil
}

func (c *Channel) Send(request *pb.GrpcMessageProto) error {
	if c.IsClosed() {
		return fmt.Errorf("stream closed")
	}
	errCh := make(chan error, 1)
	msgSendTrackers.Store(request.Id, errCh)
	defer msgSendTrackers.Delete(request.Id)

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	select {
	case c.sendCh <- request:
		select {
		case err := <-errCh:
			return err
		case <-c.closeCh:
			return fmt.Errorf("stream closed")
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for send result")
		}
	case <-timeout.C:
		return fmt.Errorf("send Chan full")
	}

}

func (c *Channel) sendProcessor() {
	defer c.wg.Done()

	for {
		select {
		case <-c.closeCh:
			return
		case msg := <-c.sendCh:
			c.mu.Lock()
			if c.IsClosed() {
				c.mu.Unlock()
				return
			}
			err := c.stream.Send(msg)
			c.mu.Unlock()
			if errChan, ok := msgSendTrackers.Load(msg.Id); ok {
				select {
				case errChan.(chan error) <- err:
				default:
				}
			}
			if err != nil && msg.MessageType != int32(message.RequestTypeHeartbeatRequest) {
				go c.close()
				return
			}
		}
	}
}
