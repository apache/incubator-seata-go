package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// AnthropicRequest represents an Anthropic API request
type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []AnthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
	Tools       []Tool             `json:"tools,omitempty"`
	ToolChoice  interface{}        `json:"tool_choice,omitempty"`
}

// AnthropicMessage represents an Anthropic message
type AnthropicMessage struct {
	Role    string                  `json:"role"`
	Content []AnthropicContentBlock `json:"content"`
}

// AnthropicContentBlock represents a content block
type AnthropicContentBlock struct {
	Type       string               `json:"type"`
	Text       string               `json:"text,omitempty"`
	ToolUse    *AnthropicToolUse    `json:"tool_use,omitempty"`
	ToolResult *AnthropicToolResult `json:"tool_result,omitempty"`
}

// AnthropicToolUse represents a tool use block
type AnthropicToolUse struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

// AnthropicToolResult represents a tool result block
type AnthropicToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

// AnthropicResponse represents an Anthropic API response
type AnthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"`
	StopSequence string                  `json:"stop_sequence"`
	Usage        AnthropicUsage          `json:"usage"`
}

// AnthropicUsage represents token usage
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamChunk represents a streaming response chunk
type AnthropicStreamChunk struct {
	Type    string                  `json:"type"`
	Index   int                     `json:"index,omitempty"`
	Delta   *AnthropicContentBlock  `json:"delta,omitempty"`
	Content []AnthropicContentBlock `json:"content,omitempty"`
}

// AnthropicClient implements the LLMClient interface for Anthropic APIs
type AnthropicClient struct {
	*BaseClient
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(config Config) *AnthropicClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com/v1"
	}

	return &AnthropicClient{
		BaseClient: NewBaseClient(config),
	}
}

// Complete sends a completion request and returns the response
func (c *AnthropicClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	c.LogRequest(req)

	c.PrepareStructuredPrompt(req)
	c.PrepareToolPrompt(req)

	anthropicReq := c.convertToAnthropicRequest(req)

	headers := c.PrepareHeaders()
	headers["x-api-key"] = c.config.APIKey
	headers["anthropic-version"] = "2023-06-01"

	url := fmt.Sprintf("%s/messages", c.config.BaseURL)

	respBytes, err := c.httpClient.Post(ctx, url, headers, anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(respBytes, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	response := c.convertFromAnthropicResponse(&anthropicResp)

	c.LogResponse(response)

	return response, nil
}

// CompleteStream sends a completion request and returns a stream of responses
func (c *AnthropicClient) CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)

		c.LogRequest(req)

		c.PrepareStructuredPrompt(req)
		c.PrepareToolPrompt(req)

		anthropicReq := c.convertToAnthropicRequest(req)
		anthropicReq.Stream = true

		headers := c.PrepareHeaders()
		headers["x-api-key"] = c.config.APIKey
		headers["anthropic-version"] = "2023-06-01"

		url := fmt.Sprintf("%s/messages", c.config.BaseURL)

		stream, err := c.httpClient.PostStream(ctx, url, headers, anthropicReq)
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

				var anthropicChunk AnthropicStreamChunk
				if err := json.Unmarshal([]byte(data), &anthropicChunk); err != nil {
					c.logger.WithError(err).WithField("data", data).Warn("Failed to parse stream chunk")
					continue
				}

				chunk := c.convertFromAnthropicStreamChunk(&anthropicChunk)

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
func (c *AnthropicClient) CompleteStructured(ctx context.Context, req *CompletionRequest, result interface{}) error {
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
func (c *AnthropicClient) CompleteStreamStructured(ctx context.Context, req *CompletionRequest, result interface{}) error {
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

// convertToAnthropicRequest converts a generic request to Anthropic format
func (c *AnthropicClient) convertToAnthropicRequest(req *CompletionRequest) *AnthropicRequest {
	anthropicReq := &AnthropicRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		Tools:       req.Tools,
	}

	if req.MaxTokens == 0 {
		anthropicReq.MaxTokens = 1024
	}

	var systemPrompt string
	var messages []AnthropicMessage

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			systemPrompt = msg.Content
			continue
		}

		anthropicMsg := AnthropicMessage{
			Role: string(msg.Role),
			Content: []AnthropicContentBlock{
				{
					Type: "text",
					Text: msg.Content,
				},
			},
		}

		if len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				var input interface{}
				if toolCall.Function.Arguments != "" {
					json.Unmarshal([]byte(toolCall.Function.Arguments), &input)
				}

				anthropicMsg.Content = append(anthropicMsg.Content, AnthropicContentBlock{
					Type: "tool_use",
					ToolUse: &AnthropicToolUse{
						ID:    toolCall.ID,
						Name:  toolCall.Function.Name,
						Input: input,
					},
				})
			}
		}

		messages = append(messages, anthropicMsg)
	}

	if req.SystemPrompt != "" {
		if systemPrompt != "" {
			systemPrompt += "\n\n" + req.SystemPrompt
		} else {
			systemPrompt = req.SystemPrompt
		}
	}

	anthropicReq.System = systemPrompt
	anthropicReq.Messages = messages

	return anthropicReq
}

// convertFromAnthropicResponse converts Anthropic response to generic format
func (c *AnthropicClient) convertFromAnthropicResponse(resp *AnthropicResponse) *CompletionResponse {
	var content string
	var toolCalls []ToolCall

	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		} else if block.Type == "tool_use" && block.ToolUse != nil {
			args, _ := json.Marshal(block.ToolUse.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ToolUse.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      block.ToolUse.Name,
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

	return &CompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: 0,
		Model:   resp.Model,
		Choices: []Choice{
			{
				Index:        0,
				Message:      message,
				FinishReason: resp.StopReason,
			},
		},
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// convertFromAnthropicStreamChunk converts Anthropic stream chunk to generic format
func (c *AnthropicClient) convertFromAnthropicStreamChunk(chunk *AnthropicStreamChunk) StreamChunk {
	var content string

	if chunk.Delta != nil && chunk.Delta.Type == "text" {
		content = chunk.Delta.Text
	}

	return StreamChunk{
		ID:      "anthropic-stream",
		Object:  "chat.completion.chunk",
		Created: 0,
		Model:   "",
		Choices: []Choice{
			{
				Index: chunk.Index,
				Delta: &Message{
					Role:    RoleAssistant,
					Content: content,
				},
				FinishReason: "",
			},
		},
	}
}
