/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tccbench

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestHistogramBucketBounds(t *testing.T) {
	for _, ns := range []uint64{0, 1, 63, 64, 65, 127, 128, 129, 1024, 1_000_000, 30_000_000_000} {
		index := bucketIndex(ns)
		upper := bucketUpper(index)
		if upper < ns {
			t.Fatalf("bucket upper bound %d is below value %d", upper, ns)
		}
		if ns >= 64 && float64(upper-ns)/float64(ns) > 1.0/64.0 {
			t.Fatalf("bucket error exceeds 1/64 for %d ns: upper=%d", ns, upper)
		}
	}
}

func TestHistogramQuantiles(t *testing.T) {
	var histogram latencyHistogram
	for i := 1; i <= 100; i++ {
		histogram.add(time.Duration(i) * time.Millisecond)
	}
	if got := histogram.quantile(.50); got < 50 || got > 51 {
		t.Fatalf("P50 = %.3f ms, want approximately 50ms", got)
	}
	if got := histogram.quantile(.95); got < 95 || got > 96 {
		t.Fatalf("P95 = %.3f ms, want approximately 95ms", got)
	}
	if got := histogram.quantile(.99); got < 99 || got > 100 {
		t.Fatalf("P99 = %.3f ms, want approximately 99ms", got)
	}
}

func TestMeasureExactOperationCountWithIndependentWorkers(t *testing.T) {
	var calls atomic.Uint64
	result := measure(8, 0, 1000, func() error {
		calls.Add(1)
		return nil
	})
	if got := calls.Load(); got != 1000 {
		t.Fatalf("operation called %d times, want 1000", got)
	}
	if result.Successes != 1000 || result.Failures != 0 || result.AllLatency.Count != 1000 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestMeasureCountsFailuresAndSamples(t *testing.T) {
	var calls atomic.Uint64
	wantErr := errors.New("expected failure")
	result := measure(4, 0, 20, func() error {
		if calls.Add(1)%2 == 0 {
			return fmt.Errorf("%w", wantErr)
		}
		return nil
	})
	if result.Successes != 10 || result.Failures != 10 || result.FailureRate != 0.5 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.ErrorSamples) != 8 {
		t.Fatalf("captured %d error samples, want bounded 8", len(result.ErrorSamples))
	}
}

func TestResourceSummaryPreservesEndAndPeak(t *testing.T) {
	start := resourceSnapshot{cpuUsec: 100, memory: 10, rss: 20}
	end := resourceSnapshot{cpuUsec: 1100100, memory: 12, rss: 22, goroutines: 3}
	peak := resourceSnapshot{memory: 30, rss: 40, goroutines: 5}
	got := summarizeResources(start, end, peak)
	if got.CgroupMemoryEndBytes != 12 || got.CgroupMemoryPeakBytes != 30 || got.ProcessRSSEndBytes != 22 || got.ProcessRSSPeakBytes != 40 || got.ProcessGoroutinePeak != 5 {
		t.Fatalf("incorrect end or peak: %+v", got)
	}
	if got.CgroupCPUUsageCPUSeconds == nil || *got.CgroupCPUUsageCPUSeconds != 1.1 {
		t.Fatalf("incorrect CPU usage: %+v", got.CgroupCPUUsageCPUSeconds)
	}
	end.cpuUsec = 0
	if got := summarizeResources(start, end, peak); got.CgroupCPUUsageCPUSeconds != nil {
		t.Fatalf("unavailable CPU usage must be null: %+v", got)
	}
}

func TestWriteJSONDoesNotOverwrite(t *testing.T) {
	path := t.TempDir() + "/result.json"
	if err := writeJSON(path, "first"); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(path, "second"); err == nil {
		t.Fatal("expected existing result to be preserved")
	}
}
