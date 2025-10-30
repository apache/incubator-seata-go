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

package api

import (
	"net/http"
	"strings"
	"time"

	"seata-go-ai-workflow-agent/pkg/logger"
)

// SetupRoutes sets up the HTTP routes
func SetupRoutes(handler *Handler, streamingHandler *StreamingHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// Legacy synchronous API routes
	mux.HandleFunc("/api/v1/orchestrate", corsMiddleware(loggingMiddleware(handler.HandleOrchestrate)))

	// Session-based streaming API routes
	mux.HandleFunc("/api/v1/sessions", corsMiddleware(loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			streamingHandler.HandleCreateSession(w, r)
		} else if r.Method == http.MethodGet {
			streamingHandler.HandleListSessions(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Session-specific routes
	mux.HandleFunc("/api/v1/sessions/", corsMiddleware(loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check for stream endpoint
		if strings.HasSuffix(path, "/stream") {
			streamingHandler.HandleStreamProgress(w, r)
			return
		}

		// Check for result endpoint
		if strings.HasSuffix(path, "/result") {
			streamingHandler.HandleSessionResult(w, r)
			return
		}

		// Check for history endpoint
		if strings.HasSuffix(path, "/history") {
			streamingHandler.HandleSessionHistory(w, r)
			return
		}

		// Otherwise, it's a status request
		if r.Method == http.MethodGet {
			streamingHandler.HandleSessionStatus(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Health check
	mux.HandleFunc("/health", corsMiddleware(handler.HandleHealth))

	// Root redirect
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			respondJSON(w, http.StatusOK, map[string]string{
				"service": "workflow-agent",
				"version": "1.0.0",
				"status":  "running",
			})
			return
		}
		http.NotFound(w, r)
	})

	return mux
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		logger.WithField("method", r.Method).
			WithField("path", r.URL.Path).
			WithField("remote", r.RemoteAddr).
			Info("http request")

		next(w, r)

		logger.WithField("method", r.Method).
			WithField("path", r.URL.Path).
			WithField("duration", time.Since(start)).
			Debug("http request completed")
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from any origin
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}
