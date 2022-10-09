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
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/config"
	"github.com/seata/seata-go/pkg/util/log"
)

type RpcClient struct {
	conf         *config.ClientConfig
	gettyClients []getty.Client
	futures      *sync.Map
}

func InitRpcClient() {
	rpcClient := &RpcClient{
		conf:         config.GetClientConfig(),
		gettyClients: make([]getty.Client, 0),
	}
	rpcClient.init()
}

func (c *RpcClient) init() {
	addressList := getAvailServerList()
	if len(addressList) == 0 {
		log.Warn("no have valid seata server list")
	}
	for _, address := range addressList {
		gettyClient := getty.NewTCPClient(
			getty.WithServerAddress(address),
			getty.WithConnectionNumber(c.conf.GettyConfig.ConnectionNum),
			getty.WithReconnectInterval(c.conf.GettyConfig.ReconnectInterval),
			getty.WithClientTaskPool(gxsync.NewTaskPoolSimple(0)),
		)
		go gettyClient.RunEventLoop(c.newSession)
		// c.gettyClients = append(c.gettyClients, gettyClient)
	}
}

// todo mock
func getAvailServerList() []string {
	return []string{"127.0.0.1:8091"}
}

func (c *RpcClient) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
		err     error
	)

	if c.conf.GettyConfig.GettySessionParam.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}
	if _, ok = session.Conn().(*tls.Conn); ok {
		session.SetName(c.conf.GettyConfig.GettySessionParam.SessionName)
		session.SetMaxMsgLen(c.conf.GettyConfig.GettySessionParam.MaxMsgLen)
		session.SetPkgHandler(rpcPkgHandler)
		session.SetEventListener(GetGettyClientHandlerInstance())
		session.SetReadTimeout(c.conf.GettyConfig.GettySessionParam.TCPReadTimeout)
		session.SetWriteTimeout(c.conf.GettyConfig.GettySessionParam.TCPWriteTimeout)
		session.SetCronPeriod((int)(c.conf.GettyConfig.GettySessionParam.CronPeriod))
		session.SetWaitTime(c.conf.GettyConfig.GettySessionParam.WaitTimeout)
		log.Debugf("server accepts new tls session:%s\n", session.Stat())
		return nil
	}
	if _, ok = session.Conn().(*net.TCPConn); !ok {
		panic(fmt.Sprintf("%s, session.conn{%#v} is not a tcp connection\n", session.Stat(), session.Conn()))
	}

	if _, ok = session.Conn().(*tls.Conn); !ok {
		if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
			return errors.New(fmt.Sprintf("%s, session.conn{%#v} is not tcp connection", session.Stat(), session.Conn()))
		}

		if err = tcpConn.SetNoDelay(c.conf.GettyConfig.GettySessionParam.TCPNoDelay); err != nil {
			return err
		}
		if err = tcpConn.SetKeepAlive(c.conf.GettyConfig.GettySessionParam.TCPKeepAlive); err != nil {
			return err
		}
		if c.conf.GettyConfig.GettySessionParam.TCPKeepAlive {
			if err = tcpConn.SetKeepAlivePeriod(c.conf.GettyConfig.GettySessionParam.KeepAlivePeriod); err != nil {
				return err
			}
		}
		if err = tcpConn.SetReadBuffer(c.conf.GettyConfig.GettySessionParam.TCPRBufSize); err != nil {
			return err
		}
		if err = tcpConn.SetWriteBuffer(c.conf.GettyConfig.GettySessionParam.TCPWBufSize); err != nil {
			return err
		}
	}

	session.SetName(c.conf.GettyConfig.GettySessionParam.SessionName)
	session.SetMaxMsgLen(c.conf.GettyConfig.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(rpcPkgHandler)
	session.SetEventListener(GetGettyClientHandlerInstance())
	session.SetReadTimeout(c.conf.GettyConfig.GettySessionParam.TCPReadTimeout)
	session.SetWriteTimeout(c.conf.GettyConfig.GettySessionParam.TCPWriteTimeout)
	session.SetCronPeriod((int)(c.conf.GettyConfig.GettySessionParam.CronPeriod.Nanoseconds() / 1e6))
	session.SetWaitTime(c.conf.GettyConfig.GettySessionParam.WaitTimeout)
	log.Debugf("rpc_client new session:%s\n", session.Stat())

	return nil
}
