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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"time"
)

import (
	gxbytes "github.com/dubbogo/gost/bytes"
	gxcontext "github.com/dubbogo/gost/context"
	gxtime "github.com/dubbogo/gost/time"

	"github.com/gorilla/websocket"

	perrors "github.com/pkg/errors"

	uatomic "go.uber.org/atomic"
)

const (
	maxReadBufLen   = 4 * 1024
	netIOTimeout    = 1e9      // 1s
	period          = 60 * 1e9 // 1 minute
	pendingDuration = 3e9
	// MaxWheelTimeSpan 900s, 15 minute
	MaxWheelTimeSpan = 900e9
	maxPacketLen     = 16 * 1024

	defaultSessionName    = "session"
	defaultTCPSessionName = "tcp-session"
	defaultUDPSessionName = "udp-session"
	defaultWSSessionName  = "ws-session"
	defaultWSSSessionName = "wss-session"
	outputFormat          = "session %s, Read Bytes: %d, Write Bytes: %d, Read Pkgs: %d, Write Pkgs: %d"
)

var defaultTimerWheel *gxtime.TimerWheel

func init() {
	gxtime.InitDefaultTimerWheel()
	defaultTimerWheel = gxtime.GetDefaultTimerWheel()
}

// Session wrap connection between the server and the client
type Session interface {
	Connection
	Reset()
	Conn() net.Conn
	Stat() string
	IsClosed() bool
	// EndPoint get endpoint type
	EndPoint() EndPoint
	SetMaxMsgLen(int)
	SetName(string)
	SetEventListener(EventListener)
	SetPkgHandler(ReadWriter)
	SetReader(Reader)
	SetWriter(Writer)
	SetCronPeriod(int)
	SetWaitTime(time.Duration)
	GetAttribute(interface{}) interface{}
	SetAttribute(interface{}, interface{})
	RemoveAttribute(interface{})

	// WritePkg the Writer will invoke this function. Pls attention that if timeout is less than 0, WritePkg will send @pkg asap.
	// for udp session, the first parameter should be UDPContext.
	// totalBytesLength: @pkg stream bytes length after encoding @pkg.
	// sendBytesLength: stream bytes length that sent out successfully.
	// err: maybe it has illegal data, encoding error, or write out system error.
	WritePkg(pkg interface{}, timeout time.Duration) (totalBytesLength int, sendBytesLength int, err error)
	WriteBytes([]byte) (int, error)
	WriteBytesArray(...[]byte) (int, error)
	Close()
}

// getty base session
type session struct {
	name     string
	endPoint EndPoint

	// net read Write
	Connection

	listener EventListener

	// codec
	reader Reader // @reader should be nil when @conn is a gettyWSConn object.
	writer Writer

	// handle logic
	maxMsgLen int32

	// heartbeat
	period time.Duration

	// done
	wait time.Duration
	once *sync.Once
	done chan struct{}

	// attribute
	attrs *gxcontext.ValuesContext

	// goroutines sync
	grNum      uatomic.Int32
	lock       sync.RWMutex
	packetLock sync.RWMutex
}

func newSession(endPoint EndPoint, conn Connection) *session {
	ss := &session{
		name:     defaultSessionName,
		endPoint: endPoint,

		Connection: conn,

		maxMsgLen: maxReadBufLen,

		period: period,

		once:  &sync.Once{},
		done:  make(chan struct{}),
		wait:  pendingDuration,
		attrs: gxcontext.NewValuesContext(context.Background()),
	}

	ss.Connection.setSession(ss)
	ss.SetWriteTimeout(netIOTimeout)
	ss.SetReadTimeout(netIOTimeout)

	return ss
}

func newTCPSession(conn net.Conn, endPoint EndPoint) Session {
	c := newGettyTCPConn(conn)
	session := newSession(endPoint, c)
	session.name = defaultTCPSessionName

	return session
}

func newUDPSession(conn *net.UDPConn, endPoint EndPoint) Session {
	c := newGettyUDPConn(conn)
	session := newSession(endPoint, c)
	session.name = defaultUDPSessionName

	return session
}

func newWSSession(conn *websocket.Conn, endPoint EndPoint) Session {
	c := newGettyWSConn(conn)
	session := newSession(endPoint, c)
	session.name = defaultWSSessionName

	return session
}

func (s *session) Reset() {
	*s = session{
		name:   defaultSessionName,
		once:   &sync.Once{},
		done:   make(chan struct{}),
		period: period,
		wait:   pendingDuration,
		attrs:  gxcontext.NewValuesContext(context.Background()),
	}
}

func (s *session) Conn() net.Conn {
	if tc, ok := s.Connection.(*gettyTCPConn); ok {
		return tc.conn
	}

	if uc, ok := s.Connection.(*gettyUDPConn); ok {
		return uc.conn
	}

	if wc, ok := s.Connection.(*gettyWSConn); ok {
		return wc.conn.UnderlyingConn()
	}

	return nil
}

func (s *session) EndPoint() EndPoint {
	return s.endPoint
}

func (s *session) gettyConn() *gettyConn {
	if tc, ok := s.Connection.(*gettyTCPConn); ok {
		return &(tc.gettyConn)
	}

	if uc, ok := s.Connection.(*gettyUDPConn); ok {
		return &(uc.gettyConn)
	}

	if wc, ok := s.Connection.(*gettyWSConn); ok {
		return &(wc.gettyConn)
	}

	return nil
}

// Stat get the connect statistic data
func (s *session) Stat() string {
	var conn *gettyConn
	if conn = s.gettyConn(); conn == nil {
		return ""
	}
	return fmt.Sprintf(
		outputFormat,
		s.sessionToken(),
		conn.readBytes.Load(),
		conn.writeBytes.Load(),
		conn.readPkgNum.Load(),
		conn.writePkgNum.Load(),
	)
}

// IsClosed check whether the session has been closed.
func (s *session) IsClosed() bool {
	select {
	case <-s.done:
		return true

	default:
		return false
	}
}

// SetMaxMsgLen set maximum package length of every package in (EventListener)OnMessage(@pkgs)
func (s *session) SetMaxMsgLen(length int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.maxMsgLen = int32(length)
}

// SetName set session name
func (s *session) SetName(name string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.name = name
}

// SetEventListener set event listener
func (s *session) SetEventListener(listener EventListener) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.listener = listener
}

// SetPkgHandler set package handler
func (s *session) SetPkgHandler(handler ReadWriter) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.reader = handler
	s.writer = handler
}

func (s *session) SetReader(reader Reader) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.reader = reader
}

func (s *session) SetWriter(writer Writer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.writer = writer
}

// SetCronPeriod period is in millisecond. Websocket session will send ping frame automatically every peroid.
func (s *session) SetCronPeriod(period int) {
	if period < 1 {
		panic("@period < 1")
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.period = time.Duration(period) * time.Millisecond
}

// SetWaitTime set maximum wait time when session got error or got exit signal
func (s *session) SetWaitTime(waitTime time.Duration) {
	if waitTime < 1 {
		panic("@wait < 1")
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.wait = waitTime
}

// GetAttribute get attribute of key @session:key
func (s *session) GetAttribute(key interface{}) interface{} {
	s.lock.RLock()
	if s.attrs == nil {
		s.lock.RUnlock()
		return nil
	}
	ret, flag := s.attrs.Get(key)
	s.lock.RUnlock()

	if !flag {
		return nil
	}

	return ret
}

// SetAttribute set attribute of key @session:key
func (s *session) SetAttribute(key interface{}, value interface{}) {
	s.lock.Lock()
	if s.attrs != nil {
		s.attrs.Set(key, value)
	}
	s.lock.Unlock()
}

// RemoveAttribute remove attribute of key @session:key
func (s *session) RemoveAttribute(key interface{}) {
	s.lock.Lock()
	if s.attrs != nil {
		s.attrs.Delete(key)
	}
	s.lock.Unlock()
}

func (s *session) sessionToken() string {
	if s.IsClosed() || s.Connection == nil {
		return "session-closed"
	}

	return fmt.Sprintf("{%s:%s:%d:%s<->%s}",
		s.name, s.EndPoint().EndPointType(), s.ID(), s.LocalAddr(), s.RemoteAddr())
}

func (s *session) WritePkg(pkg interface{}, timeout time.Duration) (int, int, error) {
	if pkg == nil {
		return 0, 0, fmt.Errorf("@pkg is nil")
	}
	if s.IsClosed() {
		return 0, 0, ErrSessionClosed
	}

	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			rBuf := make([]byte, size)
			rBuf = rBuf[:runtime.Stack(rBuf, false)]
			log.Errorf("[session.WritePkg] panic session %s: err=%s\n%s", s.sessionToken(), r, rBuf)
		}
	}()

	pkgBytes, err := s.writer.Write(s, pkg)
	if err != nil {
		log.Warnf("%s, [session.WritePkg] session.writer.Write(@pkg:%#v) = error:%+v", s.Stat(), pkg, err)
		return len(pkgBytes), 0, perrors.WithStack(err)
	}
	var udpCtxPtr *UDPContext
	if udpCtx, ok := pkg.(UDPContext); ok {
		udpCtxPtr = &udpCtx
	} else if udpCtxP, ok := pkg.(*UDPContext); ok {
		udpCtxPtr = udpCtxP
	}
	if udpCtxPtr != nil {
		udpCtxPtr.Pkg = pkgBytes
		pkg = *udpCtxPtr
	} else {
		pkg = pkgBytes
	}
	s.packetLock.RLock()
	defer s.packetLock.RUnlock()
	if 0 < timeout {
		s.Connection.SetWriteTimeout(timeout)
	}
	var succssCount int
	succssCount, err = s.Connection.send(pkg)
	if err != nil {
		log.Warnf("%s, [session.WritePkg] @s.Connection.Write(pkg:%#v) = err:%+v", s.Stat(), pkg, err)
		return len(pkgBytes), succssCount, perrors.WithStack(err)
	}
	return len(pkgBytes), succssCount, nil
}

// WriteBytes for codecs
func (s *session) WriteBytes(pkg []byte) (int, error) {
	if s.IsClosed() {
		return 0, ErrSessionClosed
	}

	leftPackageSize, totalSize, writeSize := len(pkg), len(pkg), 0
	if leftPackageSize > maxPacketLen {
		s.packetLock.Lock()
		defer s.packetLock.Unlock()
	} else {
		s.packetLock.RLock()
		defer s.packetLock.RUnlock()
	}

	for leftPackageSize > maxPacketLen {
		_, err := s.Connection.send(pkg[writeSize:(writeSize + maxPacketLen)])
		if err != nil {
			return writeSize, perrors.Wrapf(err, "s.Connection.Write(pkg len:%d)", len(pkg))
		}
		leftPackageSize -= maxPacketLen
		writeSize += maxPacketLen
	}

	if leftPackageSize == 0 {
		return writeSize, nil
	}

	_, err := s.Connection.send(pkg[writeSize:])
	if err != nil {
		return writeSize, perrors.Wrapf(err, "s.Connection.Write(pkg len:%d)", len(pkg))
	}

	return totalSize, nil
}

// WriteBytesArray Write multiple packages at once. so we invoke write sys.call just one time.
func (s *session) WriteBytesArray(pkgs ...[]byte) (int, error) {
	if s.IsClosed() {
		return 0, ErrSessionClosed
	}
	if len(pkgs) == 1 {
		return s.WriteBytes(pkgs[0])
	}

	// reduce syscall and memcopy for multiple packages
	if _, ok := s.Connection.(*gettyTCPConn); ok {
		s.packetLock.RLock()
		defer s.packetLock.RUnlock()
		lg, err := s.Connection.send(pkgs)
		if err != nil {
			return 0, perrors.Wrapf(err, "s.Connection.Write(pkgs num:%d)", len(pkgs))
		}
		return lg, nil
	}

	// get len
	var (
		l      int
		wlg    int
		err    error
		length int
		arrp   *[]byte
		arr    []byte
	)
	length = 0
	for i := 0; i < len(pkgs); i++ {
		length += len(pkgs[i])
	}

	// merge the pkgs
	arrp = gxbytes.AcquireBytes(length)
	defer gxbytes.ReleaseBytes(arrp)
	arr = *arrp

	l = 0
	for i := 0; i < len(pkgs); i++ {
		copy(arr[l:], pkgs[i])
		l += len(pkgs[i])
	}

	wlg, err = s.WriteBytes(arr)
	if err != nil {
		return 0, perrors.WithStack(err)
	}

	num := len(pkgs) - 1
	for i := 0; i < num; i++ {
		s.incWritePkgNum()
	}

	return wlg, nil
}

func heartbeat(_ gxtime.TimerID, _ time.Time, arg interface{}) error {
	ss, _ := arg.(*session)
	if ss == nil || ss.IsClosed() {
		return ErrSessionClosed
	}

	f := func() {
		wsConn, wsFlag := ss.Connection.(*gettyWSConn)
		if wsFlag {
			err := wsConn.writePing()
			if err != nil {
				log.Warnf("wsConn.writePing() = error:%+v", perrors.WithStack(err))
			}
		}

		ss.listener.OnCron(ss)
	}

	// if enable task pool, run @f asynchronously.
	if taskPool := ss.EndPoint().GetTaskPool(); taskPool != nil {
		taskPool.AddTaskAlways(f)
		return nil
	}
	f()
	return nil
}

// func (s *session) RunEventLoop() {
func (s *session) run() {
	if s.Connection == nil || s.listener == nil || s.writer == nil {
		errStr := fmt.Sprintf("session{name:%s, conn:%#v, listener:%#v, writer:%#v}",
			s.name, s.Connection, s.listener, s.writer)
		log.Error(errStr)
		panic(errStr)
	}

	// call session opened
	s.UpdateActive()
	if err := s.listener.OnOpen(s); err != nil {
		log.Errorf("[OnOpen] session %s, error: %#v", s.Stat(), err)
		s.Close()
		return
	}

	if _, err := defaultTimerWheel.AddTimer(heartbeat, gxtime.TimerLoop, s.period, s); err != nil {
		panic(fmt.Sprintf("failed to add session %s to defaultTimerWheel err:%v", s.Stat(), err))
	}

	s.grNum.Add(1)
	// start read gr
	go s.handlePackage()
}

func (s *session) addTask(pkg interface{}) {
	f := func() {
		s.listener.OnMessage(s, pkg)
		s.incReadPkgNum()
	}
	if taskPool := s.EndPoint().GetTaskPool(); taskPool != nil {
		taskPool.AddTaskAlways(f)
		return
	}
	f()
}

func (s *session) handlePackage() {
	var err error

	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			rBuf := make([]byte, size)
			rBuf = rBuf[:runtime.Stack(rBuf, false)]
			log.Errorf("[session.handlePackage] panic session %s: err=%s\n%s", s.sessionToken(), r, rBuf)
		}
		grNum := s.grNum.Add(-1)
		log.Infof("%s, [session.handlePackage] gr will exit now, left gr num %d", s.sessionToken(), grNum)
		s.stop()
		if err != nil {
			log.Errorf("%s, [session.handlePackage] error:%+v", s.sessionToken(), perrors.WithStack(err))
			if s != nil || s.listener != nil {
				s.listener.OnError(s, err)
			}
		}

		s.listener.OnClose(s)
		s.gc()
	}()

	if _, ok := s.Connection.(*gettyTCPConn); ok {
		if s.reader == nil {
			errStr := fmt.Sprintf("session{name:%s, conn:%#v, reader:%#v}", s.name, s.Connection, s.reader)
			log.Error(errStr)
			panic(errStr)
		}

		err = s.handleTCPPackage()
	} else if _, ok := s.Connection.(*gettyWSConn); ok {
		err = s.handleWSPackage()
	} else if _, ok := s.Connection.(*gettyUDPConn); ok {
		err = s.handleUDPPackage()
	} else {
		panic(fmt.Sprintf("unknown type session{%#v}", s))
	}
}

// get package from tcp stream(packet)
func (s *session) handleTCPPackage() error {
	var (
		ok       bool
		err      error
		netError net.Error
		conn     *gettyTCPConn
		exit     bool
		bufLen   int
		pkgLen   int
		buf      []byte
		pktBuf   *gxbytes.Buffer
		pkg      interface{}
	)

	pktBuf = gxbytes.NewBuffer(nil)

	conn = s.Connection.(*gettyTCPConn)
	for {
		if s.IsClosed() {
			err = nil
			// do not handle the left stream in pktBuf and exit asap.
			// it is impossible packing a package by the left stream.
			break
		}

		bufLen = 0
		for {
			// for clause for the network timeout condition check
			// s.conn.SetReadTimeout(time.Now().Add(s.rTimeout))
			buf = pktBuf.WriteNextBegin(maxReadBufLen)
			bufLen, err = conn.recv(buf)
			if err != nil {
				if netError, ok = perrors.Cause(err).(net.Error); ok && netError.Timeout() {
					break
				}
				if perrors.Cause(err) == io.EOF {
					log.Infof("%s, session.conn read EOF, client send over, session exit", s.sessionToken())
					err = nil
					exit = true
					if bufLen != 0 {
						// as https://github.com/apache/dubbo-getty/issues/77#issuecomment-939652203
						// this branch is impossible. Even if it happens, the bufLen will be zero and the error
						// is io.EOF when getty continues to read the socket.
						exit = false
						log.Infof("%s, session.conn read EOF, while the bufLen(%d) is non-zero.", s.sessionToken())
					}
					break
				}
				log.Errorf("%s, [session.conn.read] = error:%+v", s.sessionToken(), perrors.WithStack(err))
				exit = true
			}
			break
		}
		if 0 != bufLen {
			pktBuf.WriteNextEnd(bufLen)
			for {
				if pktBuf.Len() <= 0 {
					break
				}
				pkg, pkgLen, err = s.reader.Read(s, pktBuf.Bytes())
				// for case 3/case 4
				if err == nil && s.maxMsgLen > 0 && pkgLen > int(s.maxMsgLen) {
					err = perrors.Errorf("pkgLen %d > session max message len %d", pkgLen, s.maxMsgLen)
				}
				// handle case 1
				if err != nil {
					log.Warnf("%s, [session.handleTCPPackage] = len{%d}, error:%+v",
						s.sessionToken(), pkgLen, perrors.WithStack(err))
					exit = true
					break
				}
				// handle case 2/case 3
				if pkg == nil {
					break
				}
				// handle case 4
				s.UpdateActive()
				s.addTask(pkg)
				pktBuf.Next(pkgLen)
				// continue to handle case 5
			}
		}
		if exit {
			break
		}
	}

	return perrors.WithStack(err)
}

// get package from udp packet
func (s *session) handleUDPPackage() error {
	var (
		ok        bool
		err       error
		netError  net.Error
		conn      *gettyUDPConn
		bufLen    int
		maxBufLen int
		bufp      *[]byte
		buf       []byte
		addr      *net.UDPAddr
		pkgLen    int
		pkg       interface{}
	)

	conn = s.Connection.(*gettyUDPConn)
	maxBufLen = int(s.maxMsgLen + maxReadBufLen)
	if int(s.maxMsgLen<<1) < bufLen {
		maxBufLen = int(s.maxMsgLen << 1)
	}
	bufp = gxbytes.AcquireBytes(maxBufLen)
	defer gxbytes.ReleaseBytes(bufp)
	buf = *bufp
	for {
		if s.IsClosed() {
			break
		}

		bufLen, addr, err = conn.recv(buf)
		log.Debugf("conn.read() = bufLen:%d, addr:%#v, err:%+v", bufLen, addr, perrors.WithStack(err))
		if netError, ok = perrors.Cause(err).(net.Error); ok && netError.Timeout() {
			continue
		}
		if err != nil {
			log.Errorf("%s, [session.handleUDPPackage] = len:%d, error:%+v",
				s.sessionToken(), bufLen, perrors.WithStack(err))
			err = perrors.Wrapf(err, "conn.read()")
			break
		}

		if bufLen == 0 {
			log.Errorf("conn.read() = bufLen:%d, addr:%s, err:%+v", bufLen, addr, perrors.WithStack(err))
			continue
		}

		if bufLen == len(connectPingPackage) && bytes.Equal(connectPingPackage, buf[:bufLen]) {
			log.Infof("got %s connectPingPackage", addr)
			continue
		}

		pkg, pkgLen, err = s.reader.Read(s, buf[:bufLen])
		log.Debugf("s.reader.Read() = pkg:%#v, pkgLen:%d, err:%+v", pkg, pkgLen, perrors.WithStack(err))
		if err == nil && s.maxMsgLen > 0 && bufLen > int(s.maxMsgLen) {
			err = perrors.Errorf("Message Too Long, bufLen %d, session max message len %d", bufLen, s.maxMsgLen)
		}
		if err != nil {
			log.Warnf("%s, [session.handleUDPPackage] = len:%d, error:%+v",
				s.sessionToken(), pkgLen, perrors.WithStack(err))
			continue
		}
		if pkgLen == 0 {
			log.Errorf("s.reader.Read() = pkg:%#v, pkgLen:%d, err:%+v", pkg, pkgLen, perrors.WithStack(err))
			continue
		}

		s.UpdateActive()
		s.addTask(UDPContext{Pkg: pkg, PeerAddr: addr})
	}

	return perrors.WithStack(err)
}

// get package from websocket stream
func (s *session) handleWSPackage() error {
	var (
		ok           bool
		err          error
		netError     net.Error
		length       int
		conn         *gettyWSConn
		pkg          []byte
		unmarshalPkg interface{}
	)

	conn = s.Connection.(*gettyWSConn)
	for {
		if s.IsClosed() {
			break
		}
		pkg, err = conn.recv()
		if netError, ok = perrors.Cause(err).(net.Error); ok && netError.Timeout() {
			continue
		}
		if err != nil {
			log.Warnf("%s, [session.handleWSPackage] = error:%+v",
				s.sessionToken(), perrors.WithStack(err))
			return perrors.WithStack(err)
		}
		s.UpdateActive()
		if s.reader != nil {
			unmarshalPkg, length, err = s.reader.Read(s, pkg)
			if err == nil && s.maxMsgLen > 0 && length > int(s.maxMsgLen) {
				err = perrors.Errorf("Message Too Long, length %d, session max message len %d", length, s.maxMsgLen)
			}
			if err != nil {
				log.Warnf("%s, [session.handleWSPackage] = len:%d, error:%+v",
					s.sessionToken(), length, perrors.WithStack(err))
				continue
			}

			s.addTask(unmarshalPkg)
		} else {
			s.addTask(pkg)
		}
	}

	return nil
}

func (s *session) stop() {
	select {
	case <-s.done: // s.done is a blocked channel. if it has not been closed, the default branch will be invoked.
		return

	default:
		s.once.Do(func() {
			// let read/Write timeout asap
			now := time.Now()
			if conn := s.Conn(); conn != nil {
				conn.SetReadDeadline(now.Add(s.readTimeout()))
				conn.SetWriteDeadline(now.Add(s.writeTimeout()))
			}
			close(s.done)
			c := s.GetAttribute(sessionClientKey)
			if clt, ok := c.(*client); ok {
				clt.reConnect()
			}
		})
	}
}

func (s *session) gc() {
	var conn Connection

	s.lock.Lock()
	if s.attrs != nil {
		s.attrs = nil
		conn = s.Connection
		s.Connection = nil
	}
	s.lock.Unlock()

	go func() {
		if conn != nil {
			conn.close(int(s.wait))
		}
	}()
}

// Close will be invoked by NewSessionCallback(if return error is not nil)
// or (session)handleLoop automatically. It's thread safe.
func (s *session) Close() {
	s.stop()
	log.Infof("%s closed now. its current gr num is %d", s.sessionToken(), s.grNum.Load())
}

// GetActive return connection's time
func (s *session) GetActive() time.Time {
	if s == nil {
		return launchTime
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.Connection != nil {
		return s.Connection.GetActive()
	}
	return launchTime
}

// UpdateActive update connection's active time
func (s *session) UpdateActive() {
	if s == nil {
		return
	}
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.Connection != nil {
		s.Connection.UpdateActive()
	}
}

func (s *session) ID() uint32 {
	if s == nil {
		return 0
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.Connection != nil {
		return s.Connection.ID()
	}
	return 0
}

func (s *session) LocalAddr() string {
	if s == nil {
		return ""
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.Connection != nil {
		return s.Connection.LocalAddr()
	}
	return ""
}

func (s *session) RemoteAddr() string {
	if s == nil {
		return ""
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.Connection != nil {
		return s.Connection.RemoteAddr()
	}
	return ""
}

func (s *session) incReadPkgNum() {
	if s == nil {
		return
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.Connection != nil {
		s.Connection.incReadPkgNum()
	}
}

func (s *session) incWritePkgNum() {
	if s == nil {
		return
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.Connection != nil {
		s.Connection.incWritePkgNum()
	}
}

func (s *session) send(pkg interface{}) (int, error) {
	if s == nil {
		return 0, nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.Connection != nil {
		return s.Connection.send(pkg)
	}
	return 0, nil
}

func (s *session) readTimeout() time.Duration {
	if s == nil {
		return time.Duration(0)
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.Connection != nil {
		return s.Connection.readTimeout()
	}
	return time.Duration(0)
}

func (s *session) setSession(ss Session) {
	if s == nil {
		return
	}
	s.lock.RLock()
	if s.Connection != nil {
		s.Connection.setSession(ss)
	}
	s.lock.RUnlock()
}
