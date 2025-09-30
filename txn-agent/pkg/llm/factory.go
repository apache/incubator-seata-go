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

package llm

import (
	"fmt"
	"strings"
)

// NewClient creates a new LLM client based on the specified type
func NewClient(clientType ClientType, config Config) (LLMClient, error) {
	switch clientType {
	case ClientTypeOpenAI:
		return NewOpenAIClient(config), nil
	case ClientTypeAnthropic:
		return NewAnthropicClient(config), nil
	case ClientTypeGemini:
		return NewGeminiClient(config), nil
	default:
		return nil, fmt.Errorf("unsupported client type: %s", clientType)
	}
}

// AutoDetectClientType attempts to auto-detect the client type based on the base URL
func AutoDetectClientType(baseURL string) ClientType {
	baseURL = strings.ToLower(baseURL)

	if strings.Contains(baseURL, "openai") || strings.Contains(baseURL, "dashscope") {
		return ClientTypeOpenAI
	}

	if strings.Contains(baseURL, "anthropic") {
		return ClientTypeAnthropic
	}

	if strings.Contains(baseURL, "googleapis") || strings.Contains(baseURL, "gemini") {
		return ClientTypeGemini
	}

	return ClientTypeOpenAI
}

// NewClientWithAutoDetect creates a new LLM client with auto-detection
func NewClientWithAutoDetect(config Config) (LLMClient, error) {
	clientType := AutoDetectClientType(config.BaseURL)
	return NewClient(clientType, config)
}

// ConfigBuilder provides a fluent interface for building Config
type ConfigBuilder struct {
	config Config
}

// NewConfigBuilder creates a new config builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: Config{
			Timeout:  30,
			MaxRetry: 3,
		},
	}
}

// WithAPIKey sets the API key
func (b *ConfigBuilder) WithAPIKey(apiKey string) *ConfigBuilder {
	b.config.APIKey = apiKey
	return b
}

// WithBaseURL sets the base URL
func (b *ConfigBuilder) WithBaseURL(baseURL string) *ConfigBuilder {
	b.config.BaseURL = baseURL
	return b
}

// WithModel sets the model
func (b *ConfigBuilder) WithModel(model string) *ConfigBuilder {
	b.config.Model = model
	return b
}

// WithTimeout sets the timeout in seconds
func (b *ConfigBuilder) WithTimeout(timeout int) *ConfigBuilder {
	b.config.Timeout = timeout
	return b
}

// WithMaxRetry sets the maximum number of retries
func (b *ConfigBuilder) WithMaxRetry(maxRetry int) *ConfigBuilder {
	b.config.MaxRetry = maxRetry
	return b
}

// WithUserAgent sets the user agent
func (b *ConfigBuilder) WithUserAgent(userAgent string) *ConfigBuilder {
	b.config.UserAgent = userAgent
	return b
}

// Build returns the built config
func (b *ConfigBuilder) Build() Config {
	return b.config
}
