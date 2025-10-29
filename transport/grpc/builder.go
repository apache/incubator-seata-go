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

package grpc

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"seata-go-ai-transport/common"
)

// ClientBuilder implements common.ClientBuilder for gRPC
type ClientBuilder struct{}

var _ common.ClientBuilder = (*ClientBuilder)(nil)

// Build creates a new gRPC client with the given config
func (b *ClientBuilder) Build(config *common.Config) (common.Client, error) {
	if config.Protocol != common.ProtocolGRPC {
		return nil, fmt.Errorf("invalid protocol %s for gRPC client", config.Protocol)
	}

	clientConfig := &ClientConfig{
		Target:   config.Address,
		Insecure: true, // Default to insecure for simplicity
		Timeout:  30 * time.Second,
		Methods:  make(map[string]MethodInfo),
	}

	var dialOpts []grpc.DialOption

	// Parse options
	if config.Options != nil {
		if target, ok := config.Options["target"].(string); ok {
			clientConfig.Target = target
		}
		if ins, ok := config.Options["insecure"].(bool); ok {
			clientConfig.Insecure = ins
		}
		if timeoutStr, ok := config.Options["timeout"].(string); ok {
			if timeout, err := time.ParseDuration(timeoutStr); err == nil {
				clientConfig.Timeout = timeout
			}
		}
		if timeout, ok := config.Options["timeout"].(time.Duration); ok {
			clientConfig.Timeout = timeout
		}
		if options, ok := config.Options["dial_options"].([]grpc.DialOption); ok {
			dialOpts = append(dialOpts, options...)
		}
		if methods, ok := config.Options["methods"].(map[string]MethodInfo); ok {
			clientConfig.Methods = methods
		}
	}

	// Use address as target if not specified in options
	if clientConfig.Target == "" {
		clientConfig.Target = config.Address
	}

	clientConfig.Options = dialOpts

	return NewClient(clientConfig)
}

// Protocol returns the protocol this builder supports
func (b *ClientBuilder) Protocol() common.Protocol {
	return common.ProtocolGRPC
}

// ServerBuilder implements common.ServerBuilder for gRPC
type ServerBuilder struct{}

var _ common.ServerBuilder = (*ServerBuilder)(nil)

// Build creates a new gRPC server with the given config
func (b *ServerBuilder) Build(config *common.Config) (common.Server, error) {
	if config.Protocol != common.ProtocolGRPC {
		return nil, fmt.Errorf("invalid protocol %s for gRPC server", config.Protocol)
	}

	serverConfig := &ServerConfig{
		Address: config.Address,
		Methods: make(map[string]MethodInfo),
	}

	var serverOpts []grpc.ServerOption

	// Parse options
	if config.Options != nil {
		if options, ok := config.Options["server_options"].([]grpc.ServerOption); ok {
			serverOpts = append(serverOpts, options...)
		}
		if methods, ok := config.Options["methods"].(map[string]MethodInfo); ok {
			serverConfig.Methods = methods
		}
	}

	serverConfig.Options = serverOpts

	return NewServer(serverConfig)
}

// Protocol returns the protocol this builder supports
func (b *ServerBuilder) Protocol() common.Protocol {
	return common.ProtocolGRPC
}

// Helper functions for creating configurations

// NewGRPCClientConfig creates a client config with common options
func NewGRPCClientConfig(target string, isInsecure bool) *ClientConfig {
	config := &ClientConfig{
		Target:   target,
		Insecure: isInsecure,
		Timeout:  30 * time.Second,
		Methods:  make(map[string]MethodInfo),
		Options:  []grpc.DialOption{},
	}

	if isInsecure {
		config.Options = append(config.Options, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return config
}

// NewGRPCServerConfig creates a server config with common options
func NewGRPCServerConfig(address string) *ServerConfig {
	return &ServerConfig{
		Address: address,
		Methods: make(map[string]MethodInfo),
		Options: []grpc.ServerOption{},
	}
}

// WithDialOptions adds dial options to client config
func (c *ClientConfig) WithDialOptions(opts ...grpc.DialOption) *ClientConfig {
	c.Options = append(c.Options, opts...)
	return c
}

// WithServerOptions adds server options to server config
func (c *ServerConfig) WithServerOptions(opts ...grpc.ServerOption) *ServerConfig {
	c.Options = append(c.Options, opts...)
	return c
}

// WithMethods adds method information to config
func (c *ClientConfig) WithMethods(methods map[string]MethodInfo) *ClientConfig {
	if c.Methods == nil {
		c.Methods = make(map[string]MethodInfo)
	}
	for k, v := range methods {
		c.Methods[k] = v
	}
	return c
}

// WithMethods adds method information to server config
func (c *ServerConfig) WithMethods(methods map[string]MethodInfo) *ServerConfig {
	if c.Methods == nil {
		c.Methods = make(map[string]MethodInfo)
	}
	for k, v := range methods {
		c.Methods[k] = v
	}
	return c
}
