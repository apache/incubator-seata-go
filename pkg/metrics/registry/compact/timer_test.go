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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/metrics"
)

func TestTimerMeasure(t *testing.T) {
	registry := NewRegistry()
	id := metrics.Id{
		Name: "timer",
		Tags: []metrics.Tag{{Key: metrics.TagKeyStatus, Value: metrics.StatusValueCommitted}},
	}
	timer := registry.Timer(id)

	timer.Record(1500 * time.Millisecond)

	measurements := registry.Measure()
	require.Len(t, measurements, 4)
	require.Equal(t, metrics.StatisticValueCount, mustTagValue(t, measurements[0].Id, metrics.TagKeyStatistic))
	require.Equal(t, metrics.StatisticValueTotal, mustTagValue(t, measurements[1].Id, metrics.TagKeyStatistic))
	require.Equal(t, metrics.StatisticValueMax, mustTagValue(t, measurements[2].Id, metrics.TagKeyStatistic))
	require.Equal(t, metrics.StatisticValueAverage, mustTagValue(t, measurements[3].Id, metrics.TagKeyStatistic))

	require.Equal(t, float64(1), measurements[0].Value)
	require.Equal(t, float64(1.5), measurements[1].Value)
	require.Equal(t, float64(1.5), measurements[2].Value)
	require.Equal(t, float64(1.5), measurements[3].Value)
	require.Equal(t, metrics.StatusValueCommitted, mustTagValue(t, measurements[0].Id, metrics.TagKeyStatus))
}

func TestTimerAccumulatesAcrossMeasure(t *testing.T) {
	registry := NewRegistry()
	id := metrics.Id{Name: "timer.cumulative"}
	timer := registry.Timer(id)

	timer.Record(1500 * time.Millisecond)
	first := registry.Measure()
	second := registry.Measure()

	require.Equal(t, float64(1), measurementBy(t, first, id.Name, metrics.StatisticValueCount).Value)
	require.Equal(t, float64(1), measurementBy(t, second, id.Name, metrics.StatisticValueCount).Value)

	registry.Timer(id).Record(500 * time.Millisecond)
	third := registry.Measure()

	require.Equal(t, float64(2), measurementBy(t, third, id.Name, metrics.StatisticValueCount).Value)
	require.Equal(t, float64(2), measurementBy(t, third, id.Name, metrics.StatisticValueTotal).Value)
	require.Equal(t, float64(1.5), measurementBy(t, third, id.Name, metrics.StatisticValueMax).Value)
	require.Equal(t, float64(1), measurementBy(t, third, id.Name, metrics.StatisticValueAverage).Value)
}

func TestTimerIgnoresNegativeDurations(t *testing.T) {
	registry := NewRegistry()
	id := metrics.Id{Name: "timer.negative"}
	timer := registry.Timer(id)

	timer.Record(-time.Second)
	timer.Record(0)

	measurements := registry.Measure()
	require.Equal(t, float64(1), measurementBy(t, measurements, id.Name, metrics.StatisticValueCount).Value)
	require.Equal(t, float64(0), measurementBy(t, measurements, id.Name, metrics.StatisticValueTotal).Value)
	require.Equal(t, float64(0), measurementBy(t, measurements, id.Name, metrics.StatisticValueMax).Value)
	require.Equal(t, float64(0), measurementBy(t, measurements, id.Name, metrics.StatisticValueAverage).Value)
}

func mustTagValue(t *testing.T, id metrics.Id, key string) string {
	t.Helper()
	value, ok := tagValue(id, key)
	require.True(t, ok, "tag %q not found in %+v", key, id.Tags)
	return value
}
