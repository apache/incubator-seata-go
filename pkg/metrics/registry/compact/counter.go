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

package compact

import (
	"math"
	"sync/atomic"

	"seata.apache.org/seata-go/v2/pkg/metrics"
)

type counter struct {
	id    metrics.Id
	value atomic.Uint64
}

func newCounter(id metrics.Id) *counter {
	return &counter{id: id}
}

// Add increases the counter by delta. A negative delta decreases it, matching
// Java's Counter.decrease; use Gauge for values that rise and fall.
func (c *counter) Add(delta float64) {
	for {
		oldBits := c.value.Load()
		newBits := math.Float64bits(math.Float64frombits(oldBits) + delta)
		if c.value.CompareAndSwap(oldBits, newBits) {
			return
		}
	}
}

func (c *counter) measure(timestamp int64) []metrics.Measurement {
	return []metrics.Measurement{{
		Id:        c.id,
		Timestamp: timestamp,
		Value:     math.Float64frombits(c.value.Load()),
	}}
}
