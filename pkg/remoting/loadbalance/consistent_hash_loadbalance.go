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
	"crypto/md5"
	"fmt"
	"sort"
	"sync"

	getty "github.com/apache/dubbo-getty"
)

const defaultVirtualNodeNumber = 10

type Consistent struct {
	// readLock/writeLock guards hashCircle / sortedHashNodes.
	sync.RWMutex
	// refreshMu serializes concurrent refresh operations so that two
	// goroutines do not redundantly rebuild the ring and race on the final
	// assignment.
	refreshMu sync.Mutex

	virtualNodeCount int
	hashCircle       map[int64]getty.Session
	sortedHashNodes  []int64
}

// NewConsistent builds a Consistent balancer. A non-positive virtualNodes
// falls back to the default.
func NewConsistent(virtualNodes int) *Consistent {
	if virtualNodes <= 0 {
		virtualNodes = defaultVirtualNodeNumber
	}
	return &Consistent{
		virtualNodeCount: virtualNodes,
		hashCircle:       make(map[int64]getty.Session),
	}
}

func (c *Consistent) hash(key string) int64 {
	hashByte := md5.Sum([]byte(key))
	var res int64
	for i := 0; i < 4; i++ {
		res <<= 8
		res |= int64(hashByte[i]) & 0xff
	}

	return res
}

func (c *Consistent) pickByHash(hashKey int64) getty.Session {
	c.RLock()
	defer c.RUnlock()

	if len(c.sortedHashNodes) == 0 {
		return nil
	}

	index := sort.Search(len(c.sortedHashNodes), func(i int) bool {
		return c.sortedHashNodes[i] >= hashKey
	})
	if index == len(c.sortedHashNodes) {
		index = 0
	}

	return c.hashCircle[c.sortedHashNodes[index]]
}

// pick get a node
func (c *Consistent) pick(sessions *sync.Map, key string) getty.Session {
	hashKey := c.hash(key)
	session := c.pickByHash(hashKey)
	if session == nil {
		c.refreshHashCircle(sessions)
		session = c.pickByHash(hashKey)
		if session == nil {
			return RandomLoadBalance(sessions, key)
		}
	}

	if session.IsClosed() {
		c.refreshHashCircle(sessions)
		session = c.pickByHash(hashKey)
		if session == nil || session.IsClosed() {
			return RandomLoadBalance(sessions, key)
		}
	}

	return session
}

// refreshHashCircle rebuilds the hash ring from the current sessions snapshot.
// refreshMu guarantees that two concurrent refreshes do not overwrite each
// other with stale rings.
func (c *Consistent) refreshHashCircle(sessions *sync.Map) {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	var sortedHashNodes []int64
	hashCircle := make(map[int64]getty.Session)

	sessions.Range(func(key, value interface{}) bool {
		session := key.(getty.Session)
		if session.IsClosed() {
			sessions.Delete(key)
			return true
		}

		for i := 0; i < c.virtualNodeCount; i++ {
			position := c.hash(fmt.Sprintf("%s%d", session.RemoteAddr(), i))
			hashCircle[position] = session
			sortedHashNodes = append(sortedHashNodes, position)
		}
		return true
	})

	sort.Slice(sortedHashNodes, func(i, j int) bool {
		return sortedHashNodes[i] < sortedHashNodes[j]
	})

	c.Lock()
	c.sortedHashNodes = sortedHashNodes
	c.hashCircle = hashCircle
	c.Unlock()
}

// ConsistentHashLoadBalance picks a session deterministically for xid using
// the supplied Consistent ring. If c is nil it falls back to random selection
// so callers that forget to plumb a ring stay functional.
func ConsistentHashLoadBalance(c *Consistent, sessions *sync.Map, xid string) getty.Session {
	if c == nil {
		return RandomLoadBalance(sessions, xid)
	}
	return c.pick(sessions, xid)
}
