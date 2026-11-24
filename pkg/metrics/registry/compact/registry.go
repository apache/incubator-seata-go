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

// Package compact provides the default in-memory metrics registry.
package compact

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"seata.apache.org/seata-go/v2/pkg/metrics"
)

type meter interface {
	measure(timestamp int64) []metrics.Measurement
}

// Registry is an in-memory metrics registry.
//
// The zero value is ready to use. Meters are cached by name and normalized
// tags, so equivalent Ids always resolve to the same meter. An Id can only be
// bound to one meter type; requesting a different type panics.
type Registry struct {
	meters sync.Map // map[string]meter
}

// NewRegistry creates an empty in-memory registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Counter returns the cached counter for id, creating it when necessary.
func (r *Registry) Counter(id metrics.Id) metrics.Counter {
	m := r.resolve(id, func(id metrics.Id) meter {
		return newCounter(id)
	})
	counter, ok := m.(*counter)
	if !ok {
		panicMeterType(id, m, metrics.MeterValueCounter)
	}
	return counter
}

// Gauge returns the cached gauge for id, creating it when necessary.
func (r *Registry) Gauge(id metrics.Id) metrics.Gauge {
	m := r.resolve(id, func(id metrics.Id) meter {
		return newGauge(id)
	})
	gauge, ok := m.(*gauge)
	if !ok {
		panicMeterType(id, m, metrics.MeterValueGauge)
	}
	return gauge
}

// Timer returns the cached timer for id, creating it when necessary.
func (r *Registry) Timer(id metrics.Id) metrics.Timer {
	m := r.resolve(id, func(id metrics.Id) meter {
		return newTimer(id)
	})
	timer, ok := m.(*timer)
	if !ok {
		panicMeterType(id, m, metrics.MeterValueTimer)
	}
	return timer
}

// Measure returns a snapshot of all meters.
//
// All measurements share the same Unix millisecond timestamp. Timers are
// cumulative and are not reset.
func (r *Registry) Measure() []metrics.Measurement {
	timestamp := time.Now().UnixMilli()
	var measurements []metrics.Measurement
	r.meters.Range(func(_, value any) bool {
		measurements = append(measurements, value.(meter).measure(timestamp)...)
		return true
	})
	return measurements
}

func (r *Registry) resolve(id metrics.Id, create func(metrics.Id) meter) meter {
	canonical := canonicalId(id)
	key := meterKey(canonical)
	if existing, ok := r.meters.Load(key); ok {
		return existing.(meter)
	}

	candidate := create(canonical)
	actual, loaded := r.meters.LoadOrStore(key, candidate)
	if loaded {
		return actual.(meter)
	}
	return candidate
}

func canonicalId(id metrics.Id) metrics.Id {
	if len(id.Tags) == 0 {
		return metrics.Id{Name: id.Name}
	}

	tags := make([]metrics.Tag, len(id.Tags))
	copy(tags, id.Tags)
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i].Key < tags[j].Key
	})

	deduped := tags[:0]
	for _, tag := range tags {
		if n := len(deduped); n > 0 && deduped[n-1].Key == tag.Key {
			deduped[n-1] = tag
			continue
		}
		deduped = append(deduped, tag)
	}

	return metrics.Id{Name: id.Name, Tags: deduped}
}

func meterKey(id metrics.Id) string {
	var builder strings.Builder
	builder.Grow(len(id.Name) + len(id.Tags)*16)
	appendKeyPart(&builder, id.Name)
	for _, tag := range id.Tags {
		appendKeyPart(&builder, tag.Key)
		appendKeyPart(&builder, tag.Value)
	}
	return builder.String()
}

func appendKeyPart(builder *strings.Builder, value string) {
	builder.WriteString(strconv.Itoa(len(value)))
	builder.WriteByte(':')
	builder.WriteString(value)
	builder.WriteByte(';')
}

func panicMeterType(id metrics.Id, existing meter, requested string) {
	panic(fmt.Sprintf(
		"metrics: id %q is already registered as %s, cannot be used as %s",
		id.Name, meterKind(existing), requested,
	))
}

func meterKind(m meter) string {
	switch m.(type) {
	case *counter:
		return metrics.MeterValueCounter
	case *gauge:
		return metrics.MeterValueGauge
	case *timer:
		return metrics.MeterValueTimer
	default:
		return fmt.Sprintf("%T", m)
	}
}
