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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GeminiRequest represents a Gemini API request
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	Tools            []GeminiTool            `json:"tools,omitempty"`
	SafetySettings   []GeminiSafety          `json:"safetySettings,omitempty"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

// GeminiContent represents content in a Gemini request
type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of content
type GeminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

// GeminiFunctionCall represents a function call
type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// GeminiFunctionResponse represents a function response
type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

// GeminiTool represents a tool definition
type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

// GeminiFunctionDeclaration represents a function declaration
type GeminiFunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// GeminiGenerationConfig represents generation configuration
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GeminiSafety represents safety settings
type GeminiSafety struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiResponse represents a Gemini API response
type GeminiResponse struct {
	Candidates     []GeminiCandidate     `json:"candidates"`
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *GeminiUsageMetadata  `json:"usageMetadata,omitempty"`
}

// GeminiCandidate represents a candidate response
type GeminiCandidate struct {
	Content       GeminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason"`
	Index         int                  `json:"index"`
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings,omitempty"`
}

// GeminiSafetyRating represents a safety rating
type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// GeminiPromptFeedback represents prompt feedback
type GeminiPromptFeedback struct {
	BlockReason   string               `json:"blockReason,omitempty"`
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings,omitempty"`
}

// GeminiUsageMetadata represents usage metadata
type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// GeminiClient implements the LLMClient interface for Gemini APIs
type GeminiClient struct {
	*BaseClient
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(config Config) *GeminiClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	return &GeminiClient{
		BaseClient: NewBaseClient(config),
	}
}

// Complete sends a completion request and returns the response
func (c *GeminiClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	c.LogRequest(req)

	c.PrepareStructuredPrompt(req)
	c.PrepareToolPrompt(req)

	geminiReq := c.convertToGeminiRequest(req)

	headers := c.PrepareHeaders()

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.config.BaseURL, req.Model, c.config.APIKey)

	respBytes, err := c.httpClient.Post(ctx, url, headers, geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	response := c.convertFromGeminiResponse(&geminiResp)

	c.LogResponse(response)

	return response, nil
}

// CompleteStream sends a completion request and returns a stream of responses
func (c *GeminiClient) CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)

		c.LogRequest(req)

		c.PrepareStructuredPrompt(req)
		c.PrepareToolPrompt(req)

		geminiReq := c.convertToGeminiRequest(req)

		headers := c.PrepareHeaders()

		url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s", c.config.BaseURL, req.Model, c.config.APIKey)

		stream, err := c.httpClient.PostStream(ctx, url, headers, geminiReq)
		if err != nil {
			errorChan <- fmt.Errorf("failed to send streaming request: %w", err)
			return
		}
		defer stream.Close()

		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				data = strings.TrimSpace(data)

				if data == "[DONE]" {
					break
				}

				var geminiResp GeminiResponse
				if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
					c.logger.WithError(err).WithField("data", data).Warn("Failed to parse stream chunk")
					continue
				}

				chunk := c.convertFromGeminiStreamResponse(&geminiResp)

				select {
				case chunkChan <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errorChan <- fmt.Errorf("error reading stream: %w", err)
		}
	}()

	return chunkChan, errorChan
}

// CompleteStructured sends a completion request expecting structured JSON response
func (c *GeminiClient) CompleteStructured(ctx context.Context, req *CompletionRequest, result interface{}) error {
	req.StructuredOutput = true

	response, err := c.Complete(ctx, req)
	if err != nil {
		return err
	}

	if len(response.Choices) == 0 {
		return fmt.Errorf("no choices in response")
	}

	content := response.Choices[0].Message.Content
	return c.ParseStructuredResponse(content, result)
}

// CompleteStreamStructured sends a completion request expecting structured JSON response via stream
func (c *GeminiClient) CompleteStreamStructured(ctx context.Context, req *CompletionRequest, result interface{}) error {
	req.StructuredOutput = true

	chunkChan, errorChan := c.CompleteStream(ctx, req)

	var contentBuilder strings.Builder

	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				content := contentBuilder.String()
				if content == "" {
					return fmt.Errorf("no content received from stream")
				}
				return c.ParseStructuredResponse(content, result)
			}

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
				contentBuilder.WriteString(chunk.Choices[0].Delta.Content)
			}

		case err := <-errorChan:
			if err != nil {
				return err
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// convertToGeminiRequest converts a generic request to Gemini format
func (c *GeminiClient) convertToGeminiRequest(req *CompletionRequest) *GeminiRequest {
	geminiReq := &GeminiRequest{
		GenerationConfig: &GeminiGenerationConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			MaxOutputTokens: req.MaxTokens,
		},
	}

	if req.MaxTokens == 0 {
		geminiReq.GenerationConfig.MaxOutputTokens = 1024
	}

	var contents []GeminiContent
	var systemPrompt string

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			systemPrompt = msg.Content
			continue
		}

		role := string(msg.Role)
		if role == "assistant" {
			role = "model"
		}

		parts := []GeminiPart{
			{
				Text: msg.Content,
			},
		}

		if len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				var args map[string]interface{}
				if toolCall.Function.Arguments != "" {
					json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				}

				parts = append(parts, GeminiPart{
					FunctionCall: &GeminiFunctionCall{
						Name: toolCall.Function.Name,
						Args: args,
					},
				})
			}
		}

		contents = append(contents, GeminiContent{
			Role:  role,
			Parts: parts,
		})
	}

	if req.SystemPrompt != "" {
		if systemPrompt != "" {
			systemPrompt += "\n\n" + req.SystemPrompt
		} else {
			systemPrompt = req.SystemPrompt
		}
	}

	if systemPrompt != "" {
		systemContent := GeminiContent{
			Role: "user",
			Parts: []GeminiPart{
				{
					Text: systemPrompt,
				},
			},
		}
		contents = append([]GeminiContent{systemContent}, contents...)
	}

	geminiReq.Contents = contents

	if len(req.Tools) > 0 {
		var declarations []GeminiFunctionDeclaration
		for _, tool := range req.Tools {
			declarations = append(declarations, GeminiFunctionDeclaration{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			})
		}
		geminiReq.Tools = []GeminiTool{
			{
				FunctionDeclarations: declarations,
			},
		}
	}

	return geminiReq
}

// convertFromGeminiResponse converts Gemini response to generic format
func (c *GeminiClient) convertFromGeminiResponse(resp *GeminiResponse) *CompletionResponse {
	var choices []Choice

	for i, candidate := range resp.Candidates {
		var content string
		var toolCalls []ToolCall

		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				content += part.Text
			}

			if part.FunctionCall != nil {
				args, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, ToolCall{
					ID:   fmt.Sprintf("call_%d", i),
					Type: "function",
					Function: ToolCallFunction{
						Name:      part.FunctionCall.Name,
						Arguments: string(args),
					},
				})
			}
		}

		message := Message{
			Role:      RoleAssistant,
			Content:   content,
			ToolCalls: toolCalls,
		}

		choices = append(choices, Choice{
			Index:        candidate.Index,
			Message:      message,
			FinishReason: candidate.FinishReason,
		})
	}

	usage := Usage{}
	if resp.UsageMetadata != nil {
		usage = Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return &CompletionResponse{
		ID:      "gemini-response",
		Object:  "chat.completion",
		Created: 0,
		Model:   "",
		Choices: choices,
		Usage:   usage,
	}
}

// convertFromGeminiStreamResponse converts Gemini stream response to generic format
func (c *GeminiClient) convertFromGeminiStreamResponse(resp *GeminiResponse) StreamChunk {
	var choices []Choice

	for i, candidate := range resp.Candidates {
		var content string

		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				content += part.Text
			}
		}

		choices = append(choices, Choice{
			Index: i,
			Delta: &Message{
				Role:    RoleAssistant,
				Content: content,
			},
			FinishReason: candidate.FinishReason,
		})
	}

	return StreamChunk{
		ID:      "gemini-stream",
		Object:  "chat.completion.chunk",
		Created: 0,
		Model:   "",
		Choices: choices,
	}
}
