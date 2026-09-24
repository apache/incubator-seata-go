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

package discovery

import "sync"

type registryChangeSubscription struct {
	once        sync.Once
	listener    RegistryChangeListener
	unsubscribe func()

	mu      sync.Mutex
	initial *RegistryChangeEvent
	latest  *RegistryChangeEvent
	closed  bool
	wakeCh  chan struct{}
	doneCh  chan struct{}
}

func newRegistryChangeSubscription(listener RegistryChangeListener) *registryChangeSubscription {
	return &registryChangeSubscription{
		listener: listener,
		wakeCh:   make(chan struct{}, 1),
		doneCh:   make(chan struct{}),
	}
}

func (s *registryChangeSubscription) initialize(initial RegistryChangeEvent, unsubscribe func()) {
	s.mu.Lock()
	s.initial = &initial
	s.unsubscribe = unsubscribe
	s.mu.Unlock()
}

func (s *registryChangeSubscription) start() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	go s.dispatch()
	s.signal()
}

func (s *registryChangeSubscription) dispatch() {
	for {
		select {
		case <-s.wakeCh:
			for {
				event, ok := s.nextEvent()
				if !ok {
					break
				}
				s.listener(event)
			}
		case <-s.doneCh:
			return
		}
	}
}

func (s *registryChangeSubscription) publish(event RegistryChangeEvent) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.latest = &event
	s.mu.Unlock()

	s.signal()
}

func (s *registryChangeSubscription) signal() {
	select {
	case s.wakeCh <- struct{}{}:
	default:
	}
}

func (s *registryChangeSubscription) nextEvent() (RegistryChangeEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return RegistryChangeEvent{}, false
	}
	if s.initial != nil {
		event := *s.initial
		s.initial = nil
		return event, true
	}
	if s.latest != nil {
		event := *s.latest
		s.latest = nil
		return event, true
	}
	return RegistryChangeEvent{}, false
}

func (s *registryChangeSubscription) Unsubscribe() {
	s.once.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.initial = nil
		s.latest = nil
		unsubscribe := s.unsubscribe
		s.mu.Unlock()

		if unsubscribe != nil {
			unsubscribe()
		}
		close(s.doneCh)
	})
}
