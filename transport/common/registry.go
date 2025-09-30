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

package common

import (
	"fmt"
	"sync"
)

// Registry manages transport builders
type Registry struct {
	mu             sync.RWMutex
	clientBuilders map[Protocol]ClientBuilder
	serverBuilders map[Protocol]ServerBuilder
}

// NewRegistry creates a new transport registry
func NewRegistry() *Registry {
	return &Registry{
		clientBuilders: make(map[Protocol]ClientBuilder),
		serverBuilders: make(map[Protocol]ServerBuilder),
	}
}

// RegisterClientBuilder registers a client builder for a protocol
func (r *Registry) RegisterClientBuilder(protocol Protocol, builder ClientBuilder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clientBuilders[protocol] = builder
}

// RegisterServerBuilder registers a server builder for a protocol
func (r *Registry) RegisterServerBuilder(protocol Protocol, builder ServerBuilder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.serverBuilders[protocol] = builder
}

// GetClientBuilder gets a client builder for a protocol
func (r *Registry) GetClientBuilder(protocol Protocol) (ClientBuilder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	builder, exists := r.clientBuilders[protocol]
	if !exists {
		return nil, fmt.Errorf("no client builder registered for protocol %s", protocol)
	}
	return builder, nil
}

// GetServerBuilder gets a server builder for a protocol
func (r *Registry) GetServerBuilder(protocol Protocol) (ServerBuilder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	builder, exists := r.serverBuilders[protocol]
	if !exists {
		return nil, fmt.Errorf("no server builder registered for protocol %s", protocol)
	}
	return builder, nil
}

// CreateClient creates a client using the registered builder
func (r *Registry) CreateClient(config *Config) (Client, error) {
	builder, err := r.GetClientBuilder(config.Protocol)
	if err != nil {
		return nil, err
	}
	return builder.Build(config)
}

// CreateServer creates a server using the registered builder
func (r *Registry) CreateServer(config *Config) (Server, error) {
	builder, err := r.GetServerBuilder(config.Protocol)
	if err != nil {
		return nil, err
	}
	return builder.Build(config)
}

// SupportedProtocols returns all supported protocols
func (r *Registry) SupportedProtocols() []Protocol {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protocols := make([]Protocol, 0, len(r.clientBuilders))
	for protocol := range r.clientBuilders {
		protocols = append(protocols, protocol)
	}
	return protocols
}

// Default registry instance
var defaultRegistry = NewRegistry()

// RegisterClientBuilder registers a client builder in the default registry
func RegisterClientBuilder(protocol Protocol, builder ClientBuilder) {
	defaultRegistry.RegisterClientBuilder(protocol, builder)
}

// RegisterServerBuilder registers a server builder in the default registry
func RegisterServerBuilder(protocol Protocol, builder ServerBuilder) {
	defaultRegistry.RegisterServerBuilder(protocol, builder)
}

// CreateClient creates a client using the default registry
func CreateClient(config *Config) (Client, error) {
	return defaultRegistry.CreateClient(config)
}

// CreateServer creates a server using the default registry
func CreateServer(config *Config) (Server, error) {
	return defaultRegistry.CreateServer(config)
}

// GetSupportedProtocols returns all supported protocols from default registry
func GetSupportedProtocols() []Protocol {
	return defaultRegistry.SupportedProtocols()
}
