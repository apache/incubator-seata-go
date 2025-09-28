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

package types

import (
	"fmt"
	"sync"
)

// SingleFlight ensures that only one goroutine executes a function for a given key at a time
type SingleFlight struct {
	mu sync.Mutex
	m  map[string]*call
}

// call represents an in-flight or completed function call
type call struct {
	wg   sync.WaitGroup
	val  interface{}
	err  error
	dups int
}

// NewSingleFlight creates a new SingleFlight instance
func NewSingleFlight() *SingleFlight {
	return &SingleFlight{
		m: make(map[string]*call),
	}
}

// Do executes and returns the results of the given function, making sure that only one
// execution is in-flight for a given key at a time. If a duplicate comes in, the duplicate
// caller waits for the original to complete and receives the same results.
func (g *SingleFlight) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		c.dups++
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	g.doCall(c, key, fn)
	return c.val, c.err
}

// doCall handles the single call execution
func (g *SingleFlight) doCall(c *call, key string, fn func() (interface{}, error)) {
	defer func() {
		if r := recover(); r != nil {
			c.err = fmt.Errorf("panic in singleflight call: %v", r)
		}
		g.mu.Lock()
		defer g.mu.Unlock()
		c.wg.Done()
		delete(g.m, key)
	}()

	if fn == nil {
		c.err = fmt.Errorf("singleflight: nil function")
		return
	}

	c.val, c.err = fn()
}

// Forget tells the SingleFlight to forget about a key. Future calls to Do for this key
// will call the function rather than waiting for an earlier call to complete.
func (g *SingleFlight) Forget(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.m, key)
}
