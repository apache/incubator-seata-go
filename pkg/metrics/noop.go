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

package metrics

import "time"

// NoopRegistry is a Registry that discards all measurements.
//
// It is safe to use without nil checks and returns an empty slice from
// Measure.
var NoopRegistry Registry = noopRegistry{}

type noopRegistry struct{}

func (noopRegistry) Counter(Id) Counter {
	return noopCounter{}
}

func (noopRegistry) Gauge(Id) Gauge {
	return noopGauge{}
}

func (noopRegistry) Timer(Id) Timer {
	return noopTimer{}
}

func (noopRegistry) Measure() []Measurement {
	return nil
}

type noopCounter struct{}

func (noopCounter) Add(float64) {}

type noopGauge struct{}

func (noopGauge) Set(float64) {}

func (noopGauge) Add(float64) {}

func (noopGauge) Sub(float64) {}

type noopTimer struct{}

func (noopTimer) Record(time.Duration) {}
