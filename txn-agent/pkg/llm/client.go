package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"seata-go-ai-txn-agent/pkg/utils"
)

// BaseClient provides common functionality for all LLM clients
type BaseClient struct {
	config     Config
	httpClient *utils.RetryHTTPClient
	logger     *utils.Logger
}

// NewBaseClient creates a new base client
func NewBaseClient(config Config) *BaseClient {
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	maxRetries := config.MaxRetry
	if maxRetries == 0 {
		maxRetries = 3
	}

	httpClient := utils.NewRetryHTTPClient(timeout, maxRetries, 2*time.Second)

	logger := utils.GetLogger("llm")

	return &BaseClient{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
	}
}

// GetConfig returns the client configuration
func (c *BaseClient) GetConfig() Config {
	return c.config
}

// PrepareHeaders prepares common headers for requests
func (c *BaseClient) PrepareHeaders() map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	if c.config.UserAgent != "" {
		headers["User-Agent"] = c.config.UserAgent
	}

	return headers
}

// AddSystemPrompt adds system prompt to messages if specified
func (c *BaseClient) AddSystemPrompt(req *CompletionRequest) {
	if req.SystemPrompt != "" {
		systemMessage := Message{
			Role:    RoleSystem,
			Content: req.SystemPrompt,
		}
		req.Messages = append([]Message{systemMessage}, req.Messages...)
	}
}

// PrepareStructuredPrompt prepares the request for structured output
func (c *BaseClient) PrepareStructuredPrompt(req *CompletionRequest) {
	if req.StructuredOutput {
		structuredPrompt := `
CRITICAL INSTRUCTION: You MUST respond with ONLY valid JSON. No explanations, no markdown, no code blocks, no text before or after.
Your entire response should be parseable JSON starting with { and ending with }.
Do NOT include any Chinese characters outside of JSON string values.
Do NOT add any explanatory text.
Example of correct response format:
{"text": "分析内容", "graph": {...}, "seata_json": {...}, "phase": 1}

WRONG formats that will cause errors:
- Adding explanations before JSON
- Using markdown code blocks
- Including non-JSON text
- Multiple JSON objects`

		if req.SystemPrompt != "" {
			req.SystemPrompt += "\n\n" + structuredPrompt
		} else {
			req.SystemPrompt = structuredPrompt
		}
	}
}

// PrepareToolPrompt prepares the request for tool calling
func (c *BaseClient) PrepareToolPrompt(req *CompletionRequest) {
	if len(req.Tools) > 0 {
		toolPrompt := c.buildToolPrompt(req.Tools)

		if req.SystemPrompt != "" {
			req.SystemPrompt += "\n\n" + toolPrompt
		} else {
			req.SystemPrompt = toolPrompt
		}
	}
}

// buildToolPrompt creates a tool calling prompt for models that don't natively support tools
func (c *BaseClient) buildToolPrompt(tools []Tool) string {
	var toolsDesc strings.Builder
	toolsDesc.WriteString("You have access to the following tools:\n\n")

	for _, tool := range tools {
		toolsDesc.WriteString(fmt.Sprintf("Tool: %s\n", tool.Function.Name))
		toolsDesc.WriteString(fmt.Sprintf("Description: %s\n", tool.Function.Description))

		if tool.Function.Parameters != nil {
			paramBytes, _ := json.Marshal(tool.Function.Parameters)
			toolsDesc.WriteString(fmt.Sprintf("Parameters: %s\n\n", string(paramBytes)))
		}
	}

	toolsDesc.WriteString(`
When you need to use a tool, respond with a JSON object in this exact format:
{
  "tool_calls": [
    {
      "id": "call_123",
      "type": "function",
      "function": {
        "name": "tool_name",
        "arguments": "{\"param1\": \"value1\"}"
      }
    }
  ]
}

If you don't need to use any tools, respond normally with text.`)

	return toolsDesc.String()
}

// ParseStructuredResponse parses a structured response into the result interface
func (c *BaseClient) ParseStructuredResponse(response string, result interface{}) error {
	originalResponse := response
	cleaned := strings.TrimSpace(response)

	// Log the original response for debugging
	c.logger.WithField("original_response", originalResponse).Debug("Parsing structured response")

	// Remove any BOM (Byte Order Mark) characters
	if strings.HasPrefix(cleaned, "\ufeff") {
		cleaned = cleaned[3:] // UTF-8 BOM is 3 bytes
	}

	// Try to extract JSON from markdown code blocks
	if strings.Contains(cleaned, "```json") {
		start := strings.Index(cleaned, "```json")
		if start != -1 {
			jsonStart := start + 7 // length of "```json"
			remaining := cleaned[jsonStart:]
			end := strings.Index(remaining, "```")
			if end != -1 {
				cleaned = strings.TrimSpace(remaining[:end])
			}
		}
	} else if strings.Contains(cleaned, "```") {
		// Handle generic code blocks
		start := strings.Index(cleaned, "```")
		if start != -1 {
			lines := strings.Split(cleaned[start:], "\n")
			if len(lines) > 1 {
				jsonStart := start + len(lines[0]) + 1 // Skip first line with ```
				remaining := cleaned[jsonStart:]
				end := strings.Index(remaining, "```")
				if end != -1 {
					cleaned = strings.TrimSpace(remaining[:end])
				}
			}
		}
	}

	// Try to find JSON object boundaries if still mixed with text
	if !strings.HasPrefix(cleaned, "{") && !strings.HasPrefix(cleaned, "[") {
		if start := strings.Index(cleaned, "{"); start != -1 {
			cleaned = cleaned[start:]
		} else if start := strings.Index(cleaned, "["); start != -1 {
			cleaned = cleaned[start:]
		} else {
			return fmt.Errorf("no valid JSON found in response. Response preview: %s", truncateString(originalResponse, 200))
		}
	}

	// Find the end of the JSON object/array
	if strings.HasPrefix(cleaned, "{") {
		braceCount := 0
		for i, char := range cleaned {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					cleaned = cleaned[:i+1]
					break
				}
			}
		}
	} else if strings.HasPrefix(cleaned, "[") {
		bracketCount := 0
		for i, char := range cleaned {
			if char == '[' {
				bracketCount++
			} else if char == ']' {
				bracketCount--
				if bracketCount == 0 {
					cleaned = cleaned[:i+1]
					break
				}
			}
		}
	}

	cleaned = strings.TrimSpace(cleaned)

	// Validate that we have valid JSON structure
	if !strings.HasPrefix(cleaned, "{") && !strings.HasPrefix(cleaned, "[") {
		return fmt.Errorf("extracted content is not valid JSON. Content: %s", truncateString(cleaned, 200))
	}

	// Log cleaned JSON for debugging
	c.logger.WithField("cleaned_json", cleaned).Debug("Attempting to parse JSON")

	if err := json.Unmarshal([]byte(cleaned), result); err != nil {
		c.logger.WithError(err).WithFields(map[string]interface{}{
			"original_response": truncateString(originalResponse, 500),
			"cleaned_json":      truncateString(cleaned, 500),
		}).Error("Failed to parse structured response")
		return fmt.Errorf("failed to parse structured response: %w. Cleaned JSON: %s", err, truncateString(cleaned, 200))
	}

	return nil
}

// truncateString truncates a string to a maximum length for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ParseToolCalls parses tool calls from response content
func (c *BaseClient) ParseToolCalls(content string) ([]ToolCall, error) {
	var toolCallResponse struct {
		ToolCalls []ToolCall `json:"tool_calls"`
	}

	if err := json.Unmarshal([]byte(content), &toolCallResponse); err != nil {
		return nil, nil
	}

	if len(toolCallResponse.ToolCalls) == 0 {
		return nil, nil
	}

	for i := range toolCallResponse.ToolCalls {
		if toolCallResponse.ToolCalls[i].ID == "" {
			toolCallResponse.ToolCalls[i].ID = fmt.Sprintf("call_%d", i)
		}
		if toolCallResponse.ToolCalls[i].Type == "" {
			toolCallResponse.ToolCalls[i].Type = "function"
		}
	}

	return toolCallResponse.ToolCalls, nil
}

// LogRequest logs the request for debugging
func (c *BaseClient) LogRequest(req *CompletionRequest) {
	c.logger.WithFields(map[string]interface{}{
		"model":       req.Model,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      req.Stream,
		"tools_count": len(req.Tools),
		"messages":    len(req.Messages),
	}).Debug("Sending LLM request")
}

// LogResponse logs the response for debugging
func (c *BaseClient) LogResponse(resp *CompletionResponse) {
	if resp.Usage.TotalTokens > 0 {
		c.logger.WithFields(map[string]interface{}{
			"model":             resp.Model,
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		}).Debug("Received LLM response")
	}
}
