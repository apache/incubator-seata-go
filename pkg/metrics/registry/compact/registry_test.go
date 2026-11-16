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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/metrics"
)

func TestRegistryReturnsSameMeterForEquivalentIds(t *testing.T) {
	registry := NewRegistry()

	first := registry.Counter(metrics.Id{
		Name: "orders",
		Tags: []metrics.Tag{
			{Key: metrics.TagKeyStatus, Value: metrics.StatusValueActive},
			{Key: metrics.TagKeyRole, Value: metrics.RoleValueTM},
		},
	})
	second := registry.Counter(metrics.Id{
		Name: "orders",
		Tags: []metrics.Tag{
			{Key: metrics.TagKeyRole, Value: metrics.RoleValueTM},
			{Key: metrics.TagKeyStatus, Value: metrics.StatusValueActive},
		},
	})

	require.Same(t, first, second)

	first.Add(2)
	require.Equal(t, float64(2), measurementBy(t, registry.Measure(), "orders", "").Value)
}

func TestRegistryZeroValue(t *testing.T) {
	var registry Registry

	require.Empty(t, registry.Measure())

	registry.Counter(metrics.Id{Name: "zero"}).Add(1)
	require.Equal(t, float64(1), measurementBy(t, registry.Measure(), "zero", "").Value)
}

func TestRegistryLastDuplicateTagWins(t *testing.T) {
	registry := NewRegistry()

	first := registry.Counter(metrics.Id{
		Name: "duplicate",
		Tags: []metrics.Tag{
			{Key: "zone", Value: "a"},
			{Key: "role", Value: metrics.RoleValueRM},
			{Key: "zone", Value: "b"},
		},
	})
	second := registry.Counter(metrics.Id{
		Name: "duplicate",
		Tags: []metrics.Tag{
			{Key: "role", Value: metrics.RoleValueRM},
			{Key: "zone", Value: "b"},
		},
	})

	require.Same(t, first, second)
}

func TestRegistryDistinguishesTagKeys(t *testing.T) {
	registry := NewRegistry()

	zone := registry.Counter(metrics.Id{
		Name: "tag.keys",
		Tags: []metrics.Tag{{Key: "zone", Value: "a"}},
	})
	region := registry.Counter(metrics.Id{
		Name: "tag.keys",
		Tags: []metrics.Tag{{Key: "region", Value: "a"}},
	})
	require.NotSame(t, zone, region)

	zone.Add(1)
	region.Add(2)

	measurements := registry.Measure()
	require.Len(t, measurements, 2)

	var zoneValue, regionValue float64
	for _, measurement := range measurements {
		if _, ok := tagValue(measurement.Id, "zone"); ok {
			zoneValue = measurement.Value
		}
		if _, ok := tagValue(measurement.Id, "region"); ok {
			regionValue = measurement.Value
		}
	}
	require.Equal(t, float64(1), zoneValue)
	require.Equal(t, float64(2), regionValue)
}

func TestRegistryCopiesTags(t *testing.T) {
	registry := NewRegistry()
	tags := []metrics.Tag{{Key: "zone", Value: "a"}}

	first := registry.Counter(metrics.Id{Name: "copy", Tags: tags})
	tags[0].Value = "changed"

	second := registry.Counter(metrics.Id{
		Name: "copy",
		Tags: []metrics.Tag{{Key: "zone", Value: "a"}},
	})
	third := registry.Counter(metrics.Id{
		Name: "copy",
		Tags: []metrics.Tag{{Key: "zone", Value: "changed"}},
	})

	require.Same(t, first, second)
	require.NotEqual(t, first, third)
}

func TestRegistryPanicsOnMeterTypeConflict(t *testing.T) {
	registry := NewRegistry()
	id := metrics.Id{Name: "conflict"}

	registry.Counter(id)

	require.PanicsWithValue(t,
		`metrics: id "conflict" is already registered as counter, cannot be used as gauge`,
		func() {
			registry.Gauge(id)
		},
	)
	require.PanicsWithValue(t,
		`metrics: id "conflict" is already registered as counter, cannot be used as timer`,
		func() {
			registry.Timer(id)
		},
	)
}

func TestRegistryMeasure(t *testing.T) {
	registry := NewRegistry()

	counterId := metrics.Id{
		Name: "orders.total",
		Tags: []metrics.Tag{{Key: metrics.TagKeyStatus, Value: metrics.StatusValueCommitted}},
	}
	gaugeId := metrics.Id{
		Name: "orders.active",
		Tags: []metrics.Tag{{Key: metrics.TagKeyStatus, Value: metrics.StatusValueActive}},
	}
	timerId := metrics.Id{
		Name: "orders.duration",
		Tags: []metrics.Tag{{Key: metrics.TagKeyStatus, Value: metrics.StatusValueCommitted}},
	}

	registry.Counter(counterId).Add(3)
	registry.Gauge(gaugeId).Set(7)
	timer := registry.Timer(timerId)
	timer.Record(2 * time.Second)
	timer.Record(4 * time.Second)

	before := time.Now().UnixMilli()
	measurements := registry.Measure()
	after := time.Now().UnixMilli()

	require.Len(t, measurements, 6)

	counter := measurementBy(t, measurements, "orders.total", "")
	require.Equal(t, float64(3), counter.Value)

	gauge := measurementBy(t, measurements, "orders.active", "")
	require.Equal(t, float64(7), gauge.Value)

	count := measurementBy(t, measurements, "orders.duration", metrics.StatisticValueCount)
	total := measurementBy(t, measurements, "orders.duration", metrics.StatisticValueTotal)
	max := measurementBy(t, measurements, "orders.duration", metrics.StatisticValueMax)
	average := measurementBy(t, measurements, "orders.duration", metrics.StatisticValueAverage)

	require.Equal(t, float64(2), count.Value)
	require.Equal(t, float64(6), total.Value)
	require.Equal(t, float64(4), max.Value)
	require.Equal(t, float64(3), average.Value)

	for _, measurement := range measurements {
		require.Equal(t, measurements[0].Timestamp, measurement.Timestamp)
		require.GreaterOrEqual(t, measurement.Timestamp, before)
		require.LessOrEqual(t, measurement.Timestamp, after)
	}
}

func TestRegistryMeasureWhileWritersAreActive(t *testing.T) {
	registry := NewRegistry()
	counterId := metrics.Id{Name: "concurrent.counter"}
	gaugeId := metrics.Id{Name: "concurrent.gauge"}
	timerId := metrics.Id{Name: "concurrent.timer"}

	counter := registry.Counter(counterId)
	gauge := registry.Gauge(gaugeId)
	timer := registry.Timer(timerId)

	const (
		workers    = 8
		iterations = 500
	)

	var waitGroup sync.WaitGroup
	for i := 0; i < workers; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for j := 0; j < iterations; j++ {
				counter.Add(1)
				gauge.Add(1)
				gauge.Sub(0.5)
				gauge.Set(float64(j))
				timer.Record(time.Microsecond)
				registry.Counter(counterId)
				registry.Measure()
			}
		}()
	}
	waitGroup.Wait()

	measurements := registry.Measure()
	require.Equal(t, float64(workers*iterations), measurementBy(t, measurements, "concurrent.counter", "").Value)
	require.Equal(t, float64(workers*iterations),
		measurementBy(t, measurements, "concurrent.timer", metrics.StatisticValueCount).Value)

	gaugeValue := measurementBy(t, measurements, "concurrent.gauge", "").Value
	require.False(t, math.IsNaN(gaugeValue))
	require.False(t, math.IsInf(gaugeValue, 0))
}

func measurementBy(t *testing.T, measurements []metrics.Measurement, name, statistic string) metrics.Measurement {
	t.Helper()

	for _, measurement := range measurements {
		if measurement.Id.Name != name {
			continue
		}
		value, _ := tagValue(measurement.Id, metrics.TagKeyStatistic)
		if value == statistic {
			return measurement
		}
	}

	t.Fatalf("measurement not found: name=%s statistic=%s", name, statistic)
	return metrics.Measurement{}
}

func tagValue(id metrics.Id, key string) (string, bool) {
	for _, tag := range id.Tags {
		if tag.Key == key {
			return tag.Value, true
		}
	}
	return "", false
}
