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
	"compress/flate"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

import (
	"github.com/golang/snappy"
	"github.com/gorilla/websocket"
	perrors "github.com/pkg/errors"
	uatomic "go.uber.org/atomic"
)

var (
	launchTime = time.Now()
	connID     uatomic.Uint32
)

// Connection wrap some connection params and operations
type Connection interface {
	ID() uint32
	SetCompressType(CompressType)
	LocalAddr() string
	RemoteAddr() string
	incReadPkgNum()
	incWritePkgNum()
	// UpdateActive update session's active time
	UpdateActive()
	// GetActive get session's active time
	GetActive() time.Time
	readTimeout() time.Duration
	// SetReadTimeout sets deadline for the future read calls.
	SetReadTimeout(time.Duration)
	writeTimeout() time.Duration
	// SetWriteTimeout sets deadline for the future read calls.
	SetWriteTimeout(time.Duration)
	send(interface{}) (int, error)
	// don't distinguish between tcp connection and websocket connection. Because
	// gorilla/websocket/conn.go:(Conn)Close also invoke net.Conn.Close
	close(int)
	// set related session
	setSession(Session)
}

// ///////////////////////////////////////
// getty connection
// ///////////////////////////////////////

type gettyConn struct {
	id            uint32
	compress      CompressType
	padding1      uint8
	padding2      uint16
	readBytes     uatomic.Uint32   // read bytes
	writeBytes    uatomic.Uint32   // write bytes
	readPkgNum    uatomic.Uint32   // send pkg number
	writePkgNum   uatomic.Uint32   // recv pkg number
	active        uatomic.Int64    // last active, in milliseconds
	rTimeout      uatomic.Duration // network current limiting
	wTimeout      uatomic.Duration
	rLastDeadline uatomic.Time // last network read time
	wLastDeadline uatomic.Time // last network write time
	local         string       // local address
	peer          string       // peer address
	ss            Session
}

func (c *gettyConn) ID() uint32 {
	return c.id
}

func (c *gettyConn) LocalAddr() string {
	return c.local
}

func (c *gettyConn) RemoteAddr() string {
	return c.peer
}

func (c *gettyConn) incReadPkgNum() {
	c.readPkgNum.Add(1)
}

func (c *gettyConn) incWritePkgNum() {
	c.writePkgNum.Add(1)
}

func (c *gettyConn) UpdateActive() {
	c.active.Store(int64(time.Since(launchTime)))
}

func (c *gettyConn) GetActive() time.Time {
	return launchTime.Add(time.Duration(c.active.Load()))
}

func (c *gettyConn) send(interface{}) (int, error) {
	return 0, nil
}

func (c *gettyConn) close(int) {}

func (c gettyConn) readTimeout() time.Duration {
	return c.rTimeout.Load()
}

func (c *gettyConn) setSession(ss Session) {
	c.ss = ss
}

// SetReadTimeout Pls do not set read deadline for websocket connection. AlexStocks 20180310
// gorilla/websocket/conn.go:NextReader will always fail when got a timeout error.
//
// Pls do not set read deadline when using compression. AlexStocks 20180314.
func (c *gettyConn) SetReadTimeout(rTimeout time.Duration) {
	if rTimeout < 1 {
		panic("@rTimeout < 1")
	}

	c.rTimeout.Store(rTimeout)
	if c.wTimeout.Load() == 0 {
		c.wTimeout.Store(rTimeout)
	}
}

func (c gettyConn) writeTimeout() time.Duration {
	return c.wTimeout.Load()
}

// SetWriteTimeout Pls do not set write deadline for websocket connection. AlexStocks 20180310
// gorilla/websocket/conn.go:NextWriter will always fail when got a timeout error.
//
// Pls do not set write deadline when using compression. AlexStocks 20180314.
func (c *gettyConn) SetWriteTimeout(wTimeout time.Duration) {
	if wTimeout < 1 {
		panic("@wTimeout < 1")
	}

	c.wTimeout.Store(wTimeout)
	if c.rTimeout.Load() == 0 {
		c.rTimeout.Store(wTimeout)
	}
}

/////////////////////////////////////////
// getty tcp connection
/////////////////////////////////////////

type gettyTCPConn struct {
	gettyConn
	reader io.Reader
	writer io.Writer
	conn   net.Conn
}

// create gettyTCPConn
func newGettyTCPConn(conn net.Conn) *gettyTCPConn {
	if conn == nil {
		panic("newGettyTCPConn(conn):@conn is nil")
	}
	var localAddr, peerAddr string
	//  check conn.LocalAddr or conn.RemoteAddr is nil to defeat panic on 2016/09/27
	if conn.LocalAddr() != nil {
		localAddr = conn.LocalAddr().String()
	}
	if conn.RemoteAddr() != nil {
		peerAddr = conn.RemoteAddr().String()
	}

	return &gettyTCPConn{
		conn:   conn,
		reader: io.Reader(conn),
		writer: io.Writer(conn),
		gettyConn: gettyConn{
			id:       connID.Add(1),
			rTimeout: *uatomic.NewDuration(netIOTimeout),
			wTimeout: *uatomic.NewDuration(netIOTimeout),
			local:    localAddr,
			peer:     peerAddr,
			compress: CompressNone,
		},
	}
}

// for zip compress
type writeFlusher struct {
	flusher *flate.Writer
	lock    sync.Mutex
}

func (t *writeFlusher) Write(p []byte) (int, error) {
	var (
		n   int
		err error
	)
	t.lock.Lock()
	defer t.lock.Unlock()
	n, err = t.flusher.Write(p)
	if err != nil {
		return n, perrors.WithStack(err)
	}
	if err := t.flusher.Flush(); err != nil {
		return 0, perrors.WithStack(err)
	}

	return n, nil
}

// SetCompressType set compress type(tcp: zip/snappy, websocket:zip)
func (t *gettyTCPConn) SetCompressType(c CompressType) {
	switch c {
	case CompressNone, CompressZip, CompressBestSpeed, CompressBestCompression, CompressHuffman:
		ioReader := io.Reader(t.conn)
		t.reader = flate.NewReader(ioReader)

		ioWriter := io.Writer(t.conn)
		w, err := flate.NewWriter(ioWriter, int(c))
		if err != nil {
			panic(fmt.Sprintf("flate.NewReader(flate.DefaultCompress) = err(%s)", err))
		}
		t.writer = &writeFlusher{flusher: w}

	case CompressSnappy:
		ioReader := io.Reader(t.conn)
		t.reader = snappy.NewReader(ioReader)
		ioWriter := io.Writer(t.conn)
		t.writer = snappy.NewBufferedWriter(ioWriter)

	default:
		panic(fmt.Sprintf("illegal comparess type %d", c))
	}
	t.compress = c
}

// tcp connection read
func (t *gettyTCPConn) recv(p []byte) (int, error) {
	var (
		err         error
		currentTime time.Time
		length      int
	)

	// set read timeout deadline
	if t.compress == CompressNone && t.rTimeout.Load() > 0 {
		// Optimization: update read deadline only if more than 25%
		// of the last read deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime = time.Now()
		if currentTime.Sub(t.rLastDeadline.Load()) > t.rTimeout.Load()>>2 {
			if err = t.conn.SetReadDeadline(currentTime.Add(t.rTimeout.Load())); err != nil {
				// just a timeout error
				return 0, perrors.WithStack(err)
			}
			t.rLastDeadline.Store(currentTime)
		}
	}

	length, err = t.reader.Read(p)
	t.readBytes.Add(uint32(length))
	return length, perrors.WithStack(err)
}

// tcp connection write
func (t *gettyTCPConn) send(pkg interface{}) (int, error) {
	var (
		err         error
		currentTime time.Time
		ok          bool
		p           []byte
		length      int
		lg          int64
	)

	if t.compress == CompressNone && t.wTimeout.Load() > 0 {
		// Optimization: update write deadline only if more than 25%
		// of the last write deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime = time.Now()
		if currentTime.Sub(t.wLastDeadline.Load()) > t.wTimeout.Load()>>2 {
			if err = t.conn.SetWriteDeadline(currentTime.Add(t.wTimeout.Load())); err != nil {
				return 0, perrors.WithStack(err)
			}
			t.wLastDeadline.Store(currentTime)
		}
	}

	if buffers, ok := pkg.([][]byte); ok {
		netBuf := net.Buffers(buffers)
		lg, err = netBuf.WriteTo(t.conn)
		if err == nil {
			t.writeBytes.Add((uint32)(lg))
			t.writePkgNum.Add((uint32)(len(buffers)))
		}
		log.Debugf("localAddr: %s, remoteAddr:%s, now:%s, length:%d, err:%s",
			t.conn.LocalAddr(), t.conn.RemoteAddr(), currentTime, length, err)
		return int(lg), perrors.WithStack(err)
	}

	if p, ok = pkg.([]byte); ok {
		length, err = t.writer.Write(p)
		if err == nil {
			t.writeBytes.Add((uint32)(len(p)))
			t.writePkgNum.Add(1)
		}
		log.Debugf("localAddr: %s, remoteAddr:%s, now:%s, length:%d, err:%v",
			t.conn.LocalAddr(), t.conn.RemoteAddr(), currentTime, length, err)
		return length, perrors.WithStack(err)
	}

	return 0, perrors.Errorf("illegal @pkg{%#v} type", pkg)
}

// close tcp connection
func (t *gettyTCPConn) close(waitSec int) {
	// if tcpConn, ok := t.conn.(*net.TCPConn); ok {
	// tcpConn.SetLinger(0)
	// }

	if t.conn != nil {
		if writer, ok := t.writer.(*snappy.Writer); ok {
			if err := writer.Close(); err != nil {
				log.Errorf("snappy.Writer.Close() = error:%+v", err)
			}
		}
		if conn, ok := t.conn.(*net.TCPConn); ok {
			_ = conn.SetLinger(waitSec)
			_ = conn.Close()
		} else {
			_ = t.conn.(*tls.Conn).Close()
		}
		t.conn = nil
	}
}

// ///////////////////////////////////////
// getty udp connection
// ///////////////////////////////////////

type UDPContext struct {
	Pkg      interface{}
	PeerAddr *net.UDPAddr
}

func (c UDPContext) String() string {
	return fmt.Sprintf("{pkg:%#v, peer addr:%s}", c.Pkg, c.PeerAddr)
}

type gettyUDPConn struct {
	gettyConn
	compressType CompressType
	conn         *net.UDPConn // for server
}

// create gettyUDPConn
func newGettyUDPConn(conn *net.UDPConn) *gettyUDPConn {
	if conn == nil {
		panic("newGettyUDPConn(conn):@conn is nil")
	}

	var localAddr, peerAddr string
	if conn.LocalAddr() != nil {
		localAddr = conn.LocalAddr().String()
	}

	if conn.RemoteAddr() != nil {
		// connected udp
		peerAddr = conn.RemoteAddr().String()
	}

	return &gettyUDPConn{
		conn: conn,
		gettyConn: gettyConn{
			id:       connID.Add(1),
			rTimeout: *uatomic.NewDuration(netIOTimeout),
			wTimeout: *uatomic.NewDuration(netIOTimeout),
			local:    localAddr,
			peer:     peerAddr,
			compress: CompressNone,
		},
	}
}

func (u *gettyUDPConn) SetCompressType(c CompressType) {
	switch c {
	case CompressNone, CompressZip, CompressBestSpeed, CompressBestCompression, CompressHuffman, CompressSnappy:
		u.compressType = c

	default:
		panic(fmt.Sprintf("illegal comparess type %d", c))
	}
}

// udp connection read
func (u *gettyUDPConn) recv(p []byte) (int, *net.UDPAddr, error) {
	if u.rTimeout.Load() > 0 {
		// Optimization: update read deadline only if more than 25%
		// of the last read deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime := time.Now()
		if currentTime.Sub(u.rLastDeadline.Load()) > u.rTimeout.Load()>>2 {
			if err := u.conn.SetReadDeadline(currentTime.Add(u.rTimeout.Load())); err != nil {
				return 0, nil, perrors.WithStack(err)
			}
			u.rLastDeadline.Store(currentTime)
		}
	}

	length, addr, err := u.conn.ReadFromUDP(p) // connected udp also can get return @addr
	log.Debugf("ReadFromUDP(p:%d) = {length:%d, peerAddr:%s, error:%v}", len(p), length, addr, err)
	if err == nil {
		u.readBytes.Add(uint32(length))
	}

	return length, addr, perrors.WithStack(err)
}

// write udp packet, @ctx should be of type UDPContext
func (u *gettyUDPConn) send(udpCtx interface{}) (int, error) {
	var (
		err         error
		currentTime time.Time
		length      int
		ok          bool
		ctx         UDPContext
		buf         []byte
		peerAddr    *net.UDPAddr
	)

	if ctx, ok = udpCtx.(UDPContext); !ok {
		return 0, perrors.Errorf("illegal @udpCtx{%s} type, @udpCtx type:%T", udpCtx, udpCtx)
	}
	if buf, ok = ctx.Pkg.([]byte); !ok {
		return 0, perrors.Errorf("illegal @udpCtx.Pkg{%#v} type", udpCtx)
	}
	if u.ss.EndPoint().EndPointType() == UDP_ENDPOINT {
		peerAddr = ctx.PeerAddr
		if peerAddr == nil {
			return 0, ErrNullPeerAddr
		}
	}

	if u.wTimeout.Load() > 0 {
		// Optimization: update write deadline only if more than 25%
		// of the last write deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime = time.Now()
		if currentTime.Sub(u.wLastDeadline.Load()) > u.wTimeout.Load()>>2 {
			if err = u.conn.SetWriteDeadline(currentTime.Add(u.wTimeout.Load())); err != nil {
				return 0, perrors.WithStack(err)
			}
			u.wLastDeadline.Store(currentTime)
		}
	}

	if length, _, err = u.conn.WriteMsgUDP(buf, nil, peerAddr); err == nil {
		u.writeBytes.Add((uint32)(len(buf)))
		u.writePkgNum.Add(1)
	}
	log.Debugf("WriteMsgUDP(peerAddr:%s) = {length:%d, error:%v}", peerAddr, length, err)

	return length, perrors.WithStack(err)
}

// close udp connection
func (u *gettyUDPConn) close(_ int) {
	if u.conn != nil {
		u.conn.Close()
		u.conn = nil
	}
}

// ///////////////////////////////////////
// getty websocket connection
// ///////////////////////////////////////

type gettyWSConn struct {
	gettyConn
	conn *websocket.Conn
}

// create websocket connection
func newGettyWSConn(conn *websocket.Conn) *gettyWSConn {
	if conn == nil {
		panic("newGettyWSConn(conn):@conn is nil")
	}
	var localAddr, peerAddr string
	//  check conn.LocalAddr or conn.RemoetAddr is nil to defeat panic on 2016/09/27
	if conn.LocalAddr() != nil {
		localAddr = conn.LocalAddr().String()
	}
	if conn.RemoteAddr() != nil {
		peerAddr = conn.RemoteAddr().String()
	}

	gettyWSConn := &gettyWSConn{
		conn: conn,
		gettyConn: gettyConn{
			id:       connID.Add(1),
			rTimeout: *uatomic.NewDuration(netIOTimeout),
			wTimeout: *uatomic.NewDuration(netIOTimeout),
			local:    localAddr,
			peer:     peerAddr,
			compress: CompressNone,
		},
	}
	conn.EnableWriteCompression(false)
	conn.SetPingHandler(gettyWSConn.handlePing)
	conn.SetPongHandler(gettyWSConn.handlePong)

	return gettyWSConn
}

// SetCompressType set compress type
func (w *gettyWSConn) SetCompressType(c CompressType) {
	switch c {
	case CompressNone, CompressZip, CompressBestSpeed, CompressBestCompression, CompressHuffman:
		w.conn.EnableWriteCompression(true)
		w.conn.SetCompressionLevel(int(c))

	default:
		panic(fmt.Sprintf("illegal comparess type %d", c))
	}
	w.compress = c
}

func (w *gettyWSConn) handlePing(message string) error {
	err := w.writePong([]byte(message))
	if err == websocket.ErrCloseSent {
		err = nil
	} else if e, ok := err.(net.Error); ok && e.Temporary() {
		err = nil
	}
	if err == nil {
		w.UpdateActive()
	}

	return perrors.WithStack(err)
}

func (w *gettyWSConn) handlePong(string) error {
	w.UpdateActive()
	return nil
}

// websocket connection read
func (w *gettyWSConn) recv() ([]byte, error) {
	// Pls do not set read deadline when using ReadMessage. AlexStocks 20180310
	// gorilla/websocket/conn.go:NextReader will always fail when got a timeout error.
	_, b, e := w.conn.ReadMessage() // the first return value is message type.
	if e == nil {
		w.readBytes.Add((uint32)(len(b)))
	} else {
		if websocket.IsUnexpectedCloseError(e, websocket.CloseGoingAway) {
			log.Warnf("websocket unexpected close error: %v", e)
		}
	}

	return b, perrors.WithStack(e)
}

func (w *gettyWSConn) updateWriteDeadline() error {
	var (
		err         error
		currentTime time.Time
	)

	if w.wTimeout.Load() > 0 {
		// Optimization: update write deadline only if more than 25%
		// of the last write deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime = time.Now()
		if currentTime.Sub(w.wLastDeadline.Load()) > w.wTimeout.Load()>>2 {
			if err = w.conn.SetWriteDeadline(currentTime.Add(w.wTimeout.Load())); err != nil {
				return perrors.WithStack(err)
			}
			w.wLastDeadline.Store(currentTime)
		}
	}

	return nil
}

// websocket connection write
func (w *gettyWSConn) send(pkg interface{}) (int, error) {
	var (
		err error
		ok  bool
		p   []byte
	)

	if p, ok = pkg.([]byte); !ok {
		return 0, perrors.Errorf("illegal @pkg{%#v} type", pkg)
	}

	w.updateWriteDeadline()
	if err = w.conn.WriteMessage(websocket.BinaryMessage, p); err == nil {
		w.writeBytes.Add((uint32)(len(p)))
		w.writePkgNum.Add(1)
	}
	return len(p), perrors.WithStack(err)
}

func (w *gettyWSConn) writePing() error {
	w.updateWriteDeadline()
	return perrors.WithStack(w.conn.WriteMessage(websocket.PingMessage, []byte{}))
}

func (w *gettyWSConn) writePong(message []byte) error {
	w.updateWriteDeadline()
	return perrors.WithStack(w.conn.WriteMessage(websocket.PongMessage, message))
}

// close websocket connection
func (w *gettyWSConn) close(waitSec int) {
	w.updateWriteDeadline()
	w.conn.WriteMessage(websocket.CloseMessage, []byte("bye-bye!!!"))
	conn := w.conn.UnderlyingConn()
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetLinger(waitSec)
	} else if wsConn, ok := conn.(*tls.Conn); ok {
		wsConn.CloseWrite()
	}
	w.conn.Close()
}
