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

var (
	once                     sync.Once
	defaultVirtualNodeNumber = 10
	consistentInstance       *Consistent
)

type Consistent struct {
	sync.RWMutex
	virtualNodeCount int
	// consistent hashCircle
	hashCircle      map[int64]getty.Session
	sortedHashNodes []int64
}

func (c *Consistent) put(key int64, session getty.Session) {
	c.Lock()
	defer c.Unlock()
	c.hashCircle[key] = session
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

// pick get a  node
func (c *Consistent) pick(sessions *sync.Map, key string) getty.Session {
	hashKey := c.hash(key)
	index := sort.Search(len(c.sortedHashNodes), func(i int) bool {
		return c.sortedHashNodes[i] >= hashKey
	})

	if index == len(c.sortedHashNodes) {
		return RandomLoadBalance(sessions, key)
	}

	c.RLock()
	session, ok := c.hashCircle[c.sortedHashNodes[index]]
	if !ok {
		c.RUnlock()
		return RandomLoadBalance(sessions, key)
	}
	c.RUnlock()

	if session.IsClosed() {
		go c.refreshHashCircle(sessions)
		return c.firstKey()
	}

	return session
}

// refreshHashCircle refresh hashCircle
func (c *Consistent) refreshHashCircle(sessions *sync.Map) {
	var sortedHashNodes []int64
	hashCircle := make(map[int64]getty.Session)
	var session getty.Session
	sessions.Range(func(key, value interface{}) bool {
		session = key.(getty.Session)
		for i := 0; i < defaultVirtualNodeNumber; i++ {
			if !session.IsClosed() {
				position := c.hash(fmt.Sprintf("%s%d", session.RemoteAddr(), i))
				hashCircle[position] = session
				sortedHashNodes = append(sortedHashNodes, position)
			} else {
				sessions.Delete(key)
			}
		}
		return true
	})

	// virtual node sort
	sort.Slice(sortedHashNodes, func(i, j int) bool {
		return sortedHashNodes[i] < sortedHashNodes[j]
	})

	c.sortedHashNodes = sortedHashNodes
	c.hashCircle = hashCircle
}

func (c *Consistent) firstKey() getty.Session {
	c.RLock()
	defer c.RUnlock()

	if len(c.sortedHashNodes) > 0 {
		return c.hashCircle[c.sortedHashNodes[0]]
	}

	return nil
}

func newConsistenceInstance(sessions *sync.Map) *Consistent {
	once.Do(func() {
		consistentInstance = &Consistent{
			hashCircle: make(map[int64]getty.Session),
		}
		// construct hash circle
		sessions.Range(func(key, value interface{}) bool {
			session := key.(getty.Session)
			for i := 0; i < defaultVirtualNodeNumber; i++ {
				if !session.IsClosed() {
					position := consistentInstance.hash(fmt.Sprintf("%s%d", session.RemoteAddr(), i))
					consistentInstance.put(position, session)
					consistentInstance.sortedHashNodes = append(consistentInstance.sortedHashNodes, position)
				} else {
					sessions.Delete(key)
				}
			}
			return true
		})

		// virtual node sort
		sort.Slice(consistentInstance.sortedHashNodes, func(i, j int) bool {
			return consistentInstance.sortedHashNodes[i] < consistentInstance.sortedHashNodes[j]
		})
	})

	return consistentInstance
}

func ConsistentHashLoadBalance(sessions *sync.Map, xid string) getty.Session {
	if consistentInstance == nil {
		newConsistenceInstance(sessions)
	}

	// pick a node
	return consistentInstance.pick(sessions, xid)
}
