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
)

type ServerOption func(*ServerOptions)

type ServerOptions struct {
	addr string
	// tls
	sslEnabled       bool
	tlsConfigBuilder TlsConfigBuilder
	// websocket
	path       string
	cert       string
	privateKey string
	caCert     string
	// task queue
	tPool gxsync.GenericTaskPool
}

// WithLocalAddress @addr server listen address.
func WithLocalAddress(addr string) ServerOption {
	return func(o *ServerOptions) {
		o.addr = addr
	}
}

// WithWebsocketServerPath @path: websocket request url path
func WithWebsocketServerPath(path string) ServerOption {
	return func(o *ServerOptions) {
		o.path = path
	}
}

// WithWebsocketServerCert @cert: server certificate file
func WithWebsocketServerCert(cert string) ServerOption {
	return func(o *ServerOptions) {
		o.cert = cert
	}
}

// WithWebsocketServerPrivateKey @key: server private key(contains its public key)
func WithWebsocketServerPrivateKey(key string) ServerOption {
	return func(o *ServerOptions) {
		o.privateKey = key
	}
}

// WithWebsocketServerRootCert @cert is the root certificate file to verify the legitimacy of server
func WithWebsocketServerRootCert(cert string) ServerOption {
	return func(o *ServerOptions) {
		o.caCert = cert
	}
}

// WithServerTaskPool @pool server task pool.
func WithServerTaskPool(pool gxsync.GenericTaskPool) ServerOption {
	return func(o *ServerOptions) {
		o.tPool = pool
	}
}

// WithServerSslEnabled enable use tls
func WithServerSslEnabled(sslEnabled bool) ServerOption {
	return func(o *ServerOptions) {
		o.sslEnabled = sslEnabled
	}
}

// WithServerTlsConfigBuilder sslConfig is tls config
func WithServerTlsConfigBuilder(tlsConfigBuilder TlsConfigBuilder) ServerOption {
	return func(o *ServerOptions) {
		o.tlsConfigBuilder = tlsConfigBuilder
	}
}

/////////////////////////////////////////
// Client Options
/////////////////////////////////////////

type ClientOption func(*ClientOptions)

type ClientOptions struct {
	addr              string
	number            int
	reconnectInterval int // reConnect Interval

	// tls
	sslEnabled       bool
	tlsConfigBuilder TlsConfigBuilder

	// the cert file of wss server which may contain server domain, server ip, the starting effective date, effective
	// duration, the hash alg, the len of the private key.
	// wss client will use it.
	cert string
	// task queue
	tPool gxsync.GenericTaskPool
}

// WithServerAddress @addr is server address.
func WithServerAddress(addr string) ClientOption {
	return func(o *ClientOptions) {
		o.addr = addr
	}
}

// WithReconnectInterval @reconnectInterval is server address.
func WithReconnectInterval(reconnectInterval int) ClientOption {
	return func(o *ClientOptions) {
		if 0 < reconnectInterval {
			o.reconnectInterval = reconnectInterval
		}
	}
}

// WithClientTaskPool @pool client task pool.
func WithClientTaskPool(pool gxsync.GenericTaskPool) ClientOption {
	return func(o *ClientOptions) {
		o.tPool = pool
	}
}

// WithConnectionNumber @num is connection number.
func WithConnectionNumber(num int) ClientOption {
	return func(o *ClientOptions) {
		if 0 < num {
			o.number = num
		}
	}
}

// WithRootCertificateFile @certs is client certificate file. it can be empty.
func WithRootCertificateFile(cert string) ClientOption {
	return func(o *ClientOptions) {
		o.cert = cert
	}
}

// WithClientSslEnabled enable use tls
func WithClientSslEnabled(sslEnabled bool) ClientOption {
	return func(o *ClientOptions) {
		o.sslEnabled = sslEnabled
	}
}

// WithClientTlsConfigBuilder sslConfig is tls config
func WithClientTlsConfigBuilder(tlsConfigBuilder TlsConfigBuilder) ClientOption {
	return func(o *ClientOptions) {
		o.tlsConfigBuilder = tlsConfigBuilder
	}
}
