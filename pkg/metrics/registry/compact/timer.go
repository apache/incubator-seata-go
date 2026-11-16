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
	"sync/atomic"
	"time"

	"seata.apache.org/seata-go/v2/pkg/metrics"
)

const nanosPerSecond = float64(time.Second)

type timer struct {
	countId   metrics.Id
	totalId   metrics.Id
	maxId     metrics.Id
	averageId metrics.Id

	count      atomic.Int64
	totalNanos atomic.Int64
	maxNanos   atomic.Int64
}

func newTimer(id metrics.Id) *timer {
	return &timer{
		countId:   withTag(id, metrics.TagKeyStatistic, metrics.StatisticValueCount),
		totalId:   withTag(id, metrics.TagKeyStatistic, metrics.StatisticValueTotal),
		maxId:     withTag(id, metrics.TagKeyStatistic, metrics.StatisticValueMax),
		averageId: withTag(id, metrics.TagKeyStatistic, metrics.StatisticValueAverage),
	}
}

// Record adds a duration sample. Negative durations are ignored.
func (t *timer) Record(duration time.Duration) {
	if duration < 0 {
		return
	}

	nanos := int64(duration)
	t.count.Add(1)
	t.totalNanos.Add(nanos)
	for {
		max := t.maxNanos.Load()
		if nanos <= max || t.maxNanos.CompareAndSwap(max, nanos) {
			return
		}
	}
}

func (t *timer) measure(timestamp int64) []metrics.Measurement {
	count := t.count.Load()
	total := t.totalNanos.Load()
	max := t.maxNanos.Load()

	totalSeconds := float64(total) / nanosPerSecond
	averageSeconds := 0.0
	if count > 0 {
		averageSeconds = totalSeconds / float64(count)
	}

	return []metrics.Measurement{
		{Id: t.countId, Timestamp: timestamp, Value: float64(count)},
		{Id: t.totalId, Timestamp: timestamp, Value: totalSeconds},
		{Id: t.maxId, Timestamp: timestamp, Value: float64(max) / nanosPerSecond},
		{Id: t.averageId, Timestamp: timestamp, Value: averageSeconds},
	}
}

func withTag(id metrics.Id, key, value string) metrics.Id {
	tags := make([]metrics.Tag, 0, len(id.Tags)+1)
	tags = append(tags, id.Tags...)
	tags = append(tags, metrics.Tag{Key: key, Value: value})
	return canonicalId(metrics.Id{Name: id.Name, Tags: tags})
}
