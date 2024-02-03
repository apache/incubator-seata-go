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
	"math"
	"sort"
	"sync"
	"sync/atomic"

	getty "github.com/apache/dubbo-getty"
)

var sequence int32

func RoundRobinLoadBalance(sessions *sync.Map, s string) getty.Session {
	// collect sync.Map adderToSession
	// filter out closed session instance
	adderToSession := make(map[string]getty.Session, 0)
	// map has no sequence, we should sort it to make sure the sequence is always the same
	adders := make([]string, 0)
	sessions.Range(func(key, value interface{}) bool {
		session := key.(getty.Session)
		if session.IsClosed() {
			sessions.Delete(key)
		} else {
			adderToSession[session.RemoteAddr()] = session
			adders = append(adders, session.RemoteAddr())
		}
		return true
	})
	sort.Strings(adders)
	// adderToSession eq 0 means there are no available session
	if len(adderToSession) == 0 {
		return nil
	}
	index := getPositiveSequence() % len(adderToSession)
	return adderToSession[adders[index]]
}

func getPositiveSequence() int {
	for {
		current := atomic.LoadInt32(&sequence)
		var next int32
		if current == math.MaxInt32 {
			next = 0
		} else {
			next = current + 1
		}
		if atomic.CompareAndSwapInt32(&sequence, current, next) {
			return int(current)
		}
	}
}
