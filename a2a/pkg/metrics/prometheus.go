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

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics wraps Prometheus metrics for A2A system
type PrometheusMetrics struct {
	registry *prometheus.Registry

	// Task metrics
	tasksCreatedTotal     *prometheus.CounterVec
	tasksCompletedTotal   *prometheus.CounterVec
	tasksFailedTotal      *prometheus.CounterVec
	tasksCancelledTotal   *prometheus.CounterVec
	tasksActiveGauge      *prometheus.GaugeVec
	taskDurationHistogram *prometheus.HistogramVec

	// Authentication metrics
	authAttemptsTotal *prometheus.CounterVec
	authFailuresTotal *prometheus.CounterVec
	authDurationHist  *prometheus.HistogramVec

	// JWKS metrics
	jwksFetchTotal     *prometheus.CounterVec
	jwksCacheHitTotal  *prometheus.CounterVec
	jwksCacheMissTotal *prometheus.CounterVec

	// gRPC metrics
	grpcRequestsTotal   *prometheus.CounterVec
	grpcRequestDuration *prometheus.HistogramVec
	grpcActiveStreams   *prometheus.GaugeVec

	// Health metrics
	healthCheckStatus *prometheus.GaugeVec
	healthCheckTotal  *prometheus.CounterVec

	// System metrics
	memoryUsageBytes *prometheus.GaugeVec
	goroutineCount   prometheus.Gauge
}

// NewPrometheusMetrics creates a new Prometheus metrics instance
func NewPrometheusMetrics(subsystem string) *PrometheusMetrics {
	if subsystem == "" {
		subsystem = "a2a"
	}

	registry := prometheus.NewRegistry()

	// Add Go runtime metrics
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	pm := &PrometheusMetrics{
		registry: registry,

		// Task metrics
		tasksCreatedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "tasks_created_total",
				Help:      "Total number of tasks created",
			},
			[]string{"context_id"},
		),

		tasksCompletedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "tasks_completed_total",
				Help:      "Total number of tasks completed",
			},
			[]string{"context_id"},
		),

		tasksFailedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "tasks_failed_total",
				Help:      "Total number of tasks failed",
			},
			[]string{"context_id", "reason"},
		),

		tasksCancelledTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "tasks_cancelled_total",
				Help:      "Total number of tasks cancelled",
			},
			[]string{"context_id"},
		),

		tasksActiveGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      "tasks_active",
				Help:      "Number of currently active tasks",
			},
			[]string{"state"},
		),

		taskDurationHistogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      "task_duration_seconds",
				Help:      "Task execution duration in seconds",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
			},
			[]string{"context_id", "state"},
		),

		// Authentication metrics
		authAttemptsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "auth_attempts_total",
				Help:      "Total number of authentication attempts",
			},
			[]string{"method", "result"},
		),

		authFailuresTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "auth_failures_total",
				Help:      "Total number of authentication failures",
			},
			[]string{"method", "reason"},
		),

		authDurationHist: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      "auth_duration_seconds",
				Help:      "Authentication duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method"},
		),

		// JWKS metrics
		jwksFetchTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "jwks_fetch_total",
				Help:      "Total number of JWKS fetch attempts",
			},
			[]string{"url", "result"},
		),

		jwksCacheHitTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "jwks_cache_hit_total",
				Help:      "Total number of JWKS cache hits",
			},
			[]string{"url"},
		),

		jwksCacheMissTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "jwks_cache_miss_total",
				Help:      "Total number of JWKS cache misses",
			},
			[]string{"url"},
		),

		// gRPC metrics
		grpcRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "grpc_requests_total",
				Help:      "Total number of gRPC requests",
			},
			[]string{"method", "code"},
		),

		grpcRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      "grpc_request_duration_seconds",
				Help:      "gRPC request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method"},
		),

		grpcActiveStreams: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      "grpc_active_streams",
				Help:      "Number of active gRPC streams",
			},
			[]string{"method"},
		),

		// Health metrics
		healthCheckStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      "health_check_status",
				Help:      "Health check status (1 = healthy, 0 = unhealthy)",
			},
			[]string{"component"},
		),

		healthCheckTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "health_checks_total",
				Help:      "Total number of health checks performed",
			},
			[]string{"component", "result"},
		),

		// System metrics
		memoryUsageBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      "memory_usage_bytes",
				Help:      "Memory usage in bytes",
			},
			[]string{"type"},
		),

		goroutineCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      "goroutines",
				Help:      "Number of goroutines",
			},
		),
	}

	// Register all metrics
	registry.MustRegister(
		pm.tasksCreatedTotal,
		pm.tasksCompletedTotal,
		pm.tasksFailedTotal,
		pm.tasksCancelledTotal,
		pm.tasksActiveGauge,
		pm.taskDurationHistogram,
		pm.authAttemptsTotal,
		pm.authFailuresTotal,
		pm.authDurationHist,
		pm.jwksFetchTotal,
		pm.jwksCacheHitTotal,
		pm.jwksCacheMissTotal,
		pm.grpcRequestsTotal,
		pm.grpcRequestDuration,
		pm.grpcActiveStreams,
		pm.healthCheckStatus,
		pm.healthCheckTotal,
		pm.memoryUsageBytes,
		pm.goroutineCount,
	)

	return pm
}

// Task metrics methods
func (pm *PrometheusMetrics) RecordTaskCreated(contextID string) {
	pm.tasksCreatedTotal.WithLabelValues(contextID).Inc()
}

func (pm *PrometheusMetrics) RecordTaskCompleted(contextID string, duration time.Duration) {
	pm.tasksCompletedTotal.WithLabelValues(contextID).Inc()
	pm.taskDurationHistogram.WithLabelValues(contextID, "completed").Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordTaskFailed(contextID, reason string, duration time.Duration) {
	pm.tasksFailedTotal.WithLabelValues(contextID, reason).Inc()
	pm.taskDurationHistogram.WithLabelValues(contextID, "failed").Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordTaskCancelled(contextID string, duration time.Duration) {
	pm.tasksCancelledTotal.WithLabelValues(contextID).Inc()
	pm.taskDurationHistogram.WithLabelValues(contextID, "cancelled").Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) SetActiveTasksCount(state string, count int) {
	pm.tasksActiveGauge.WithLabelValues(state).Set(float64(count))
}

// Authentication metrics methods
func (pm *PrometheusMetrics) RecordAuthAttempt(method, result string, duration time.Duration) {
	pm.authAttemptsTotal.WithLabelValues(method, result).Inc()
	pm.authDurationHist.WithLabelValues(method).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordAuthFailure(method, reason string) {
	pm.authFailuresTotal.WithLabelValues(method, reason).Inc()
}

// JWKS metrics methods
func (pm *PrometheusMetrics) RecordJWKSFetch(url, result string) {
	pm.jwksFetchTotal.WithLabelValues(url, result).Inc()
}

func (pm *PrometheusMetrics) RecordJWKSCacheHit(url string) {
	pm.jwksCacheHitTotal.WithLabelValues(url).Inc()
}

func (pm *PrometheusMetrics) RecordJWKSCacheMiss(url string) {
	pm.jwksCacheMissTotal.WithLabelValues(url).Inc()
}

// gRPC metrics methods
func (pm *PrometheusMetrics) RecordGRPCRequest(method, code string, duration time.Duration) {
	pm.grpcRequestsTotal.WithLabelValues(method, code).Inc()
	pm.grpcRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) SetActiveStreamsCount(method string, count int) {
	pm.grpcActiveStreams.WithLabelValues(method).Set(float64(count))
}

// Health check methods
func (pm *PrometheusMetrics) RecordHealthCheck(component string, healthy bool) {
	status := 0.0
	result := "unhealthy"
	if healthy {
		status = 1.0
		result = "healthy"
	}

	pm.healthCheckStatus.WithLabelValues(component).Set(status)
	pm.healthCheckTotal.WithLabelValues(component, result).Inc()
}

// System metrics methods
func (pm *PrometheusMetrics) SetMemoryUsage(memType string, bytes int64) {
	pm.memoryUsageBytes.WithLabelValues(memType).Set(float64(bytes))
}

func (pm *PrometheusMetrics) SetGoroutineCount(count int) {
	pm.goroutineCount.Set(float64(count))
}

// GetRegistry returns the Prometheus registry
func (pm *PrometheusMetrics) GetRegistry() *prometheus.Registry {
	return pm.registry
}

// GetHandler returns an HTTP handler for metrics endpoint
func (pm *PrometheusMetrics) GetHandler() http.Handler {
	return promhttp.HandlerFor(pm.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// StartMetricsServer starts a dedicated HTTP server for metrics
func (pm *PrometheusMetrics) StartMetricsServer(port int) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", pm.GetHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}
