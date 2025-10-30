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
	"github.com/firebase/genkit/go/ai"
)

// Message represents a single message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Tool represents a tool that can be called by the model
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool invocation request from the model
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string                 `json:"tool_call_id"`
	Output     map[string]interface{} `json:"output"`
}

// GenerateRequest represents a request to generate content
type GenerateRequest struct {
	Messages       []Message              `json:"messages"`
	Model          string                 `json:"model,omitempty"`
	Temperature    *float32               `json:"temperature,omitempty"`
	MaxTokens      *int                   `json:"max_tokens,omitempty"`
	TopP           *float32               `json:"top_p,omitempty"`
	TopK           *int                   `json:"top_k,omitempty"`
	Stream         bool                   `json:"stream"`
	Tools          []Tool                 `json:"tools,omitempty"`
	ToolChoice     string                 `json:"tool_choice,omitempty"`
	ResponseFormat *ResponseFormat        `json:"response_format,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
}

// ResponseFormat specifies the format of the model response
type ResponseFormat struct {
	Type   string                 `json:"type"` // "text", "json_object", "json_schema"
	Schema map[string]interface{} `json:"schema,omitempty"`
}

// GenerateResponse represents a response from the model
type GenerateResponse struct {
	Content      string      `json:"content"`
	ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
	FinishReason string      `json:"finish_reason"`
	Usage        *UsageStats `json:"usage,omitempty"`
	RawResponse  interface{} `json:"raw_response,omitempty"`
}

// UsageStats contains token usage information
type UsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamCallback is called for each chunk in streaming mode
type StreamCallback func(chunk string) error

// convertToAIMessages converts messages to genkit ai.Message format
func convertToAIMessages(messages []Message) []*ai.Message {
	aiMessages := make([]*ai.Message, 0, len(messages))
	for _, msg := range messages {
		var role ai.Role
		switch msg.Role {
		case "user":
			role = ai.RoleUser
		case "model", "assistant":
			role = ai.RoleModel
		case "system":
			role = ai.RoleSystem
		default:
			role = ai.RoleUser
		}

		aiMessages = append(aiMessages, &ai.Message{
			Role:    role,
			Content: []*ai.Part{ai.NewTextPart(msg.Content)},
		})
	}
	return aiMessages
}

// extractTextFromResponse extracts text content from genkit response
func extractTextFromResponse(resp *ai.ModelResponse) string {
	if resp == nil {
		return ""
	}
	return resp.Text()
}

// extractToolCallsFromResponse extracts tool calls from genkit response
func extractToolCallsFromResponse(resp *ai.ModelResponse) []ToolCall {
	if resp == nil {
		return nil
	}

	toolRequests := resp.ToolRequests()
	if len(toolRequests) == 0 {
		return nil
	}

	toolCalls := make([]ToolCall, 0, len(toolRequests))
	for _, tr := range toolRequests {
		args, ok := tr.Input.(map[string]interface{})
		if !ok {
			args = make(map[string]interface{})
		}
		toolCalls = append(toolCalls, ToolCall{
			ID:        tr.Name,
			Name:      tr.Name,
			Arguments: args,
		})
	}
	return toolCalls
}

// extractUsageStats extracts usage statistics from genkit response
func extractUsageStats(resp *ai.ModelResponse) *UsageStats {
	if resp == nil || resp.Usage == nil {
		return nil
	}

	return &UsageStats{
		PromptTokens:     resp.Usage.InputTokens,
		CompletionTokens: resp.Usage.OutputTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
}

// extractFinishReason extracts finish reason from genkit response
func extractFinishReason(resp *ai.ModelResponse) string {
	if resp == nil {
		return "unknown"
	}

	switch resp.FinishReason {
	case ai.FinishReasonStop:
		return "stop"
	case ai.FinishReasonLength:
		return "length"
	case ai.FinishReasonBlocked:
		return "content_filter"
	case ai.FinishReasonOther:
		return "other"
	default:
		return "unknown"
	}
}
