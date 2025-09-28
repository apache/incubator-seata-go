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

package jsonrpc

import (
	"encoding/json"
	"fmt"
	"time"

	"seata-go-ai-transport/common"
)

// ClientBuilder implements common.ClientBuilder for JSON-RPC
type ClientBuilder struct{}

var _ common.ClientBuilder = (*ClientBuilder)(nil)

// Build creates a new JSON-RPC client with the given config
func (b *ClientBuilder) Build(config *common.Config) (common.Client, error) {
	if config.Protocol != common.ProtocolJSONRPC {
		return nil, fmt.Errorf("invalid protocol %s for JSON-RPC client", config.Protocol)
	}

	clientConfig := &ClientConfig{
		Endpoint: config.Address,
		Timeout:  30 * time.Second,
		Headers:  make(map[string]string),
	}

	// Parse options
	if config.Options != nil {
		if endpoint, ok := config.Options["endpoint"].(string); ok {
			clientConfig.Endpoint = endpoint
		}
		if timeoutStr, ok := config.Options["timeout"].(string); ok {
			if timeout, err := time.ParseDuration(timeoutStr); err == nil {
				clientConfig.Timeout = timeout
			}
		}
		if timeout, ok := config.Options["timeout"].(time.Duration); ok {
			clientConfig.Timeout = timeout
		}
		if headers, ok := config.Options["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				if s, ok := v.(string); ok {
					clientConfig.Headers[k] = s
				}
			}
		}
		if headers, ok := config.Options["headers"].(map[string]string); ok {
			for k, v := range headers {
				clientConfig.Headers[k] = v
			}
		}
	}

	// Use address as endpoint if not specified in options
	if clientConfig.Endpoint == "" {
		clientConfig.Endpoint = config.Address
	}

	return NewClient(clientConfig), nil
}

// Protocol returns the protocol this builder supports
func (b *ClientBuilder) Protocol() common.Protocol {
	return common.ProtocolJSONRPC
}

// ServerBuilder implements common.ServerBuilder for JSON-RPC
type ServerBuilder struct{}

var _ common.ServerBuilder = (*ServerBuilder)(nil)

// Build creates a new JSON-RPC server with the given config
func (b *ServerBuilder) Build(config *common.Config) (common.Server, error) {
	if config.Protocol != common.ProtocolJSONRPC {
		return nil, fmt.Errorf("invalid protocol %s for JSON-RPC server", config.Protocol)
	}

	serverConfig := &ServerConfig{
		Address:      config.Address,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Parse options
	if config.Options != nil {
		if readTimeoutStr, ok := config.Options["read_timeout"].(string); ok {
			if timeout, err := time.ParseDuration(readTimeoutStr); err == nil {
				serverConfig.ReadTimeout = timeout
			}
		}
		if readTimeout, ok := config.Options["read_timeout"].(time.Duration); ok {
			serverConfig.ReadTimeout = readTimeout
		}
		if writeTimeoutStr, ok := config.Options["write_timeout"].(string); ok {
			if timeout, err := time.ParseDuration(writeTimeoutStr); err == nil {
				serverConfig.WriteTimeout = timeout
			}
		}
		if writeTimeout, ok := config.Options["write_timeout"].(time.Duration); ok {
			serverConfig.WriteTimeout = writeTimeout
		}
		if idleTimeoutStr, ok := config.Options["idle_timeout"].(string); ok {
			if timeout, err := time.ParseDuration(idleTimeoutStr); err == nil {
				serverConfig.IdleTimeout = timeout
			}
		}
		if idleTimeout, ok := config.Options["idle_timeout"].(time.Duration); ok {
			serverConfig.IdleTimeout = idleTimeout
		}
	}

	return NewServer(serverConfig), nil
}

// Protocol returns the protocol this builder supports
func (b *ServerBuilder) Protocol() common.Protocol {
	return common.ProtocolJSONRPC
}

// NewClientConfig creates a client config from JSON
func NewClientConfig(data []byte) (*ClientConfig, error) {
	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal client config: %w", err)
	}
	return &config, nil
}

// NewServerConfig creates a server config from JSON
func NewServerConfig(data []byte) (*ServerConfig, error) {
	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal server config: %w", err)
	}
	return &config, nil
}
