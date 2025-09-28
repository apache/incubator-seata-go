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
	"fmt"
	"time"

	"seata-go-ai-transport/common"
)

// SSEClientBuilder implements common.ClientBuilder for SSE-enabled JSON-RPC
type SSEClientBuilder struct{}

var _ common.ClientBuilder = (*SSEClientBuilder)(nil)

// Build creates a new SSE JSON-RPC client with the given config
func (b *SSEClientBuilder) Build(config *common.Config) (common.Client, error) {
	if config.Protocol != common.ProtocolJSONRPC {
		return nil, fmt.Errorf("invalid protocol %s for SSE JSON-RPC client", config.Protocol)
	}

	clientConfig := &ClientConfig{
		Endpoint: config.Address,
		Timeout:  30 * time.Second,
		Headers:  make(map[string]string),
	}

	sseConfig := &SSEClientConfig{
		ClientConfig: clientConfig,
	}

	// Parse options
	if config.Options != nil {
		if endpoint, ok := config.Options["endpoint"].(string); ok {
			clientConfig.Endpoint = endpoint
		}
		if streamEndpoint, ok := config.Options["stream_endpoint"].(string); ok {
			sseConfig.StreamEndpoint = streamEndpoint
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
		// Support streaming flag to determine if SSE should be used
		if streaming, ok := config.Options["streaming"].(bool); ok && streaming {
			return NewSSEClient(sseConfig), nil
		}
	}

	// Use address as endpoint if not specified in options
	if clientConfig.Endpoint == "" {
		clientConfig.Endpoint = config.Address
	}

	// Return SSE client by default for JSON-RPC with streaming capability
	return NewSSEClient(sseConfig), nil
}

// Protocol returns the protocol this builder supports
func (b *SSEClientBuilder) Protocol() common.Protocol {
	return common.ProtocolJSONRPC
}

// SSEServerBuilder implements common.ServerBuilder for SSE-enabled JSON-RPC
type SSEServerBuilder struct{}

var _ common.ServerBuilder = (*SSEServerBuilder)(nil)

// Build creates a new SSE JSON-RPC server with the given config
func (b *SSEServerBuilder) Build(config *common.Config) (common.Server, error) {
	if config.Protocol != common.ProtocolJSONRPC {
		return nil, fmt.Errorf("invalid protocol %s for SSE JSON-RPC server", config.Protocol)
	}

	serverConfig := &ServerConfig{
		Address:      config.Address,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	sseConfig := &SSEServerConfig{
		ServerConfig: serverConfig,
	}

	// Parse options
	if config.Options != nil {
		if streamPath, ok := config.Options["stream_path"].(string); ok {
			sseConfig.StreamPath = streamPath
		}
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
		// Support streaming flag to determine if SSE should be used
		if streaming, ok := config.Options["streaming"].(bool); ok && streaming {
			return NewSSEServer(sseConfig), nil
		}
	}

	// Return SSE server by default for JSON-RPC with streaming capability
	return NewSSEServer(sseConfig), nil
}

// Protocol returns the protocol this builder supports
func (b *SSEServerBuilder) Protocol() common.Protocol {
	return common.ProtocolJSONRPC
}

// Convenience constructors

// NewSSEJSONRPCClient creates a new SSE JSON-RPC client directly
func NewSSEJSONRPCClient(endpoint string, streamEndpoint string) *SSEClient {
	config := &SSEClientConfig{
		ClientConfig: &ClientConfig{
			Endpoint: endpoint,
			Headers:  make(map[string]string),
		},
		StreamEndpoint: streamEndpoint,
	}
	return NewSSEClient(config)
}

// NewSSEJSONRPCServer creates a new SSE JSON-RPC server directly
func NewSSEJSONRPCServer(address string, streamPath string) *SSEServer {
	config := &SSEServerConfig{
		ServerConfig: &ServerConfig{
			Address: address,
		},
		StreamPath: streamPath,
	}
	return NewSSEServer(config)
}
