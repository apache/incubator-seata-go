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
	"encoding/json"
	"fmt"
	"strings"
)

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Input          []string `json:"input"`
	Model          string   `json:"model"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
	Dimensions     int      `json:"dimensions,omitempty"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  EmbeddingUsage  `json:"usage"`
}

// EmbeddingData represents individual embedding data
type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// EmbeddingUsage represents token usage for embeddings
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingClient interface for generating embeddings
type EmbeddingClient interface {
	// GenerateEmbedding creates embeddings for the given texts
	GenerateEmbedding(ctx context.Context, texts []string) (*EmbeddingResponse, error)

	// GetDimension returns the dimension of the embedding vectors
	GetDimension() int

	// GetModel returns the embedding model name
	GetModel() string
}

// OpenAIEmbeddingClient implements EmbeddingClient for OpenAI-compatible APIs
type OpenAIEmbeddingClient struct {
	*BaseClient
	model      string
	dimensions int
}

// NewOpenAIEmbeddingClient creates a new OpenAI embedding client
func NewOpenAIEmbeddingClient(config Config, model string) *OpenAIEmbeddingClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}

	// Set default model and dimensions
	if model == "" {
		model = "text-embedding-ada-002"
	}

	dimensions := 1536 // Default for text-embedding-ada-002
	switch model {
	case "text-embedding-3-small":
		dimensions = 1536
	case "text-embedding-3-large":
		dimensions = 3072
	case "text-embedding-ada-002":
		dimensions = 1536
	}

	return &OpenAIEmbeddingClient{
		BaseClient: NewBaseClient(config),
		model:      model,
		dimensions: dimensions,
	}
}

// GenerateEmbedding creates embeddings for the given texts
func (c *OpenAIEmbeddingClient) GenerateEmbedding(ctx context.Context, texts []string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Object: "list",
			Data:   []EmbeddingData{},
			Model:  c.model,
		}, nil
	}

	headers := c.PrepareHeaders()
	headers["Authorization"] = fmt.Sprintf("Bearer %s", c.config.APIKey)

	request := EmbeddingRequest{
		Input:          texts,
		Model:          c.model,
		EncodingFormat: "float",
	}

	url := fmt.Sprintf("%s/embeddings", c.config.BaseURL)

	respBytes, err := c.httpClient.Post(ctx, url, headers, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send embedding request: %w", err)
	}

	var response EmbeddingResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to parse embedding response: %w", err)
	}

	return &response, nil
}

// GetDimension returns the dimension of the embedding vectors
func (c *OpenAIEmbeddingClient) GetDimension() int {
	return c.dimensions
}

// GetModel returns the embedding model name
func (c *OpenAIEmbeddingClient) GetModel() string {
	return c.model
}

// AnthropicEmbeddingClient implements EmbeddingClient for Anthropic APIs
type AnthropicEmbeddingClient struct {
	*BaseClient
	model      string
	dimensions int
}

// NewAnthropicEmbeddingClient creates a new Anthropic embedding client
func NewAnthropicEmbeddingClient(config Config, model string) *AnthropicEmbeddingClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com/v1"
	}

	if model == "" {
		model = "claude-3-haiku"
	}

	// Anthropic doesn't have dedicated embedding models yet,
	// this is a placeholder implementation
	dimensions := 1536

	return &AnthropicEmbeddingClient{
		BaseClient: NewBaseClient(config),
		model:      model,
		dimensions: dimensions,
	}
}

// GenerateEmbedding creates embeddings using Anthropic's approach
func (c *AnthropicEmbeddingClient) GenerateEmbedding(ctx context.Context, texts []string) (*EmbeddingResponse, error) {
	// Note: Anthropic doesn't currently provide embedding APIs
	// This is a placeholder that would need to be implemented
	// when Anthropic releases embedding endpoints
	return nil, fmt.Errorf("anthropic embedding API not yet available")
}

// GetDimension returns the dimension of the embedding vectors
func (c *AnthropicEmbeddingClient) GetDimension() int {
	return c.dimensions
}

// GetModel returns the embedding model name
func (c *AnthropicEmbeddingClient) GetModel() string {
	return c.model
}

// GeminiEmbeddingClient implements EmbeddingClient for Gemini APIs
type GeminiEmbeddingClient struct {
	*BaseClient
	model      string
	dimensions int
}

// GeminiEmbeddingRequest represents a Gemini embedding request
type GeminiEmbeddingRequest struct {
	Content Content `json:"content"`
	Model   string  `json:"model,omitempty"`
}

// Content represents Gemini embedding content
type Content struct {
	Parts []Part `json:"parts"`
}

// Part represents a part of Gemini content
type Part struct {
	Text string `json:"text"`
}

// GeminiEmbeddingResponse represents a Gemini embedding response
type GeminiEmbeddingResponse struct {
	Embedding GeminiEmbedding `json:"embedding"`
}

// GeminiEmbedding represents Gemini embedding data
type GeminiEmbedding struct {
	Values []float64 `json:"values"`
}

// NewGeminiEmbeddingClient creates a new Gemini embedding client
func NewGeminiEmbeddingClient(config Config, model string) *GeminiEmbeddingClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	if model == "" {
		model = "embedding-001"
	}

	dimensions := 768 // Default for Gemini embedding models

	return &GeminiEmbeddingClient{
		BaseClient: NewBaseClient(config),
		model:      model,
		dimensions: dimensions,
	}
}

// GenerateEmbedding creates embeddings for the given texts using Gemini API
func (c *GeminiEmbeddingClient) GenerateEmbedding(ctx context.Context, texts []string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Object: "list",
			Data:   []EmbeddingData{},
			Model:  c.model,
		}, nil
	}

	headers := c.PrepareHeaders()

	var embeddingData []EmbeddingData

	// Gemini embedding API processes one text at a time
	for i, text := range texts {
		request := GeminiEmbeddingRequest{
			Content: Content{
				Parts: []Part{{Text: text}},
			},
		}

		url := fmt.Sprintf("%s/models/%s:embedContent?key=%s", c.config.BaseURL, c.model, c.config.APIKey)

		respBytes, err := c.httpClient.Post(ctx, url, headers, request)
		if err != nil {
			return nil, fmt.Errorf("failed to send embedding request for text %d: %w", i, err)
		}

		var geminiResp GeminiEmbeddingResponse
		if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
			return nil, fmt.Errorf("failed to parse embedding response for text %d: %w", i, err)
		}

		embeddingData = append(embeddingData, EmbeddingData{
			Object:    "embedding",
			Index:     i,
			Embedding: geminiResp.Embedding.Values,
		})
	}

	return &EmbeddingResponse{
		Object: "list",
		Data:   embeddingData,
		Model:  c.model,
		Usage: EmbeddingUsage{
			PromptTokens: len(strings.Join(texts, " ")),
			TotalTokens:  len(strings.Join(texts, " ")),
		},
	}, nil
}

// GetDimension returns the dimension of the embedding vectors
func (c *GeminiEmbeddingClient) GetDimension() int {
	return c.dimensions
}

// GetModel returns the embedding model name
func (c *GeminiEmbeddingClient) GetModel() string {
	return c.model
}

// NewEmbeddingClient creates a new embedding client based on the specified type
func NewEmbeddingClient(clientType ClientType, config Config, model string) (EmbeddingClient, error) {
	switch clientType {
	case ClientTypeOpenAI:
		return NewOpenAIEmbeddingClient(config, model), nil
	case ClientTypeAnthropic:
		return NewAnthropicEmbeddingClient(config, model), nil
	case ClientTypeGemini:
		return NewGeminiEmbeddingClient(config, model), nil
	default:
		return nil, fmt.Errorf("unsupported embedding client type: %s", clientType)
	}
}

// BatchEmbeddingClient provides efficient batch embedding processing
type BatchEmbeddingClient struct {
	client    EmbeddingClient
	batchSize int
}

// NewBatchEmbeddingClient creates a new batch embedding client
func NewBatchEmbeddingClient(client EmbeddingClient, batchSize int) *BatchEmbeddingClient {
	if batchSize <= 0 {
		batchSize = 10
	}

	return &BatchEmbeddingClient{
		client:    client,
		batchSize: batchSize,
	}
}

// GenerateEmbedding processes texts in batches for efficiency
func (b *BatchEmbeddingClient) GenerateEmbedding(ctx context.Context, texts []string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Object: "list",
			Data:   []EmbeddingData{},
			Model:  b.client.GetModel(),
		}, nil
	}

	var allEmbeddingData []EmbeddingData
	var totalUsage EmbeddingUsage

	// Process texts in batches
	for i := 0; i < len(texts); i += b.batchSize {
		end := i + b.batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		response, err := b.client.GenerateEmbedding(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to process batch %d-%d: %w", i, end-1, err)
		}

		// Adjust indices for the overall response
		for j := range response.Data {
			response.Data[j].Index = i + j
		}

		allEmbeddingData = append(allEmbeddingData, response.Data...)
		totalUsage.PromptTokens += response.Usage.PromptTokens
		totalUsage.TotalTokens += response.Usage.TotalTokens
	}

	return &EmbeddingResponse{
		Object: "list",
		Data:   allEmbeddingData,
		Model:  b.client.GetModel(),
		Usage:  totalUsage,
	}, nil
}

// GetDimension returns the dimension of the embedding vectors
func (b *BatchEmbeddingClient) GetDimension() int {
	return b.client.GetDimension()
}

// GetModel returns the embedding model name
func (b *BatchEmbeddingClient) GetModel() string {
	return b.client.GetModel()
}
