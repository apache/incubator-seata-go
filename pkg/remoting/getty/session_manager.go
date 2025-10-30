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

	"seata.apache.org/seata-go/pkg/discovery"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/remoting/loadbalance"
	"seata.apache.org/seata-go/pkg/util/log"
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
	serverSessions sync.Map
	allSessions    sync.Map
	sessionSize    int32
	gettyConf      *config.Config
}

func initSessionManager(gettyConfig *config.Config) {
	if sessionManager == nil {
		onceSessionManager.Do(func() {
			sessionManager = &SessionManager{
				allSessions:    sync.Map{},
				serverSessions: sync.Map{},
				gettyConf:      gettyConfig,
			}
			sessionManager.init()
		})
	}
}

func (g *SessionManager) init() {
	addressList := g.getAvailServerList()
	if len(addressList) == 0 {
		log.Warn("no have valid seata server list")
	}
	for _, address := range addressList {
		gettyClient := getty.NewTCPClient(
			getty.WithServerAddress(net.JoinHostPort(address.Addr, strconv.Itoa(address.Port))),
			// todo if read c.gettyConf.ConnectionNum, will cause the connect to fail
			getty.WithConnectionNumber(1),
			getty.WithReconnectInterval(g.gettyConf.ReconnectInterval),
			getty.WithClientTaskPool(gxsync.NewTaskPoolSimple(0)),
		)
		go gettyClient.RunEventLoop(g.newSession)
	}
}

func (g *SessionManager) getAvailServerList() []*discovery.ServiceInstance {
	registryService := discovery.GetRegistry()
	instances, err := registryService.Lookup(config.GetSeataConfig().TxServiceGroup)
	if err != nil {
		return nil
	}
	return instances
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
	session := loadbalance.Select(loadbalance.GetLoadBalanceConfig().Type, &g.allSessions, g.getXid(msg)).(getty.Session)
	if session != nil {
		return session
	}

	if g.sessionSize == 0 {
		ticker := time.NewTicker(time.Duration(checkAliveInternal) * time.Millisecond)
		defer ticker.Stop()
		for i := 0; i < maxCheckAliveRetry; i++ {
			<-ticker.C
			g.allSessions.Range(func(key, value interface{}) bool {
				session = key.(getty.Session)
				if session.IsClosed() {
					g.releaseSession(session)
				} else {
					return false
				}
				return true
			})
			if session != nil {
				return session
			}
		}
	}
	return nil
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
	g.allSessions.Delete(session)
	if !session.IsClosed() {
		m, _ := g.serverSessions.LoadOrStore(session.RemoteAddr(), &sync.Map{})
		sMap := m.(*sync.Map)
		sMap.Delete(session)
		session.Close()
	}
	atomic.AddInt32(&g.sessionSize, -1)
}

func (g *SessionManager) registerSession(session getty.Session) {
	g.allSessions.Store(session, true)
	m, _ := g.serverSessions.LoadOrStore(session.RemoteAddr(), &sync.Map{})
	sMap := m.(*sync.Map)
	sMap.Store(session, true)
	atomic.AddInt32(&g.sessionSize, 1)
}
