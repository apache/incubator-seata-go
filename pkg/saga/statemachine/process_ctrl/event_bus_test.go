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

package process_ctrl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type acceptingEventConsumer struct{}

func (acceptingEventConsumer) Accept(Event) bool {
	return true
}

func (acceptingEventConsumer) Process(context.Context, Event) error {
	return nil
}

func TestAsyncEventBus_OfferErrors(t *testing.T) {
	event := struct{}{}

	t.Run("no matching consumer", func(t *testing.T) {
		bus := NewAsyncEventBus(context.Background(), 0, 0)

		accepted, err := bus.Offer(context.Background(), event)

		assert.False(t, accepted)
		assert.EqualError(t, err, "cannot find event handler by type: struct {}")
	})

	t.Run("event is not a process context", func(t *testing.T) {
		bus := NewAsyncEventBus(context.Background(), 0, 0)
		bus.RegisterEventConsumer(acceptingEventConsumer{})

		accepted, err := bus.Offer(context.Background(), event)

		assert.False(t, accepted)
		assert.EqualError(t, err, "event struct {} is illegal, required process_ctrl.ProcessContext")
	})
}
