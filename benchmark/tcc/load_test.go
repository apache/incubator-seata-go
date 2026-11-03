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
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/bits"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	configPathFlag  = flag.String("tcc.config", os.Getenv(configEnv), "Seata client configuration")
	logPathFlag     = flag.String("tcc.log", "/tmp/seata-tcc-benchmark.log", "SDK WARN/ERROR log")
	loadFlag        = flag.Bool("tcc.load", false, "Enable the timed integration load test")
	concurrencyFlag = flag.Int("tcc.concurrency", 1, "Number of transaction workers, independent of GOMAXPROCS")
	gomaxprocsFlag  = flag.Int("tcc.gomaxprocs", runtime.GOMAXPROCS(0), "Go scheduler parallelism, independent of transaction workers")
	warmupFlag      = flag.Duration("tcc.warmup", 10*time.Second, "Warm-up launch window")
	durationFlag    = flag.Duration("tcc.duration", 30*time.Second, "Measured launch window")
	outputFlag      = flag.String("tcc.output", "", "JSON result path")
	payloadFlag     = flag.Int("tcc.payload-bytes", 0, "ASCII payload size; excludes protocol/metadata overhead")

	requestParameters struct {
		Payload string `tccParam:"payload"`
	}
)

func validateLoadFlags() error {
	if *concurrencyFlag < 1 || *concurrencyFlag > 256 {
		return fmt.Errorf("tcc.concurrency must be between 1 and 256")
	}
	if *gomaxprocsFlag < 1 || *gomaxprocsFlag > 256 {
		return fmt.Errorf("tcc.gomaxprocs must be between 1 and 256")
	}
	if *warmupFlag < 0 || *durationFlag <= 0 {
		return fmt.Errorf("warmup must be nonnegative and duration must be positive")
	}
	if *payloadFlag < 0 || *payloadFlag > 1<<20 {
		return fmt.Errorf("payload-bytes must be between 0 and 1048576")
	}
	if *outputFlag != "" {
		if info, err := os.Stat(filepath.Dir(*outputFlag)); err != nil || !info.IsDir() {
			return fmt.Errorf("result directory does not exist: %s", filepath.Dir(*outputFlag))
		}
	}
	return nil
}

// Fixed memory, with 64 subdivisions per power of two. Quantiles return the
// bucket upper bound, at most 1/64 above an observation (exact below 64 ns).
// Memory use does not grow with the number of completed transactions.
type latencyHistogram struct {
	buckets [4096]uint64
	count   uint64
	sumNS   uint64
	minNS   uint64
	maxNS   uint64
}

func bucketIndex(ns uint64) int {
	if ns < 64 {
		return int(ns)
	}
	exponent := bits.Len64(ns) - 1
	return 64 + (exponent-6)*64 + int((ns-(uint64(1)<<exponent))>>(exponent-6))
}

func bucketUpper(index int) uint64 {
	if index < 64 {
		return uint64(index)
	}
	exponent := 6 + (index-64)/64
	subdivision := uint64((index - 64) % 64)
	return (uint64(1) << exponent) + ((subdivision + 1) << (exponent - 6)) - 1
}

func (h *latencyHistogram) add(d time.Duration) {
	ns := uint64(d)
	h.buckets[bucketIndex(ns)]++
	if h.count == 0 || ns < h.minNS {
		h.minNS = ns
	}
	if ns > h.maxNS {
		h.maxNS = ns
	}
	h.count++
	h.sumNS += ns
}

func (h *latencyHistogram) merge(other *latencyHistogram) {
	if other.count == 0 {
		return
	}
	if h.count == 0 || other.minNS < h.minNS {
		h.minNS = other.minNS
	}
	if other.maxNS > h.maxNS {
		h.maxNS = other.maxNS
	}
	h.count += other.count
	h.sumNS += other.sumNS
	for i, count := range other.buckets {
		h.buckets[i] += count
	}
}

func (h *latencyHistogram) quantile(q float64) float64 {
	if h.count == 0 {
		return 0
	}
	target := uint64(math.Ceil(float64(h.count) * q))
	var total uint64
	for i, count := range h.buckets {
		total += count
		if count != 0 && total >= target {
			return float64(bucketUpper(i)) / 1e6
		}
	}
	return float64(h.maxNS) / 1e6
}

type histogramBin struct {
	UpperNS uint64 `json:"upper_ns"`
	Count   uint64 `json:"count"`
}

type latencySummary struct {
	Count     uint64         `json:"count"`
	MeanMS    float64        `json:"mean_ms"`
	MinMS     float64        `json:"min_ms"`
	P50MS     float64        `json:"p50_ms"`
	P95MS     float64        `json:"p95_ms"`
	P99MS     float64        `json:"p99_ms"`
	MaxMS     float64        `json:"max_ms"`
	Histogram []histogramBin `json:"histogram"`
}

func (h *latencyHistogram) summary() latencySummary {
	summary := latencySummary{Count: h.count, MinMS: float64(h.minNS) / 1e6,
		MaxMS: float64(h.maxNS) / 1e6, P50MS: h.quantile(.50), P95MS: h.quantile(.95), P99MS: h.quantile(.99)}
	if h.count > 0 {
		summary.MeanMS = float64(h.sumNS) / float64(h.count) / 1e6
	}
	for i, count := range h.buckets {
		if count != 0 {
			summary.Histogram = append(summary.Histogram, histogramBin{UpperNS: bucketUpper(i), Count: count})
		}
	}
	return summary
}

type workerStats struct {
	successes uint64
	failures  uint64
	all       latencyHistogram
	success   latencyHistogram
	errors    map[string]uint64
	samples   []string
}

type phaseResult struct {
	StartedAt       time.Time         `json:"started_at"`
	FinishedAt      time.Time         `json:"finished_at"`
	LaunchSeconds   float64           `json:"launch_seconds"`
	ElapsedSeconds  float64           `json:"elapsed_seconds"`
	DrainSeconds    float64           `json:"drain_seconds"`
	Attempts        uint64            `json:"attempts"`
	Successes       uint64            `json:"successes"`
	Failures        uint64            `json:"failures"`
	FailureRate     float64           `json:"failure_rate"`
	SuccessfulTPS   float64           `json:"successful_tps"`
	AllLatency      latencySummary    `json:"latency_all"`
	SuccessLatency  latencySummary    `json:"latency_success"`
	ErrorStages     map[string]uint64 `json:"error_stages"`
	ErrorSamples    []string          `json:"error_samples"`
	TryCallbacks    uint64            `json:"try_callbacks"`
	CommitCallbacks uint64            `json:"confirm_callbacks"`
	CancelCallbacks uint64            `json:"cancel_callbacks"`
	Resources       resourceUsage     `json:"resources"`
}

// A closed loop with exactly concurrency workers. Stop launching at the end of
// the window, then wait for every in-flight operation. Count that drain time in
// the throughput denominator, so unfinished work cannot inflate the result.
// For Go's Benchmark API, duration=0 instead executes exactly operations calls.
func measure(concurrency int, duration time.Duration, operations int64, operation func() error) phaseResult {
	workers := make([]workerStats, concurrency)
	var ready, finished sync.WaitGroup
	ready.Add(concurrency)
	finished.Add(concurrency)
	gate := make(chan struct{})
	var deadline time.Time
	var next atomic.Int64
	for i := range workers {
		go func(stats *workerStats) {
			defer finished.Done()
			stats.errors = make(map[string]uint64)
			ready.Done()
			<-gate
			for {
				start := time.Now()
				if duration > 0 {
					if !start.Before(deadline) {
						return
					}
				} else {
					operationNumber := next.Add(1)
					if operationNumber > operations {
						return
					}
				}
				err := operation()
				elapsed := time.Since(start)
				stats.all.add(elapsed)
				if err == nil {
					stats.successes++
					stats.success.add(elapsed)
				} else {
					stats.failures++
					stats.errors[failureStage(err)]++
					if len(stats.samples) < 8 {
						stats.samples = append(stats.samples, err.Error())
					}
				}
			}
		}(&workers[i])
	}
	ready.Wait()
	beforeTry, beforeCommit, beforeCancel := preparedCount.Load(), committedCount.Load(), rolledBackCount.Load()
	resourceSampler := startResourceSampler()
	start := time.Now()
	deadline = start.Add(duration)
	close(gate)
	finished.Wait()
	end := time.Now()
	resourceResult := resourceSampler.stop()
	result := phaseResult{StartedAt: start.UTC(), FinishedAt: end.UTC(), LaunchSeconds: duration.Seconds(),
		ElapsedSeconds: end.Sub(start).Seconds(), ErrorStages: make(map[string]uint64),
		TryCallbacks: preparedCount.Load() - beforeTry, CommitCallbacks: committedCount.Load() - beforeCommit,
		CancelCallbacks: rolledBackCount.Load() - beforeCancel, Resources: resourceResult}
	var all, success latencyHistogram
	for i := range workers {
		stats := &workers[i]
		result.Successes += stats.successes
		result.Failures += stats.failures
		all.merge(&stats.all)
		success.merge(&stats.success)
		for stage, count := range stats.errors {
			result.ErrorStages[stage] += count
		}
		for _, sample := range stats.samples {
			if len(result.ErrorSamples) < 8 {
				result.ErrorSamples = append(result.ErrorSamples, sample)
			}
		}
	}
	result.Attempts = result.Successes + result.Failures
	if result.Attempts > 0 {
		result.FailureRate = float64(result.Failures) / float64(result.Attempts)
	}
	if result.ElapsedSeconds > 0 {
		result.SuccessfulTPS = float64(result.Successes) / result.ElapsedSeconds
	}
	if duration > 0 && end.After(deadline) {
		result.DrainSeconds = end.Sub(deadline).Seconds()
	}
	result.AllLatency, result.SuccessLatency = all.summary(), success.summary()
	return result
}

func writeJSON(path string, value interface{}) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".tcc-result-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(value)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Link(file.Name(), path)
}

type resourceUsage struct {
	CgroupCPUUsageCPUSeconds *float64 `json:"cgroup_cpu_usage_cpu_seconds"`
	CgroupMemoryStartBytes   uint64   `json:"cgroup_memory_start_bytes"`
	CgroupMemoryEndBytes     uint64   `json:"cgroup_memory_end_bytes"`
	CgroupMemoryPeakBytes    uint64   `json:"cgroup_memory_peak_bytes"`
	ProcessRSSStartBytes     uint64   `json:"process_rss_start_bytes"`
	ProcessRSSEndBytes       uint64   `json:"process_rss_end_bytes"`
	ProcessRSSPeakBytes      uint64   `json:"process_rss_peak_bytes"`
	ProcessGoroutinePeak     int      `json:"process_goroutine_peak"`
	CgroupCPUMax             string   `json:"cgroup_cpu_max,omitempty"`
	CgroupMemoryMaxBytes     uint64   `json:"cgroup_memory_max_bytes,omitempty"`
	SamplingIntervalMS       int64    `json:"sampling_interval_ms"`
}

type resourceSnapshot struct {
	cpuUsec    uint64
	memory     uint64
	memoryMax  uint64
	rss        uint64
	cpuMax     string
	goroutines int
}

func readUintFile(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	return value
}

func readCgroupCPUUsec() uint64 {
	file, err := os.Open("/sys/fs/cgroup/cpu.stat")
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] == "usage_usec" {
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			return value
		}
	}
	return 0
}

func readProcessRSS() uint64 {
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && fields[0] == "VmRSS:" {
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			return value * 1024
		}
	}
	return 0
}

func snapshotResources() resourceSnapshot {
	cpuMaxBytes, _ := os.ReadFile("/sys/fs/cgroup/cpu.max")
	return resourceSnapshot{cpuUsec: readCgroupCPUUsec(),
		memory:    readUintFile("/sys/fs/cgroup/memory.current"),
		memoryMax: readUintFile("/sys/fs/cgroup/memory.max"),
		rss:       readProcessRSS(), cpuMax: strings.TrimSpace(string(cpuMaxBytes)),
		goroutines: runtime.NumGoroutine()}
}

type resourceRun struct {
	started resourceSnapshot
	current atomic.Pointer[resourceSnapshot]
	stopCh  chan struct{}
	done    chan struct{}
}

func startResourceSampler() *resourceRun {
	run := &resourceRun{started: snapshotResources(), stopCh: make(chan struct{}), done: make(chan struct{})}
	initial := run.started
	run.current.Store(&initial)
	go func() {
		defer close(run.done)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				snapshot := snapshotResources()
				previous := run.current.Load()
				if snapshot.memory < previous.memory {
					snapshot.memory = previous.memory
				}
				if snapshot.rss < previous.rss {
					snapshot.rss = previous.rss
				}
				if snapshot.goroutines < previous.goroutines {
					snapshot.goroutines = previous.goroutines
				}
				run.current.Store(&snapshot)
			case <-run.stopCh:
				return
			}
		}
	}()
	return run
}

func (r *resourceRun) stop() resourceUsage {
	end := snapshotResources()
	close(r.stopCh)
	<-r.done
	return summarizeResources(r.started, end, *r.current.Load())
}

func summarizeResources(start, end, peak resourceSnapshot) resourceUsage {
	if peak.memory < end.memory {
		peak.memory = end.memory
	}
	if peak.rss < end.rss {
		peak.rss = end.rss
	}
	if peak.goroutines < end.goroutines {
		peak.goroutines = end.goroutines
	}
	usage := resourceUsage{CgroupMemoryStartBytes: start.memory, CgroupMemoryEndBytes: end.memory,
		CgroupMemoryPeakBytes: peak.memory, ProcessRSSStartBytes: start.rss,
		ProcessRSSEndBytes: end.rss, ProcessRSSPeakBytes: peak.rss,
		ProcessGoroutinePeak: peak.goroutines, CgroupCPUMax: start.cpuMax,
		CgroupMemoryMaxBytes: start.memoryMax, SamplingIntervalMS: 500}
	if start.cpuUsec != 0 && end.cpuUsec >= start.cpuUsec {
		seconds := float64(end.cpuUsec-start.cpuUsec) / 1e6
		usage.CgroupCPUUsageCPUSeconds = &seconds
	}
	return usage
}

type runMetadata struct {
	TimestampUTC      time.Time `json:"timestamp_utc"`
	GitCommit         string    `json:"git_commit,omitempty"`
	GoVersion         string    `json:"go_version"`
	GOOS              string    `json:"goos"`
	GOARCH            string    `json:"goarch"`
	NumCPU            int       `json:"num_cpu"`
	GOMAXPROCS        int       `json:"gomaxprocs"`
	Kernel            string    `json:"kernel,omitempty"`
	ServerVersion     string    `json:"server_version"`
	ServerStoreMode   string    `json:"server_store_mode"`
	ServerHeap        string    `json:"server_heap"`
	GoClientAndServer string    `json:"client_server_placement"`
	WorkingTreeDirty  bool      `json:"working_tree_dirty"`
	ConfigSHA256      string    `json:"client_config_sha256,omitempty"`
}

func commandOutput(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	if cmd.Run() != nil {
		return ""
	}
	return strings.TrimSpace(output.String())
}

func collectMetadata() runMetadata {
	metadata := runMetadata{TimestampUTC: time.Now().UTC(), GOOS: runtime.GOOS,
		GOARCH: runtime.GOARCH, NumCPU: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0),
		ServerVersion:     envOr("SEATA_SERVER_VERSION", "unknown"),
		ServerStoreMode:   envOr("SEATA_SERVER_STORE_MODE", "unknown"),
		ServerHeap:        envOr("SEATA_SERVER_HEAP", "unknown"),
		GoClientAndServer: envOr("SEATA_BENCH_TOPOLOGY", "unknown"),
		GoVersion:         runtime.Version()}
	metadata.GitCommit = commandOutput("git", "rev-parse", "HEAD")
	metadata.Kernel = commandOutput("uname", "-srmo")
	metadata.WorkingTreeDirty = commandOutput("git", "status", "--porcelain") != ""
	if config, err := os.ReadFile(*configPathFlag); err == nil {
		metadata.ConfigSHA256 = fmt.Sprintf("%x", sha256.Sum256(config))
	}
	return metadata
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func validateSuccessfulPhase(result phaseResult) error {
	if result.Failures != 0 || result.Successes == 0 {
		return fmt.Errorf("phase had %d successes and %d failures: %v", result.Successes, result.Failures, result.ErrorSamples)
	}
	if result.AllLatency.Count != result.Attempts || result.SuccessLatency.Count != result.Successes {
		return fmt.Errorf("latency sample counts do not match transaction counts: %+v", result)
	}
	if result.TryCallbacks != result.Successes || result.CommitCallbacks != result.Successes || result.CancelCallbacks != 0 {
		return fmt.Errorf("callback counts do not match successful transactions: tries=%d confirms=%d cancels=%d successes=%d",
			result.TryCallbacks, result.CommitCallbacks, result.CancelCallbacks, result.Successes)
	}
	return nil
}

func TestTCCLoad(t *testing.T) {
	if !*loadFlag {
		t.Skip("enable with -tcc.load; this test sends real transactions")
	}
	if err := validateLoadFlags(); err != nil {
		t.Fatal(err)
	}
	runtime.GOMAXPROCS(*gomaxprocsFlag)
	if err := initialize(); err != nil {
		t.Fatal(err)
	}
	result := struct {
		SchemaVersion      int          `json:"schema_version"`
		Concurrency        int          `json:"concurrency"`
		WorkerCount        int          `json:"worker_count"`
		GOMAXPROCS         int          `json:"gomaxprocs"`
		PayloadBytes       int          `json:"payload_bytes"`
		TransactionTimeout string       `json:"transaction_timeout"`
		PercentileMethod   string       `json:"percentile_method"`
		Metadata           runMetadata  `json:"metadata"`
		Warmup             *phaseResult `json:"warmup"`
		Measurement        *phaseResult `json:"measurement"`
		Status             string       `json:"status"`
	}{SchemaVersion: 1, Concurrency: *concurrencyFlag, WorkerCount: *concurrencyFlag,
		GOMAXPROCS: runtime.GOMAXPROCS(0), Metadata: collectMetadata(),
		PayloadBytes: *payloadFlag, TransactionTimeout: transactionTimeout.String(),
		PercentileMethod: "histogram upper bound; <=1.5625% relative bucket width; exact below 64ns", Status: "failed"}
	// Persist even a failed warm-up, and never mix warm-up latency with measurement.
	defer func() {
		if *outputFlag != "" {
			if err := writeJSON(*outputFlag, result); err != nil {
				t.Errorf("write result: %v", err)
			}
		}
	}()
	if *warmupFlag > 0 {
		fmt.Println("TCC_PHASE=warmup")
		warmup := measure(*concurrencyFlag, *warmupFlag, 0, runTransaction)
		result.Warmup = &warmup
		if err := validateSuccessfulPhase(warmup); err != nil {
			t.Fatalf("warm-up validation failed: %v", err)
		}
	}
	fmt.Println("TCC_PHASE=measurement")
	measurement := measure(*concurrencyFlag, *durationFlag, 0, runTransaction)
	result.Measurement = &measurement
	t.Logf("workers=%d GOMAXPROCS=%d success=%d failure=%d TPS=%.2f P95=%.3fms P99=%.3fms drain=%.3fs",
		result.Concurrency, result.GOMAXPROCS, measurement.Successes, measurement.Failures,
		measurement.SuccessfulTPS, measurement.SuccessLatency.P95MS, measurement.SuccessLatency.P99MS, measurement.DrainSeconds)
	if err := validateSuccessfulPhase(measurement); err != nil {
		t.Fatalf("measured phase validation failed: %v", err)
	}
	result.Status = "passed"
}
