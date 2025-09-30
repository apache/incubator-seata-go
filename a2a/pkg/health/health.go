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

package health

import (
	"context"
	"net/http"
	"time"

	"seata-go-ai-a2a/pkg/metrics"

	"github.com/heptiolabs/healthcheck"
)

// Manager wraps the healthcheck library and integrates with our metrics
type Manager struct {
	handler healthcheck.Handler
	metrics *metrics.PrometheusMetrics
}

// NewManager creates a new health check manager using heptiolabs/healthcheck
func NewManager(metrics *metrics.PrometheusMetrics) *Manager {
	handler := healthcheck.NewHandler()

	return &Manager{
		handler: handler,
		metrics: metrics,
	}
}

// AddDatabaseCheck adds a database health check
func (m *Manager) AddDatabaseCheck(name string, db interface {
	HealthCheck(ctx context.Context) error
}) {
	m.handler.AddReadinessCheck(name, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := db.HealthCheck(ctx)

		// Record metrics
		if m.metrics != nil {
			m.metrics.RecordHealthCheck(name, err == nil)
		}

		return err
	})
}

// AddLivenessCheck adds a custom liveness check
func (m *Manager) AddLivenessCheck(name string, check healthcheck.Check) {
	m.handler.AddLivenessCheck(name, func() error {
		err := check()

		// Record metrics
		if m.metrics != nil {
			m.metrics.RecordHealthCheck(name+"_liveness", err == nil)
		}

		return err
	})
}

// AddReadinessCheck adds a custom readiness check
func (m *Manager) AddReadinessCheck(name string, check healthcheck.Check) {
	m.handler.AddReadinessCheck(name, func() error {
		err := check()

		// Record metrics
		if m.metrics != nil {
			m.metrics.RecordHealthCheck(name+"_readiness", err == nil)
		}

		return err
	})
}

// GetHandler returns the HTTP handler for health checks
// Provides these endpoints:
// GET /live - liveness probe
// GET /ready - readiness probe
func (m *Manager) GetHandler() http.Handler {
	return m.handler
}

// GetHTTPHandler returns an extended HTTP handler with additional endpoints
func (m *Manager) GetHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// Mount the healthcheck handler
	mux.HandleFunc("/live", m.handler.LiveEndpoint)
	mux.HandleFunc("/ready", m.handler.ReadyEndpoint)

	// Add a combined health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return mux
}
