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
	"sync"

	getty "github.com/apache/dubbo-getty"
	gxsync "github.com/dubbogo/gost/sync"

	"seata.apache.org/seata-go/pkg/discovery"
	"seata.apache.org/seata-go/pkg/protocol/codec"
	"seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/util/log"
)

type RpcClient struct {
	gettyConf    *config.Config
	seataConf    *config.SeataConfig
	gettyClients []getty.Client
	futures      *sync.Map
}

func InitRpcClient(gettyConfig *config.Config, seataConfig *config.SeataConfig) {
	config.IniConfig(seataConfig)
	rpcClient := &RpcClient{
		gettyConf:    gettyConfig,
		seataConf:    seataConfig,
		gettyClients: make([]getty.Client, 0),
	}
	codec.Init()
	rpcClient.init()
}

func (c *RpcClient) init() {
	addressList := c.getAvailServerList()
	if len(addressList) == 0 {
		log.Warn("no have valid seata server list")
	}
	for _, address := range addressList {
		gettyClient := getty.NewTCPClient(
			getty.WithServerAddress(fmt.Sprintf("%s:%d", address.Addr, address.Port)),
			// todo if read c.gettyConf.ConnectionNum, will cause the connect to fail
			getty.WithConnectionNumber(1),
			getty.WithReconnectInterval(c.gettyConf.ReconnectInterval),
			getty.WithClientTaskPool(gxsync.NewTaskPoolSimple(0)),
		)
		go gettyClient.RunEventLoop(c.newSession)
		// c.gettyClients = append(c.gettyClients, gettyClient)
	}
}

func (c *RpcClient) getAvailServerList() []*discovery.ServiceInstance {
	registryService := discovery.GetRegistry()
	instances, err := registryService.Lookup(c.seataConf.TxServiceGroup)
	if err != nil {
		return nil
	}
	return instances
}

func (c *RpcClient) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
		err     error
	)

	if c.gettyConf.SessionConfig.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}
	if _, ok = session.Conn().(*tls.Conn); ok {
		c.setSessionConfig(session)
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

		if err = tcpConn.SetNoDelay(c.gettyConf.SessionConfig.TCPNoDelay); err != nil {
			return err
		}
		if err = tcpConn.SetKeepAlive(c.gettyConf.SessionConfig.TCPKeepAlive); err != nil {
			return err
		}
		if c.gettyConf.SessionConfig.TCPKeepAlive {
			if err = tcpConn.SetKeepAlivePeriod(c.gettyConf.SessionConfig.KeepAlivePeriod); err != nil {
				return err
			}
		}
		if err = tcpConn.SetReadBuffer(c.gettyConf.SessionConfig.TCPRBufSize); err != nil {
			return err
		}
		if err = tcpConn.SetWriteBuffer(c.gettyConf.SessionConfig.TCPWBufSize); err != nil {
			return err
		}
	}

	c.setSessionConfig(session)
	log.Debugf("rpc_client new session:%s\n", session.Stat())

	return nil
}

func (c *RpcClient) setSessionConfig(session getty.Session) {
	session.SetName(c.gettyConf.SessionConfig.SessionName)
	session.SetMaxMsgLen(c.gettyConf.SessionConfig.MaxMsgLen)
	session.SetPkgHandler(rpcPkgHandler)
	session.SetEventListener(GetGettyClientHandlerInstance())
	session.SetReadTimeout(c.gettyConf.SessionConfig.TCPReadTimeout)
	session.SetWriteTimeout(c.gettyConf.SessionConfig.TCPWriteTimeout)
	session.SetCronPeriod((int)(c.gettyConf.SessionConfig.CronPeriod.Milliseconds()))
	session.SetWaitTime(c.gettyConf.SessionConfig.WaitTimeout)
}
