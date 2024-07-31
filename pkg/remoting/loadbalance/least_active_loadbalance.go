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
	"math/rand"
	"sync"
	"time"

	"seata.apache.org/seata-go/pkg/remoting/rpc"

	getty "github.com/apache/dubbo-getty"
)

func LeastActiveLoadBalance(sessions *sync.Map, xid string) getty.Session {
	var session getty.Session
	var leastActive int32 = -1
	leastCount := 0
	var leastIndexes []getty.Session
	sessions.Range(func(key, value interface{}) bool {
		session = key.(getty.Session)
		if session.IsClosed() {
			sessions.Delete(session)
		} else {
			active := rpc.GetStatus(session.RemoteAddr()).GetActive()
			if leastActive == -1 || active < leastActive {
				leastActive = active
				leastCount = 1
				if len(leastIndexes) > 0 {
					leastIndexes = leastIndexes[:0]
				}
				leastIndexes = append(leastIndexes, session)
			} else if active == leastActive {
				leastIndexes = append(leastIndexes, session)
				leastCount++
			}
		}
		return true
	})

	if leastCount == 0 {
		return nil
	}

	if leastCount == 1 {
		return leastIndexes[0]
	} else {
		return leastIndexes[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(leastCount)]
	}
}
