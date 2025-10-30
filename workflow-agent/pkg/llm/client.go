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
	"context"
)

import (
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

import (
	"seata-go-ai-workflow-agent/pkg/errors"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// Client wraps genkit for LLM interactions
type Client struct {
	genkit       *genkit.Genkit
	defaultModel string
	apiKey       string
	baseURL      string
	config       map[string]interface{}
}

// Config holds the LLM client configuration
type Config struct {
	Genkit       *genkit.Genkit
	DefaultModel string
	APIKey       string
	BaseURL      string
	Config       map[string]interface{}
}

// NewClient creates a new LLM client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.ConfigError("config cannot be nil")
	}

	if cfg.Genkit == nil {
		return nil, errors.ConfigError("genkit instance is required")
	}

	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "openai/gpt-4o-mini"
	}

	// Initialize global config map
	globalConfig := make(map[string]interface{})
	if cfg.Config != nil {
		for k, v := range cfg.Config {
			globalConfig[k] = v
		}
	}

	// Add API key to config if provided
	if cfg.APIKey != "" {
		globalConfig["apiKey"] = cfg.APIKey
	}

	// Add base URL to config if provided
	if cfg.BaseURL != "" {
		globalConfig["baseUrl"] = cfg.BaseURL
	}

	logger.WithFields(map[string]interface{}{
		"defaultModel": defaultModel,
		"hasAPIKey":    cfg.APIKey != "",
		"hasBaseURL":   cfg.BaseURL != "",
	}).Info("LLM client initialized")

	return &Client{
		genkit:       cfg.Genkit,
		defaultModel: defaultModel,
		apiKey:       cfg.APIKey,
		baseURL:      cfg.BaseURL,
		config:       globalConfig,
	}, nil
}

// Generate performs non-streaming generation
func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, errors.InvalidArgumentError("request cannot be nil")
	}

	if len(req.Messages) == 0 {
		return nil, errors.InvalidArgumentError("messages cannot be empty")
	}

	opts := c.buildGenerateOptions(req)

	logger.WithFields(map[string]interface{}{
		"model":        c.getModelName(req),
		"messageCount": len(req.Messages),
		"stream":       req.Stream,
	}).Debug("generating response")

	resp, err := genkit.Generate(ctx, c.genkit, opts...)
	if err != nil {
		logger.WithField("error", err).Error("generation failed")
		return nil, errors.Wrap(errors.ErrConnection, "generation failed", err)
	}

	response := &GenerateResponse{
		Content:      extractTextFromResponse(resp),
		ToolCalls:    extractToolCallsFromResponse(resp),
		FinishReason: extractFinishReason(resp),
		Usage:        extractUsageStats(resp),
		RawResponse:  resp,
	}

	logger.WithFields(map[string]interface{}{
		"finishReason": response.FinishReason,
		"toolCalls":    len(response.ToolCalls),
	}).Debug("generation completed")

	return response, nil
}

// GenerateStream performs streaming generation
func (c *Client) GenerateStream(ctx context.Context, req *GenerateRequest, callback StreamCallback) (*GenerateResponse, error) {
	if req == nil {
		return nil, errors.InvalidArgumentError("request cannot be nil")
	}

	if callback == nil {
		return nil, errors.InvalidArgumentError("callback cannot be nil")
	}

	if len(req.Messages) == 0 {
		return nil, errors.InvalidArgumentError("messages cannot be empty")
	}

	req.Stream = true
	opts := c.buildGenerateOptions(req)

	// Add streaming callback
	opts = append(opts, ai.WithStreaming(func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		if chunk == nil {
			return nil
		}

		text := ""
		if chunk.Content != nil {
			for _, part := range chunk.Content {
				if part.IsText() {
					text += part.Text
				}
			}
		}

		if text != "" {
			if err := callback(text); err != nil {
				return err
			}
		}

		return nil
	}))

	logger.WithFields(map[string]interface{}{
		"model":        c.getModelName(req),
		"messageCount": len(req.Messages),
	}).Debug("starting streaming generation")

	resp, err := genkit.Generate(ctx, c.genkit, opts...)
	if err != nil {
		logger.WithField("error", err).Error("streaming generation failed")
		return nil, errors.Wrap(errors.ErrConnection, "streaming generation failed", err)
	}

	response := &GenerateResponse{
		Content:      extractTextFromResponse(resp),
		ToolCalls:    extractToolCallsFromResponse(resp),
		FinishReason: extractFinishReason(resp),
		Usage:        extractUsageStats(resp),
		RawResponse:  resp,
	}

	logger.Debug("streaming generation completed")

	return response, nil
}

// GenerateWithTools performs generation with tool calling support
func (c *Client) GenerateWithTools(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, errors.InvalidArgumentError("request cannot be nil")
	}

	if len(req.Tools) == 0 {
		return c.Generate(ctx, req)
	}

	opts := c.buildGenerateOptions(req)

	// Convert tools to genkit format
	// Note: Tool handling in genkit needs to be defined separately
	// For now, we pass tool definitions as config
	logger.WithFields(map[string]interface{}{
		"model":     c.getModelName(req),
		"toolCount": len(req.Tools),
	}).Debug("generating with tools")

	resp, err := genkit.Generate(ctx, c.genkit, opts...)
	if err != nil {
		logger.WithField("error", err).Error("tool-enabled generation failed")
		return nil, errors.Wrap(errors.ErrConnection, "tool-enabled generation failed", err)
	}

	response := &GenerateResponse{
		Content:      extractTextFromResponse(resp),
		ToolCalls:    extractToolCallsFromResponse(resp),
		FinishReason: extractFinishReason(resp),
		Usage:        extractUsageStats(resp),
		RawResponse:  resp,
	}

	logger.WithField("toolCallsCount", len(response.ToolCalls)).Debug("tool-enabled generation completed")

	return response, nil
}

// GenerateStructured performs generation with structured output
func (c *Client) GenerateStructured(ctx context.Context, req *GenerateRequest, output interface{}) error {
	if req == nil {
		return errors.InvalidArgumentError("request cannot be nil")
	}

	if output == nil {
		return errors.InvalidArgumentError("output cannot be nil")
	}

	opts := c.buildGenerateOptions(req)

	// Add output format if response format is specified
	if req.ResponseFormat != nil {
		switch req.ResponseFormat.Type {
		case "json_object", "json_schema":
			opts = append(opts, ai.WithOutputFormat(ai.OutputFormatJSON))
		}
	}

	logger.WithField("model", c.getModelName(req)).Debug("generating structured output")

	resp, err := genkit.Generate(ctx, c.genkit, opts...)
	if err != nil {
		logger.WithField("error", err).Error("structured generation failed")
		return errors.Wrap(errors.ErrConnection, "structured generation failed", err)
	}

	if err := resp.Output(output); err != nil {
		logger.WithField("error", err).Error("failed to parse structured output")
		return errors.Wrap(errors.ErrUnknown, "failed to parse structured output", err)
	}

	logger.Debug("structured generation completed")

	return nil
}

// buildGenerateOptions builds genkit generate options from request
func (c *Client) buildGenerateOptions(req *GenerateRequest) []ai.GenerateOption {
	opts := []ai.GenerateOption{
		ai.WithModelName(c.getModelName(req)),
		ai.WithMessages(convertToAIMessages(req.Messages)...),
	}

	// Merge global config with request config
	mergedConfig := make(map[string]interface{})

	// Start with client global config
	for k, v := range c.config {
		mergedConfig[k] = v
	}

	// Override with request config
	if req.Config != nil {
		for k, v := range req.Config {
			mergedConfig[k] = v
		}
	}

	// Add temperature if specified
	if req.Temperature != nil {
		mergedConfig["temperature"] = *req.Temperature
	}

	// Add max tokens if specified
	if req.MaxTokens != nil {
		mergedConfig["maxOutputTokens"] = *req.MaxTokens
	}

	// Add top_p if specified
	if req.TopP != nil {
		mergedConfig["topP"] = *req.TopP
	}

	// Add top_k if specified
	if req.TopK != nil {
		mergedConfig["topK"] = *req.TopK
	}

	// Apply merged config if not empty
	if len(mergedConfig) > 0 {
		opts = append(opts, ai.WithConfig(mergedConfig))
	}

	return opts
}

// getModelName returns the model name from request or default
func (c *Client) getModelName(req *GenerateRequest) string {
	if req.Model != "" {
		return req.Model
	}
	return c.defaultModel
}

// SetDefaultModel sets the default model for the client
func (c *Client) SetDefaultModel(model string) {
	c.defaultModel = model
	logger.WithField("model", model).Info("default model updated")
}

// GetDefaultModel returns the current default model
func (c *Client) GetDefaultModel() string {
	return c.defaultModel
}

// SetAPIKey sets the API key for the client
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
	c.config["apiKey"] = apiKey
	logger.Info("API key updated")
}

// SetBaseURL sets the base URL for the client
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
	c.config["baseUrl"] = baseURL
	logger.WithField("baseURL", baseURL).Info("base URL updated")
}

// SetConfig sets a custom configuration parameter
func (c *Client) SetConfig(key string, value interface{}) {
	c.config[key] = value
	logger.WithFields(map[string]interface{}{
		"key":   key,
		"value": value,
	}).Debug("config parameter set")
}

// GetConfig returns a configuration parameter
func (c *Client) GetConfig(key string) (interface{}, bool) {
	val, ok := c.config[key]
	return val, ok
}

// GetAllConfig returns all configuration parameters
func (c *Client) GetAllConfig() map[string]interface{} {
	configCopy := make(map[string]interface{})
	for k, v := range c.config {
		configCopy[k] = v
	}
	return configCopy
}
