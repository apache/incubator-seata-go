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

package server

import (
	"net/http"
	"time"

	"agenthub/pkg/auth"
	"agenthub/pkg/utils"
)

// ServerBuilder implements builder pattern for server configuration
type ServerBuilder struct {
	server *HTTPServer
}

// HTTPServer represents an HTTP server following K8s patterns
type HTTPServer struct {
	addr           string
	handler        http.Handler
	readTimeout    time.Duration
	writeTimeout   time.Duration
	maxHeaderBytes int
	middleware     []Middleware
	routes         []Route
	authMiddleware *auth.Middleware
	logger         *utils.Logger
}

// Middleware represents HTTP middleware
type Middleware func(http.Handler) http.Handler

// Route represents an HTTP route
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// NewServerBuilder creates a new server builder
func NewServerBuilder() *ServerBuilder {
	return &ServerBuilder{
		server: &HTTPServer{
			addr:           ":8080",
			readTimeout:    30 * time.Second,
			writeTimeout:   30 * time.Second,
			maxHeaderBytes: 1 << 20, // 1 MB
			middleware:     make([]Middleware, 0),
			routes:         make([]Route, 0),
			logger:         utils.WithField("component", "http-server"),
		},
	}
}

// WithAddress sets the server address
func (b *ServerBuilder) WithAddress(addr string) *ServerBuilder {
	b.server.addr = addr
	return b
}

// WithReadTimeout sets the read timeout
func (b *ServerBuilder) WithReadTimeout(timeout time.Duration) *ServerBuilder {
	b.server.readTimeout = timeout
	return b
}

// WithWriteTimeout sets the write timeout
func (b *ServerBuilder) WithWriteTimeout(timeout time.Duration) *ServerBuilder {
	b.server.writeTimeout = timeout
	return b
}

// WithMaxHeaderBytes sets the maximum header bytes
func (b *ServerBuilder) WithMaxHeaderBytes(bytes int) *ServerBuilder {
	b.server.maxHeaderBytes = bytes
	return b
}

// WithMiddleware adds middleware to the server
func (b *ServerBuilder) WithMiddleware(middleware ...Middleware) *ServerBuilder {
	b.server.middleware = append(b.server.middleware, middleware...)
	return b
}

// WithAuthMiddleware sets the authentication middleware
func (b *ServerBuilder) WithAuthMiddleware(authMiddleware *auth.Middleware) *ServerBuilder {
	b.server.authMiddleware = authMiddleware
	return b
}

// WithLogger sets the logger
func (b *ServerBuilder) WithLogger(logger *utils.Logger) *ServerBuilder {
	b.server.logger = logger
	return b
}

// WithRoute adds a route to the server
func (b *ServerBuilder) WithRoute(method, path string, handler http.HandlerFunc) *ServerBuilder {
	b.server.routes = append(b.server.routes, Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	})
	return b
}

// WithRoutes adds multiple routes to the server
func (b *ServerBuilder) WithRoutes(routes []Route) *ServerBuilder {
	b.server.routes = append(b.server.routes, routes...)
	return b
}

// Build creates the configured server
func (b *ServerBuilder) Build() *HTTPServer {
	// Setup routes
	mux := http.NewServeMux()
	for _, route := range b.server.routes {
		// Use traditional path-only routing for compatibility
		mux.HandleFunc(route.Path, route.Handler)
	}

	// Apply middleware
	handler := http.Handler(mux)

	// Apply authentication middleware if configured
	if b.server.authMiddleware != nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b.server.authMiddleware.Handler(handler.ServeHTTP)(w, r)
		})
	}

	// Apply other middleware in reverse order
	for i := len(b.server.middleware) - 1; i >= 0; i-- {
		handler = b.server.middleware[i](handler)
	}

	b.server.handler = handler
	return b.server
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	s.logger.Info("HTTP server starting on %s", s.addr)

	server := &http.Server{
		Addr:           s.addr,
		Handler:        s.handler,
		ReadTimeout:    s.readTimeout,
		WriteTimeout:   s.writeTimeout,
		MaxHeaderBytes: s.maxHeaderBytes,
	}

	return server.ListenAndServe()
}

// GetAddress returns the server address
func (s *HTTPServer) GetAddress() string {
	return s.addr
}

// GetHandler returns the server handler
func (s *HTTPServer) GetHandler() http.Handler {
	return s.handler
}

// Common middleware implementations
func LoggingMiddleware(logger *utils.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			logger.Info("HTTP %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
			next.ServeHTTP(w, r)
			logger.Debug("HTTP %s %s completed in %v", r.Method, r.URL.Path, time.Since(start))
		})
	}
}

func CORSMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RecoveryMiddleware(logger *utils.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("HTTP panic recovered: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
