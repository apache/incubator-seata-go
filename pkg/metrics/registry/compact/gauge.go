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

type gauge struct {
	id    metrics.Id
	value atomic.Uint64
}

func newGauge(id metrics.Id) *gauge {
	return &gauge{id: id}
}

// Set replaces the current gauge value.
func (g *gauge) Set(value float64) {
	g.value.Store(math.Float64bits(value))
}

// Add increases the gauge by delta.
func (g *gauge) Add(delta float64) {
	for {
		oldBits := g.value.Load()
		newBits := math.Float64bits(math.Float64frombits(oldBits) + delta)
		if g.value.CompareAndSwap(oldBits, newBits) {
			return
		}
	}
}

// Sub decreases the gauge by delta.
func (g *gauge) Sub(delta float64) {
	g.Add(-delta)
}

func (g *gauge) measure(timestamp int64) []metrics.Measurement {
	return []metrics.Measurement{{
		Id:        g.id,
		Timestamp: timestamp,
		Value:     math.Float64frombits(g.value.Load()),
	}}
}
