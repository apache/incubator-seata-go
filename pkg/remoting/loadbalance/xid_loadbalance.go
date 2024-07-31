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
	"strings"
	"sync"

	getty "github.com/apache/dubbo-getty"
)

func XidLoadBalance(sessions *sync.Map, xid string) getty.Session {
	var session getty.Session

	// ip:port:transactionId
	tmpSplits := strings.Split(xid, ":")
	if len(tmpSplits) == 3 {
		ip := tmpSplits[0]
		port := tmpSplits[1]
		ipPort := ip + ":" + port
		sessions.Range(func(key, value interface{}) bool {
			tmpSession := key.(getty.Session)
			if tmpSession.IsClosed() {
				sessions.Delete(tmpSession)
				return true
			}
			connectedIpPort := tmpSession.RemoteAddr()
			if ipPort == connectedIpPort {
				session = tmpSession
				return false
			}
			return true
		})
	}

	if session == nil {
		return RandomLoadBalance(sessions, xid)
	}

	return session
}
