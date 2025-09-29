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

package types

import (
	"context"
	"encoding/json"
	"time"
)

// AgentRequest represents a request to the agent
type AgentRequest struct {
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id"`
	Input       string                 `json:"input"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Tools       []string               `json:"tools,omitempty"`
	MaxSteps    int                    `json:"max_steps,omitempty"`
	Temperature float32                `json:"temperature,omitempty"`
	Model       string                 `json:"model,omitempty"`
}

// AgentResponse represents the agent's response
type AgentResponse struct {
	SessionID     string      `json:"session_id"`
	Response      string      `json:"response"`
	Steps         []Step      `json:"steps"`
	ToolCalls     []ToolCall  `json:"tool_calls,omitempty"`
	Status        Status      `json:"status"`
	Error         *AgentError `json:"error,omitempty"`
	Metadata      Metadata    `json:"metadata"`
	ExecutionTime Duration    `json:"execution_time"`
}

// Step represents a single reasoning step
type Step struct {
	ID        string                 `json:"id"`
	Type      StepType               `json:"type"`
	Input     string                 `json:"input"`
	Output    string                 `json:"output"`
	Reasoning string                 `json:"reasoning,omitempty"`
	ToolCall  *ToolCall              `json:"tool_call,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Input    map[string]interface{} `json:"input"`
	Output   interface{}            `json:"output,omitempty"`
	Error    *ToolError             `json:"error,omitempty"`
	Duration time.Duration          `json:"duration"`
}

// Tool represents an available tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]Parameter   `json:"parameters"`
	Handler     ToolHandler            `json:"-"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Parameter represents a tool parameter
type Parameter struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// ToolHandler defines the interface for tool execution
type ToolHandler func(ctx context.Context, input map[string]interface{}) (interface{}, error)

// AgentState represents the current state of agent execution
type AgentState struct {
	SessionID    string                 `json:"session_id"`
	CurrentStep  int                    `json:"current_step"`
	MaxSteps     int                    `json:"max_steps"`
	Status       Status                 `json:"status"`
	Steps        []Step                 `json:"steps"`
	Context      map[string]interface{} `json:"context"`
	Memory       []MemoryItem           `json:"memory"`
	StartTime    time.Time              `json:"start_time"`
	LastActivity time.Time              `json:"last_activity"`
}

// MemoryItem represents a piece of agent memory
type MemoryItem struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Type      MemoryType  `json:"type"`
	TTL       *time.Time  `json:"ttl,omitempty"`
	Metadata  Metadata    `json:"metadata,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

// StepType represents the type of reasoning step
type StepType string

const (
	StepTypeReasoning StepType = "reasoning"
	StepTypeAction    StepType = "action"
	StepTypeObserve   StepType = "observe"
	StepTypeFinal     StepType = "final"
)

// Status represents execution status
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusTimeout   Status = "timeout"
)

// MemoryType represents the type of memory item
type MemoryType string

const (
	MemoryTypeShort MemoryType = "short"
	MemoryTypeLong  MemoryType = "long"
	MemoryTypeWork  MemoryType = "working"
)

// AgentError represents an agent execution error
type AgentError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Retryable bool                   `json:"retryable"`
}

// ToolError represents a tool execution error
type ToolError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Retryable bool                   `json:"retryable"`
}

// Metadata holds arbitrary metadata
type Metadata map[string]interface{}

// Duration is a wrapper for time.Duration with JSON support
type Duration time.Duration

// MarshalJSON implements json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

// Agent represents the core agent interface
type Agent interface {
	// Process executes the agent workflow
	Process(ctx context.Context, req *AgentRequest) (*AgentResponse, error)

	// GetState returns current agent state
	GetState(ctx context.Context, sessionID string) (*AgentState, error)

	// Cancel cancels an ongoing execution
	Cancel(ctx context.Context, sessionID string) error

	// RegisterTool registers a new tool
	RegisterTool(tool *Tool) error

	// ListTools returns available tools
	ListTools() []*Tool

	// Close shuts down the agent
	Close() error
}

// Reasoner handles the reasoning logic
type Reasoner interface {
	// Reason performs reasoning based on current context
	Reason(ctx context.Context, input string, context map[string]interface{}) (*ReasoningResult, error)

	// ParseResponse parses LLM response into structured format
	ParseResponse(response string) (*ParsedResponse, error)
}

// ToolRegistry manages available tools
type ToolRegistry interface {
	// Register adds a tool to the registry
	Register(tool *Tool) error

	// Get retrieves a tool by name
	Get(name string) (*Tool, error)

	// List returns all registered tools
	List() []*Tool

	// Unregister removes a tool from the registry
	Unregister(name string) error

	// Execute runs a tool with given parameters
	Execute(ctx context.Context, name string, input map[string]interface{}) (interface{}, error)
}

// StateManager manages agent execution state
type StateManager interface {
	// SaveState persists agent state
	SaveState(ctx context.Context, state *AgentState) error

	// LoadState retrieves agent state
	LoadState(ctx context.Context, sessionID string) (*AgentState, error)

	// DeleteState removes agent state
	DeleteState(ctx context.Context, sessionID string) error

	// UpdateStep updates a specific step
	UpdateStep(ctx context.Context, sessionID string, step *Step) error
}

// ReasoningResult represents the result of reasoning
type ReasoningResult struct {
	Thought    string                 `json:"thought"`
	Action     string                 `json:"action,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Final      bool                   `json:"final"`
	Confidence float32                `json:"confidence"`
}

// ParsedResponse represents a parsed LLM response
type ParsedResponse struct {
	Type       ResponseType           `json:"type"`
	Content    string                 `json:"content"`
	Action     string                 `json:"action,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Reasoning  string                 `json:"reasoning,omitempty"`
	Final      bool                   `json:"final"`
}

// ResponseType represents the type of parsed response
type ResponseType string

const (
	ResponseTypeThought ResponseType = "thought"
	ResponseTypeAction  ResponseType = "action"
	ResponseTypeFinal   ResponseType = "final"
)

// NewAgentRequest creates a new agent request
func NewAgentRequest(sessionID, userID, input string) *AgentRequest {
	return &AgentRequest{
		SessionID:   sessionID,
		UserID:      userID,
		Input:       input,
		Context:     make(map[string]interface{}),
		MaxSteps:    10,
		Temperature: 0.7,
		Model:       "gpt-4",
	}
}

// NewStep creates a new step
func NewStep(id string, stepType StepType, input string) *Step {
	return &Step{
		ID:        id,
		Type:      stepType,
		Input:     input,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// NewToolCall creates a new tool call
func NewToolCall(id, name string, input map[string]interface{}) *ToolCall {
	return &ToolCall{
		ID:    id,
		Name:  name,
		Input: input,
	}
}
