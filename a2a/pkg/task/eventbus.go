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

package task

import (
	"sync"
	"time"

	"seata-go-ai-a2a/pkg/types"
)

// EventBus manages task event subscriptions and publishing
type EventBus struct {
	mu            sync.RWMutex
	subscriptions map[string][]*TaskSubscription
	closing       bool
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscriptions: make(map[string][]*TaskSubscription),
	}
}

// Subscribe creates a new subscription for a task
func (eb *EventBus) Subscribe(taskID string, subscription *TaskSubscription) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closing {
		subscription.Close()
		return
	}

	eb.subscriptions[taskID] = append(eb.subscriptions[taskID], subscription)
}

// Unsubscribe removes a subscription for a task
func (eb *EventBus) Unsubscribe(taskID string, subscription *TaskSubscription) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs, exists := eb.subscriptions[taskID]
	if !exists {
		return
	}

	// Remove the subscription from the slice
	for i, sub := range subs {
		if sub == subscription {
			eb.subscriptions[taskID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	// Clean up empty subscription list
	if len(eb.subscriptions[taskID]) == 0 {
		delete(eb.subscriptions, taskID)
	}
}

// PublishToTask publishes an event to all subscribers of a task
func (eb *EventBus) PublishToTask(taskID string, response types.StreamResponse) {
	eb.mu.RLock()
	subs, exists := eb.subscriptions[taskID]
	if !exists {
		eb.mu.RUnlock()
		return
	}

	// Copy subscriptions to avoid holding lock during delivery
	subscriptions := make([]*TaskSubscription, len(subs))
	copy(subscriptions, subs)
	eb.mu.RUnlock()

	// Deliver to all subscriptions (non-blocking)
	for _, sub := range subscriptions {
		select {
		case sub.events <- response:
		case <-time.After(100 * time.Millisecond):
			// Subscription is slow or closed, remove it
			eb.Unsubscribe(taskID, sub)
			sub.Close()
		}
	}
}

// CleanupTask removes all subscriptions for a task
func (eb *EventBus) CleanupTask(taskID string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs, exists := eb.subscriptions[taskID]
	if !exists {
		return
	}

	// Close all subscriptions
	for _, sub := range subs {
		sub.Close()
	}

	// Remove from map
	delete(eb.subscriptions, taskID)
}

// Close shuts down the event bus and closes all subscriptions
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.closing = true

	// Close all subscriptions
	for taskID, subs := range eb.subscriptions {
		for _, sub := range subs {
			sub.Close()
		}
		delete(eb.subscriptions, taskID)
	}
}

// TaskSubscription represents a subscription to task events
type TaskSubscription struct {
	taskID       string
	events       chan types.StreamResponse
	eventBus     *EventBus
	subscribedAt time.Time
	closed       bool
	closeOnce    sync.Once
	mu           sync.RWMutex
}

// NewTaskSubscription creates a new task subscription
func NewTaskSubscription(taskID string, eventBus *EventBus) *TaskSubscription {
	subscription := &TaskSubscription{
		taskID:       taskID,
		events:       make(chan types.StreamResponse, 10), // Buffered channel
		eventBus:     eventBus,
		subscribedAt: time.Now(),
	}

	// Register with event bus
	eventBus.Subscribe(taskID, subscription)

	return subscription
}

// Events returns the channel for receiving stream responses
func (ts *TaskSubscription) Events() <-chan types.StreamResponse {
	return ts.events
}

// TaskID returns the task ID this subscription is for
func (ts *TaskSubscription) TaskID() string {
	return ts.taskID
}

// SubscribedAt returns when this subscription was created
func (ts *TaskSubscription) SubscribedAt() time.Time {
	return ts.subscribedAt
}

// IsClosed returns whether the subscription is closed
func (ts *TaskSubscription) IsClosed() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.closed
}

// Close closes the subscription
func (ts *TaskSubscription) Close() {
	ts.closeOnce.Do(func() {
		ts.mu.Lock()
		ts.closed = true
		ts.mu.Unlock()

		// Unregister from event bus
		ts.eventBus.Unsubscribe(ts.taskID, ts)

		// Close the channel
		close(ts.events)
	})
}
