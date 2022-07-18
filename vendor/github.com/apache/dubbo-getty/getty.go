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
	gxsync "github.com/dubbogo/gost/sync"

	perrors "github.com/pkg/errors"
)

var (
	ErrSessionClosed  = perrors.New("session Already Closed")
	ErrSessionBlocked = perrors.New("session Full Blocked")
	ErrNullPeerAddr   = perrors.New("peer address is nil")
)

// NewSessionCallback will be invoked when server accepts a new client connection or client connects to server successfully.
// If there are too many client connections or u do not want to connect a server again, u can return non-nil error. And
// then getty will close the new session.
type NewSessionCallback func(Session) error

// Reader is used to unmarshal a complete pkg from buffer
type Reader interface {
	// Read Parse tcp/udp/websocket pkg from buffer and if possible return a complete pkg.
	// When receiving a tcp network streaming segment, there are 4 cases as following:
	// case 1: a error found in the streaming segment;
	// case 2: can not unmarshal a pkg header from the streaming segment;
	// case 3: unmarshal a pkg header but can not unmarshal a pkg from the streaming segment;
	// case 4: just unmarshal a pkg from the streaming segment;
	// case 5: unmarshal more than one pkg from the streaming segment;
	//
	// The return value is (nil, 0, error) as case 1.
	// The return value is (nil, 0, nil) as case 2.
	// The return value is (nil, pkgLen, nil) as case 3.
	// The return value is (pkg, pkgLen, nil) as case 4.
	// The handleTcpPackage may invoke func Read many times as case 5.
	Read(Session, []byte) (interface{}, int, error)
}

// Writer is used to marshal pkg and write to session
type Writer interface {
	// Write if @Session is udpGettySession, the second parameter is UDPContext.
	Write(Session, interface{}) ([]byte, error)
}

// ReadWriter interface use for handle application packages
type ReadWriter interface {
	Reader
	Writer
}

// EventListener is used to process pkg that received from remote session
type EventListener interface {
	// OnOpen invoked when session opened
	// If the return error is not nil, @Session will be closed.
	OnOpen(Session) error

	// OnClose invoked when session closed.
	OnClose(Session)

	// OnError invoked when got error.
	OnError(Session, error)

	// OnCron invoked periodically, its period can be set by (Session)SetCronPeriod
	OnCron(Session)

	// OnMessage invoked when getty received a package. Pls attention that do not handle long time
	// logic processing in this func. You'd better set the package's maximum length.
	// If the message's length is greater than it, u should should return err in
	// Reader{Read} and getty will close this connection soon.
	//
	// If ur logic processing in this func will take a long time, u should start a goroutine
	// pool(like working thread pool in cpp) to handle the processing asynchronously. Or u
	// can do the logic processing in other asynchronous way.
	// !!!In short, ur OnMessage callback func should return asap.
	//
	// If this is a udp event listener, the second parameter type is UDPContext.
	OnMessage(Session, interface{})
}

// EndPoint represents the identity of the client/server
type EndPoint interface {
	// ID get EndPoint ID
	ID() EndPointID
	// EndPointType get endpoint type
	EndPointType() EndPointType
	// RunEventLoop run event loop and serves client request.
	RunEventLoop(newSession NewSessionCallback)
	// IsClosed check the endpoint has been closed
	IsClosed() bool
	// Close close the endpoint and free its resource
	Close()
	// GetTaskPool get task pool implemented by dubbogo/gost
	GetTaskPool() gxsync.GenericTaskPool
}
