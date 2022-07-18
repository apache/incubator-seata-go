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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/dubbogo/gost/bytes"
	"github.com/dubbogo/gost/net"
	gxsync "github.com/dubbogo/gost/sync"
	gxtime "github.com/dubbogo/gost/time"

	"github.com/gorilla/websocket"

	perrors "github.com/pkg/errors"
)

const (
	reconnectInterval = 3e8 // 300ms
	connectInterval   = 5e8 // 500ms
	connectTimeout    = 3e9
	maxTimes          = 10
)

var (
	sessionClientKey   = "session-client-owner"
	connectPingPackage = []byte("connect-ping")

	clientID = EndPointID(0)
)

type Client interface {
	EndPoint
}

type client struct {
	ClientOptions

	// endpoint ID
	endPointID EndPointID

	// net
	sync.Mutex
	endPointType EndPointType

	newSession NewSessionCallback
	ssMap      map[Session]struct{}

	sync.Once
	done chan struct{}
	wg   sync.WaitGroup
}

func (c *client) init(opts ...ClientOption) {
	for _, opt := range opts {
		opt(&(c.ClientOptions))
	}
}

func newClient(t EndPointType, opts ...ClientOption) *client {
	c := &client{
		endPointID:   atomic.AddInt32(&clientID, 1),
		endPointType: t,
		done:         make(chan struct{}),
	}

	c.init(opts...)

	if c.number <= 0 || c.addr == "" {
		panic(fmt.Sprintf("client type:%s, @connNum:%d, @serverAddr:%s", t, c.number, c.addr))
	}

	c.ssMap = make(map[Session]struct{}, c.number)

	return c
}

// NewTCPClient builds a tcp client.
func NewTCPClient(opts ...ClientOption) Client {
	return newClient(TCP_CLIENT, opts...)
}

// NewUDPClient builds a connected udp client
func NewUDPClient(opts ...ClientOption) Client {
	return newClient(UDP_CLIENT, opts...)
}

// NewWSClient builds a ws client.
func NewWSClient(opts ...ClientOption) Client {
	c := newClient(WS_CLIENT, opts...)

	if !strings.HasPrefix(c.addr, "ws://") {
		panic(fmt.Sprintf("the prefix @serverAddr:%s is not ws://", c.addr))
	}

	return c
}

// NewWSSClient function builds a wss client.
func NewWSSClient(opts ...ClientOption) Client {
	c := newClient(WSS_CLIENT, opts...)

	if c.cert == "" {
		panic(fmt.Sprintf("@cert:%s", c.cert))
	}
	if !strings.HasPrefix(c.addr, "wss://") {
		panic(fmt.Sprintf("the prefix @serverAddr:%s is not wss://", c.addr))
	}

	return c
}

func (c *client) ID() EndPointID {
	return c.endPointID
}

func (c *client) EndPointType() EndPointType {
	return c.endPointType
}

func (c *client) dialTCP() Session {
	var (
		err  error
		conn net.Conn
	)

	for {
		if c.IsClosed() {
			return nil
		}
		if c.sslEnabled {
			if sslConfig, buildTlsConfErr := c.tlsConfigBuilder.BuildTlsConfig(); buildTlsConfErr == nil && sslConfig != nil {
				d := &net.Dialer{Timeout: connectTimeout}
				conn, err = tls.DialWithDialer(d, "tcp", c.addr, sslConfig)
			}
		} else {
			conn, err = net.DialTimeout("tcp", c.addr, connectTimeout)
		}
		if err == nil && gxnet.IsSameAddr(conn.RemoteAddr(), conn.LocalAddr()) {
			conn.Close()
			err = errSelfConnect
		}
		if err == nil {
			return newTCPSession(conn, c)
		}

		log.Infof("net.DialTimeout(addr:%s, timeout:%v) = error:%+v", c.addr, connectTimeout, perrors.WithStack(err))
		<-gxtime.After(connectInterval)
	}
}

func (c *client) dialUDP() Session {
	var (
		err       error
		conn      *net.UDPConn
		localAddr *net.UDPAddr
		peerAddr  *net.UDPAddr
		length    int
		bufp      *[]byte
		buf       []byte
	)

	bufp = gxbytes.GetBytes(128)
	defer gxbytes.PutBytes(bufp)
	buf = *bufp
	localAddr = &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	peerAddr, _ = net.ResolveUDPAddr("udp", c.addr)
	for {
		if c.IsClosed() {
			return nil
		}
		conn, err = net.DialUDP("udp", localAddr, peerAddr)
		if err == nil && gxnet.IsSameAddr(conn.RemoteAddr(), conn.LocalAddr()) {
			conn.Close()
			err = errSelfConnect
		}
		if err != nil {
			log.Warnf("net.DialTimeout(addr:%s, timeout:%v) = error:%+v", c.addr, perrors.WithStack(err))
			<-gxtime.After(connectInterval)
			continue
		}

		// check connection alive by write/read action
		conn.SetWriteDeadline(time.Now().Add(1e9))
		if length, err = conn.Write(connectPingPackage[:]); err != nil {
			conn.Close()
			log.Warnf("conn.Write(%s) = {length:%d, err:%+v}", string(connectPingPackage), length, perrors.WithStack(err))
			<-gxtime.After(connectInterval)
			continue
		}
		conn.SetReadDeadline(time.Now().Add(1e9))
		length, err = conn.Read(buf)
		if netErr, ok := perrors.Cause(err).(net.Error); ok && netErr.Timeout() {
			err = nil
		}
		if err != nil {
			log.Infof("conn{%#v}.Read() = {length:%d, err:%+v}", conn, length, perrors.WithStack(err))
			conn.Close()
			<-gxtime.After(connectInterval)
			continue
		}
		return newUDPSession(conn, c)
	}
}

func (c *client) dialWS() Session {
	var (
		err    error
		dialer websocket.Dialer
		conn   *websocket.Conn
		ss     Session
	)

	dialer.EnableCompression = true
	for {
		if c.IsClosed() {
			return nil
		}
		conn, _, err = dialer.Dial(c.addr, nil)
		log.Infof("websocket.dialer.Dial(addr:%s) = error:%+v", c.addr, perrors.WithStack(err))
		if err == nil && gxnet.IsSameAddr(conn.RemoteAddr(), conn.LocalAddr()) {
			conn.Close()
			err = errSelfConnect
		}
		if err == nil {
			ss = newWSSession(conn, c)
			if ss.(*session).maxMsgLen > 0 {
				conn.SetReadLimit(int64(ss.(*session).maxMsgLen))
			}

			return ss
		}

		log.Infof("websocket.dialer.Dial(addr:%s) = error:%+v", c.addr, perrors.WithStack(err))
		<-gxtime.After(connectInterval)
	}
}

func (c *client) dialWSS() Session {
	var (
		err      error
		root     *x509.Certificate
		roots    []*x509.Certificate
		certPool *x509.CertPool
		config   *tls.Config
		dialer   websocket.Dialer
		conn     *websocket.Conn
		ss       Session
	)

	dialer.EnableCompression = true

	config = &tls.Config{
		InsecureSkipVerify: true,
	}

	if c.cert != "" {
		certPEMBlock, err := ioutil.ReadFile(c.cert)
		if err != nil {
			panic(fmt.Sprintf("ioutil.ReadFile(cert:%s) = error:%+v", c.cert, perrors.WithStack(err)))
		}

		var cert tls.Certificate
		for {
			var certDERBlock *pem.Block
			certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
			if certDERBlock == nil {
				break
			}
			if certDERBlock.Type == "CERTIFICATE" {
				cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
			}
		}
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0] = cert
	}

	certPool = x509.NewCertPool()
	for _, c := range config.Certificates {
		roots, err = x509.ParseCertificates(c.Certificate[len(c.Certificate)-1])
		if err != nil {
			panic(fmt.Sprintf("error parsing server's root cert: %+v\n", perrors.WithStack(err)))
		}
		for _, root = range roots {
			certPool.AddCert(root)
		}
	}
	config.InsecureSkipVerify = true
	config.RootCAs = certPool

	// dialer.EnableCompression = true
	dialer.TLSClientConfig = config
	for {
		if c.IsClosed() {
			return nil
		}
		conn, _, err = dialer.Dial(c.addr, nil)
		if err == nil && gxnet.IsSameAddr(conn.RemoteAddr(), conn.LocalAddr()) {
			conn.Close()
			err = errSelfConnect
		}
		if err == nil {
			ss = newWSSession(conn, c)
			if ss.(*session).maxMsgLen > 0 {
				conn.SetReadLimit(int64(ss.(*session).maxMsgLen))
			}
			ss.SetName(defaultWSSSessionName)

			return ss
		}

		log.Infof("websocket.dialer.Dial(addr:%s) = error:%+v", c.addr, perrors.WithStack(err))
		<-gxtime.After(connectInterval)
	}
}

func (c *client) dial() Session {
	switch c.endPointType {
	case TCP_CLIENT:
		return c.dialTCP()
	case UDP_CLIENT:
		return c.dialUDP()
	case WS_CLIENT:
		return c.dialWS()
	case WSS_CLIENT:
		return c.dialWSS()
	}

	return nil
}

func (c *client) GetTaskPool() gxsync.GenericTaskPool {
	return c.tPool
}

func (c *client) sessionNum() int {
	var num int

	c.Lock()
	for s := range c.ssMap {
		if s.IsClosed() {
			delete(c.ssMap, s)
		}
	}
	num = len(c.ssMap)
	c.Unlock()

	return num
}

func (c *client) connect() {
	var (
		err error
		ss  Session
	)

	for {
		ss = c.dial()
		if ss == nil {
			// client has been closed
			break
		}
		err = c.newSession(ss)
		if err == nil {
			ss.(*session).run()
			c.Lock()
			if c.ssMap == nil {
				c.Unlock()
				break
			}
			c.ssMap[ss] = struct{}{}
			c.Unlock()
			ss.SetAttribute(sessionClientKey, c)
			break
		}
		// don't distinguish between tcp connection and websocket connection. Because
		// gorilla/websocket/conn.go:(Conn)Close also invoke net.Conn.Close()
		ss.Conn().Close()
	}
}

// there are two methods to keep connection pool. the first approach is like
// redigo's lazy connection pool(https://github.com/gomodule/redigo/blob/master/redis/pool.go:),
// in which you should apply testOnBorrow to check alive of the connection.
// the second way is as follows. @RunEventLoop detects the aliveness of the connection
// in regular time interval.
// the active method maybe overburden the cpu slightly.
// however, you can get a active tcp connection very quickly.
func (c *client) RunEventLoop(newSession NewSessionCallback) {
	c.Lock()
	c.newSession = newSession
	c.Unlock()
	c.reConnect()
}

// a for-loop connect to make sure the connection pool is valid
func (c *client) reConnect() {
	var num, max, times, interval int

	max = c.number
	interval = c.reconnectInterval
	if interval == 0 {
		interval = reconnectInterval
	}
	for {
		if c.IsClosed() {
			log.Warnf("client{peer:%s} goroutine exit now.", c.addr)
			break
		}

		num = c.sessionNum()
		if max <= num {
			break
		}
		c.connect()
		times++
		if maxTimes < times {
			times = maxTimes
		}
		<-gxtime.After(time.Duration(int64(times) * int64(interval)))
	}
}

func (c *client) stop() {
	select {
	case <-c.done:
		return
	default:
		c.Once.Do(func() {
			close(c.done)
			c.Lock()
			for s := range c.ssMap {
				s.RemoveAttribute(sessionClientKey)
				s.Close()
			}
			c.ssMap = nil

			c.Unlock()
		})
	}
}

func (c *client) IsClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

func (c *client) Close() {
	c.stop()
	c.wg.Wait()
}
