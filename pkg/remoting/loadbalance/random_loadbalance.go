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

	getty "github.com/apache/dubbo-getty"
)

func RandomLoadBalance(sessions *sync.Map, xid string) getty.Session {
	//collect sync.Map keys
	//filted out closed session instance
	var keys []getty.Session
	sessions.Range(func(key, value interface{}) bool {
		session := key.(getty.Session)
		if session.IsClosed() {
			sessions.Delete(key)
		} else {
			keys = append(keys, session)
		}
		return true
	})
	//keys eq 0 means there are no available session
	if len(keys) == 0 {
		return nil
	}
	//random in keys
	randomIndex := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(keys))
	return keys[randomIndex]
}
