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

// OpenAIClient implements the LLMClient interface for OpenAI-compatible APIs
type OpenAIClient struct {
	*BaseClient
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config Config) *OpenAIClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}

	return &OpenAIClient{
		BaseClient: NewBaseClient(config),
	}
}

// Complete sends a completion request and returns the response
func (c *OpenAIClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	c.LogRequest(req)

	c.AddSystemPrompt(req)
	c.PrepareStructuredPrompt(req)
	c.PrepareToolPrompt(req)

	headers := c.PrepareHeaders()
	headers["Authorization"] = fmt.Sprintf("Bearer %s", c.config.APIKey)

	req.Stream = false

	url := fmt.Sprintf("%s/chat/completions", c.config.BaseURL)

	respBytes, err := c.httpClient.Post(ctx, url, headers, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var response CompletionResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) > 0 {
		choice := &response.Choices[0]
		if toolCalls, err := c.ParseToolCalls(choice.Message.Content); err == nil && len(toolCalls) > 0 {
			choice.Message.ToolCalls = toolCalls
			choice.Message.Content = ""
		}
	}

	c.LogResponse(&response)

	return &response, nil
}

// CompleteStream sends a completion request and returns a stream of responses
func (c *OpenAIClient) CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)

		c.LogRequest(req)

		c.AddSystemPrompt(req)
		c.PrepareStructuredPrompt(req)
		c.PrepareToolPrompt(req)

		headers := c.PrepareHeaders()
		headers["Authorization"] = fmt.Sprintf("Bearer %s", c.config.APIKey)

		req.Stream = true

		url := fmt.Sprintf("%s/chat/completions", c.config.BaseURL)

		stream, err := c.httpClient.PostStream(ctx, url, headers, req)
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

				var chunk StreamChunk
				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					c.logger.WithError(err).WithField("data", data).Warn("Failed to parse stream chunk")
					continue
				}

				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
					if toolCalls, err := c.ParseToolCalls(chunk.Choices[0].Delta.Content); err == nil && len(toolCalls) > 0 {
						chunk.Choices[0].Delta.ToolCalls = toolCalls
						chunk.Choices[0].Delta.Content = ""
					}
				}

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
func (c *OpenAIClient) CompleteStructured(ctx context.Context, req *CompletionRequest, result interface{}) error {
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
func (c *OpenAIClient) CompleteStreamStructured(ctx context.Context, req *CompletionRequest, result interface{}) error {
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
