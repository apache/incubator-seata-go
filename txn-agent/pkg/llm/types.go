package llm

import (
	"context"
)

// MessageRole represents the role of a message
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// Message represents a conversation message
type Message struct {
	Role       MessageRole `json:"role"`
	Content    string      `json:"content"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction represents a tool call function
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool represents a tool definition
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction represents a tool function definition
type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// CompletionRequest represents a completion request
type CompletionRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	MaxTokens        int       `json:"max_tokens,omitempty"`
	Temperature      float64   `json:"temperature,omitempty"`
	TopP             float64   `json:"top_p,omitempty"`
	Stream           bool      `json:"stream,omitempty"`
	Tools            []Tool    `json:"tools,omitempty"`
	ToolChoice       string    `json:"tool_choice,omitempty"`
	ResponseFormat   *Format   `json:"response_format,omitempty"`
	SystemPrompt     string    `json:"-"`
	StructuredOutput bool      `json:"-"`
}

// Format represents response format
type Format struct {
	Type string `json:"type"`
}

// CompletionResponse represents a completion response
type CompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int         `json:"index"`
	Message      Message     `json:"message"`
	Delta        *Message    `json:"delta,omitempty"`
	FinishReason string      `json:"finish_reason"`
	LogProbs     interface{} `json:"logprobs,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Config represents LLM client configuration
type Config struct {
	APIKey    string
	BaseURL   string
	Model     string
	Timeout   int
	MaxRetry  int
	UserAgent string
}

// LLMClient interface defines the contract for LLM clients
type LLMClient interface {
	// Complete sends a completion request and returns the response
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// CompleteStream sends a completion request and returns a stream of responses
	CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, <-chan error)

	// CompleteStructured sends a completion request expecting structured JSON response
	CompleteStructured(ctx context.Context, req *CompletionRequest, result interface{}) error

	// CompleteStreamStructured sends a completion request expecting structured JSON response via stream
	CompleteStreamStructured(ctx context.Context, req *CompletionRequest, result interface{}) error

	// GetConfig returns the client configuration
	GetConfig() Config
}

// ClientType represents the type of LLM client
type ClientType string

const (
	ClientTypeOpenAI    ClientType = "openai"
	ClientTypeAnthropic ClientType = "anthropic"
	ClientTypeGemini    ClientType = "gemini"
)

// StreamReader interface for handling streaming responses
type StreamReader interface {
	Read() (*StreamChunk, error)
	Close() error
}

// ToolExecutor interface for executing tools
type ToolExecutor interface {
	Execute(ctx context.Context, toolCall ToolCall) (string, error)
}

// ToolRegistry interface for managing tools
type ToolRegistry interface {
	Register(tool Tool, executor ToolExecutor)
	GetTool(name string) (Tool, bool)
	GetExecutor(name string) (ToolExecutor, bool)
	ListTools() []Tool
}
