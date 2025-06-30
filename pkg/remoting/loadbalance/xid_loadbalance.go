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

package loadbalance

import (
	"fmt"
	"strings"
	"sync"

	"seata.apache.org/seata-go/pkg/util/log"
	"seata.apache.org/seata-go/pkg/util/net"
	"seata.apache.org/seata-go/pkg/protocol/connection"
)

func XidLoadBalance(sessions *sync.Map, xid string) connection.Connection {
	var session connection.Connection
	const delimiter = ":"

	if len(xid) > 0 && strings.Contains(xid, delimiter) {
		// ip:port:transactionId -> ip:port
		index := strings.LastIndex(xid, delimiter)
		serverAddress := xid[:index]

		// ip:port -> port
		// ipv4/v6
		ip, port, err := net.SplitIPPortStr(serverAddress)
		if err != nil {
			log.Errorf("xid load balance err, xid:%s, %v , change use random load balance", xid, err)
		} else {
			sessions.Range(func(key, value interface{}) bool {
				tmpSession := key.(connection.Connection)
				if tmpSession.IsClosed() {
					sessions.Delete(tmpSession)
					return true
				}
				ipPort := fmt.Sprintf("%s:%d", ip, port)
				connectedIpPort := tmpSession.RemoteAddr()
				if ipPort == connectedIpPort {
					session = tmpSession
					return false
				}
				return true
			})
		}
	}

	if session == nil {
		return RandomLoadBalance(sessions, xid)
	}

	return session
}
