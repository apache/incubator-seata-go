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
// The package contains interfaces only; concrete registries live in
// subpackages such as registry/compact. Callers should resolve a meter once
// during initialization and reuse it on hot paths. Implementations must be
// safe for concurrent use.
//
// Timers are cumulative: Registry.Measure does not reset them. There is no
// package-level mutable registry; use NoopRegistry when metrics are disabled.
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
// more than once, the last value wins.
type Id struct {
	Name string
	Tags []Tag
}

// Counter is a cumulative meter.
type Counter interface {
	Add(float64)
}

// Gauge is a meter exposing a single current value.
type Gauge interface {
	Set(float64)
	Add(float64)
	Sub(float64)
}

// Timer records elapsed durations.
//
// Timers are cumulative and are not reset by Registry.Measure. Negative
// durations are ignored.
type Timer interface {
	Record(time.Duration)
}

// Measurement is a single value produced by a meter.
//
// Timestamp is the measurement time in Unix milliseconds.
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
