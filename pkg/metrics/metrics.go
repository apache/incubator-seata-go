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

// Package metrics defines the metrics abstraction used by seata-go.
//
// The package contains interfaces and shared definitions; concrete registries
// live in subpackages such as registry/compact. Callers should resolve a meter
// once during initialization and reuse it on hot paths. Implementations must
// be safe for concurrent use.
//
// Timers are cumulative: Registry.Measure does not reset them. This differs
// from Java's compact registry, which resets timer statistics on every
// measure; count and total are suitable for rate/increase queries, while max
// and average are lifetime values. There is no package-level mutable registry;
// use NoopRegistry when metrics are disabled.
package metrics

import "time"

// Tag is a key/value pair attached to a meter.
type Tag struct {
	Key   string
	Value string
}

// Id identifies a meter by name and tags.
//
// Tags are treated as an unordered set by registries. If a tag key appears
// more than once, the last value wins. After an Id is passed to a Registry,
// its Tags must be treated as read-only.
type Id struct {
	Name string
	Tags []Tag
}

// Counter is a cumulative meter.
//
// A negative delta decreases the counter, matching Java's Counter.decrease.
// Use Gauge for values that are expected to rise and fall. Exporters that map
// Counter to a backend with monotonic counters, such as Prometheus, must
// reject or clamp negative deltas.
type Counter interface {
	Add(float64)
}

// Gauge is a meter exposing a single current value.
//
// Callers are responsible for keeping Add and Sub balanced. This is a
// write-based gauge; unlike Java's supplier-based CompactGauge, the value is
// stored by the meter rather than sampled during Measure.
type Gauge interface {
	Set(float64)
	Add(float64)
	Sub(float64)
}

// Timer records elapsed durations.
//
// Timers are cumulative and are not reset by Registry.Measure. Durations are
// exported in seconds, and timestamps use Unix milliseconds. Java's compact
// timer resets its statistics on each measure, so Go's average and max are
// lifetime values rather than per-window values. Negative durations are
// ignored.
type Timer interface {
	Record(time.Duration)
}

// Measurement is a single value produced by a meter.
//
// Timestamp is the measurement time in Unix milliseconds. Measurement.Id and
// its Tags are owned by the meter and must be treated as read-only.
type Measurement struct {
	Id        Id
	Timestamp int64
	Value     float64
}

// Registry creates, caches, and measures meters.
type Registry interface {
	Counter(Id) Counter
	Gauge(Id) Gauge
	Timer(Id) Timer
	Measure() []Measurement
}
