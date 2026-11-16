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
	"fmt"
	"testing"
	"time"

	"seata.apache.org/seata-go/v2/pkg/metrics"
)

func BenchmarkRegistryCounterLookup(b *testing.B) {
	registry := NewRegistry()
	id := metrics.Id{
		Name: "bench.lookup",
		Tags: []metrics.Tag{
			{Key: metrics.TagKeyRole, Value: metrics.RoleValueTM},
			{Key: metrics.TagKeyStatus, Value: metrics.StatusValueActive},
		},
	}
	registry.Counter(id)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Counter(id)
	}
}

func BenchmarkCounterAdd(b *testing.B) {
	registry := NewRegistry()
	counter := registry.Counter(metrics.Id{Name: "bench.counter"})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Add(1)
	}
}

func BenchmarkGaugeSet(b *testing.B) {
	registry := NewRegistry()
	gauge := registry.Gauge(metrics.Id{Name: "bench.gauge"})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gauge.Set(float64(i))
	}
}

func BenchmarkTimerRecord(b *testing.B) {
	registry := NewRegistry()
	timer := registry.Timer(metrics.Id{Name: "bench.timer"})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Record(time.Microsecond)
	}
}

func BenchmarkRegistryMeasure(b *testing.B) {
	registry := NewRegistry()
	for i := 0; i < 100; i++ {
		registry.Counter(metrics.Id{Name: fmt.Sprintf("bench.counter.%d", i)}).Add(1)
	}
	registry.Gauge(metrics.Id{Name: "bench.gauge"}).Set(1)
	registry.Timer(metrics.Id{Name: "bench.timer"}).Record(time.Millisecond)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Measure()
	}
}
