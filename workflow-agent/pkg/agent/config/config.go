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

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// AgentConfig holds the complete agent configuration
type AgentConfig struct {
	Agent     AgentSettings     `json:"agent" yaml:"agent"`
	Reasoning ReasoningSettings `json:"reasoning" yaml:"reasoning"`
	Tools     ToolsSettings     `json:"tools" yaml:"tools"`
	State     StateSettings     `json:"state" yaml:"state"`
	Parser    ParserSettings    `json:"parser" yaml:"parser"`
	Logging   LoggingSettings   `json:"logging" yaml:"logging"`
	Session   SessionSettings   `json:"session" yaml:"session"`
}

// AgentSettings configures the core agent behavior
type AgentSettings struct {
	MaxSteps         int           `json:"max_steps" yaml:"max_steps"`
	DefaultModel     string        `json:"default_model" yaml:"default_model"`
	DefaultTemp      float32       `json:"default_temperature" yaml:"default_temperature"`
	ExecutionTimeout time.Duration `json:"execution_timeout" yaml:"execution_timeout"`
	MaxConcurrent    int           `json:"max_concurrent" yaml:"max_concurrent"`
	EnableMetrics    bool          `json:"enable_metrics" yaml:"enable_metrics"`
}

// ReasoningSettings configures the reasoning module
type ReasoningSettings struct {
	Model           string  `json:"model" yaml:"model"`
	Temperature     float32 `json:"temperature" yaml:"temperature"`
	MaxTokens       int     `json:"max_tokens" yaml:"max_tokens"`
	SystemPrompt    string  `json:"system_prompt" yaml:"system_prompt"`
	ReasoningPrompt string  `json:"reasoning_prompt" yaml:"reasoning_prompt"`
	ActionPrompt    string  `json:"action_prompt" yaml:"action_prompt"`
	FinalPrompt     string  `json:"final_prompt" yaml:"final_prompt"`
}

// ToolsSettings configures the tool registry
type ToolsSettings struct {
	MaxExecutionTime time.Duration `json:"max_execution_time" yaml:"max_execution_time"`
	EnableMetrics    bool          `json:"enable_metrics" yaml:"enable_metrics"`
	EnableValidation bool          `json:"enable_validation" yaml:"enable_validation"`
	EnableBuiltin    bool          `json:"enable_builtin" yaml:"enable_builtin"`
	CustomTools      []CustomTool  `json:"custom_tools" yaml:"custom_tools"`
}

// CustomTool defines a custom tool configuration
type CustomTool struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Endpoint    string            `json:"endpoint" yaml:"endpoint"`
	Method      string            `json:"method" yaml:"method"`
	Headers     map[string]string `json:"headers" yaml:"headers"`
	Timeout     time.Duration     `json:"timeout" yaml:"timeout"`
}

// StateSettings configures state management
type StateSettings struct {
	Backend         string        `json:"backend" yaml:"backend"`
	TTL             time.Duration `json:"ttl" yaml:"ttl"`
	EnableCleanup   bool          `json:"enable_cleanup" yaml:"enable_cleanup"`
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`
	KeyPrefix       string        `json:"key_prefix" yaml:"key_prefix"`
}

// ParserSettings configures response parsing
type ParserSettings struct {
	EnableJSON     bool              `json:"enable_json" yaml:"enable_json"`
	EnableMarkdown bool              `json:"enable_markdown" yaml:"enable_markdown"`
	EnableXML      bool              `json:"enable_xml" yaml:"enable_xml"`
	EnableRegex    bool              `json:"enable_regex" yaml:"enable_regex"`
	StrictMode     bool              `json:"strict_mode" yaml:"strict_mode"`
	FallbackToText bool              `json:"fallback_to_text" yaml:"fallback_to_text"`
	CustomPatterns map[string]string `json:"custom_patterns" yaml:"custom_patterns"`
}

// LoggingSettings configures logging
type LoggingSettings struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"`
	Output     string `json:"output" yaml:"output"`
	EnableFile bool   `json:"enable_file" yaml:"enable_file"`
	FilePath   string `json:"file_path" yaml:"file_path"`
	MaxSize    int    `json:"max_size" yaml:"max_size"`
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
	MaxAge     int    `json:"max_age" yaml:"max_age"`
}

// SessionSettings configures session management
type SessionSettings struct {
	UseExisting     bool          `json:"use_existing" yaml:"use_existing"`
	DefaultTTL      time.Duration `json:"default_ttl" yaml:"default_ttl"`
	MaxMessageCount int           `json:"max_message_count" yaml:"max_message_count"`
}

// DefaultConfig returns a default agent configuration
func DefaultConfig() *AgentConfig {
	return &AgentConfig{
		Agent: AgentSettings{
			MaxSteps:         10,
			DefaultModel:     "gpt-4",
			DefaultTemp:      0.7,
			ExecutionTimeout: 5 * time.Minute,
			MaxConcurrent:    10,
			EnableMetrics:    true,
		},
		Reasoning: ReasoningSettings{
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   2048,
			SystemPrompt: `You are a helpful AI agent that uses a Reason-Act-Observe approach to solve problems.

You should think step by step about the problem and decide whether to:
1. Use a tool to get more information or perform an action
2. Provide a final answer if you have enough information

When using tools, always explain your reasoning clearly.

Format your responses as:
Thought: [Your reasoning about what to do next]
Action: [tool_name] (if you want to use a tool)
Parameters: {key: value} (if action requires parameters)
Final: [your final answer] (if you're ready to conclude)

Only use "Final:" when you have a complete answer to the user's question.`,
			ReasoningPrompt: `Think carefully about the user's request and the current context. Consider what information you have and what you might need to find out.

If you need more information, choose an appropriate tool to help you.
If you have enough information, provide a final answer.`,
			ActionPrompt: `If you need to use a tool, specify:
- Action: [exact tool name]
- Parameters: [required parameters as JSON]

If you're ready to give a final answer, use:
- Final: [your complete answer]`,
			FinalPrompt: `Based on all the information gathered, provide a comprehensive answer to the user's question.`,
		},
		Tools: ToolsSettings{
			MaxExecutionTime: 30 * time.Second,
			EnableMetrics:    true,
			EnableValidation: true,
			EnableBuiltin:    true,
			CustomTools:      []CustomTool{},
		},
		State: StateSettings{
			Backend:         "session",
			TTL:             24 * time.Hour,
			EnableCleanup:   true,
			CleanupInterval: time.Hour,
			KeyPrefix:       "agent_state:",
		},
		Parser: ParserSettings{
			EnableJSON:     true,
			EnableMarkdown: true,
			EnableXML:      false,
			EnableRegex:    true,
			StrictMode:     false,
			FallbackToText: true,
			CustomPatterns: make(map[string]string),
		},
		Logging: LoggingSettings{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			EnableFile: false,
			FilePath:   "/var/log/seata-agent.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
		},
		Session: SessionSettings{
			UseExisting:     true,
			DefaultTTL:      24 * time.Hour,
			MaxMessageCount: 100,
		},
	}
}

// LoadFromFile loads configuration from a file
func LoadFromFile(filepath string) (*AgentConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()

	// Determine file format by extension
	switch {
	case isJSONFile(filepath):
		err = json.Unmarshal(data, config)
	case isYAMLFile(filepath):
		err = yaml.Unmarshal(data, config)
	default:
		return nil, fmt.Errorf("unsupported config file format")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *AgentConfig {
	config := DefaultConfig()

	// Agent settings
	if val := os.Getenv("AGENT_MAX_STEPS"); val != "" {
		if steps, err := parseInt(val); err == nil {
			config.Agent.MaxSteps = steps
		}
	}

	if val := os.Getenv("AGENT_DEFAULT_MODEL"); val != "" {
		config.Agent.DefaultModel = val
	}

	if val := os.Getenv("AGENT_DEFAULT_TEMPERATURE"); val != "" {
		if temp, err := parseFloat32(val); err == nil {
			config.Agent.DefaultTemp = temp
		}
	}

	if val := os.Getenv("AGENT_EXECUTION_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Agent.ExecutionTimeout = duration
		}
	}

	if val := os.Getenv("AGENT_MAX_CONCURRENT"); val != "" {
		if concurrent, err := parseInt(val); err == nil {
			config.Agent.MaxConcurrent = concurrent
		}
	}

	// Reasoning settings
	if val := os.Getenv("REASONING_MODEL"); val != "" {
		config.Reasoning.Model = val
	}

	if val := os.Getenv("REASONING_TEMPERATURE"); val != "" {
		if temp, err := parseFloat32(val); err == nil {
			config.Reasoning.Temperature = temp
		}
	}

	if val := os.Getenv("REASONING_MAX_TOKENS"); val != "" {
		if tokens, err := parseInt(val); err == nil {
			config.Reasoning.MaxTokens = tokens
		}
	}

	// Tools settings
	if val := os.Getenv("TOOLS_MAX_EXECUTION_TIME"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Tools.MaxExecutionTime = duration
		}
	}

	// State settings
	if val := os.Getenv("STATE_BACKEND"); val != "" {
		config.State.Backend = val
	}

	if val := os.Getenv("STATE_TTL"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.State.TTL = duration
		}
	}

	// Logging settings
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		config.Logging.Level = val
	}

	if val := os.Getenv("LOG_FORMAT"); val != "" {
		config.Logging.Format = val
	}

	if val := os.Getenv("LOG_OUTPUT"); val != "" {
		config.Logging.Output = val
	}

	return config
}

// SaveToFile saves configuration to a file
func (c *AgentConfig) SaveToFile(filepath string) error {
	var data []byte
	var err error

	switch {
	case isJSONFile(filepath):
		data, err = json.MarshalIndent(c, "", "  ")
	case isYAMLFile(filepath):
		data, err = yaml.Marshal(c)
	default:
		return fmt.Errorf("unsupported file format")
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(filepath, data, 0644)
}

// Validate validates the configuration
func (c *AgentConfig) Validate() error {
	// Validate agent settings
	if c.Agent.MaxSteps <= 0 {
		return fmt.Errorf("agent max_steps must be positive")
	}

	if c.Agent.DefaultModel == "" {
		return fmt.Errorf("agent default_model cannot be empty")
	}

	if c.Agent.DefaultTemp < 0 || c.Agent.DefaultTemp > 2 {
		return fmt.Errorf("agent default_temperature must be between 0 and 2")
	}

	if c.Agent.ExecutionTimeout <= 0 {
		return fmt.Errorf("agent execution_timeout must be positive")
	}

	if c.Agent.MaxConcurrent <= 0 {
		return fmt.Errorf("agent max_concurrent must be positive")
	}

	// Validate reasoning settings
	if c.Reasoning.Model == "" {
		return fmt.Errorf("reasoning model cannot be empty")
	}

	if c.Reasoning.Temperature < 0 || c.Reasoning.Temperature > 2 {
		return fmt.Errorf("reasoning temperature must be between 0 and 2")
	}

	if c.Reasoning.MaxTokens <= 0 {
		return fmt.Errorf("reasoning max_tokens must be positive")
	}

	// Validate tools settings
	if c.Tools.MaxExecutionTime <= 0 {
		return fmt.Errorf("tools max_execution_time must be positive")
	}

	// Validate state settings
	validBackends := map[string]bool{
		"session": true,
		"memory":  true,
		"redis":   true,
		"mysql":   true,
	}
	if !validBackends[c.State.Backend] {
		return fmt.Errorf("invalid state backend: %s", c.State.Backend)
	}

	if c.State.TTL <= 0 {
		return fmt.Errorf("state ttl must be positive")
	}

	// Validate logging settings
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	validFormats := map[string]bool{
		"json": true, "text": true,
	}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s", c.Logging.Format)
	}

	return nil
}

// Merge merges another configuration into this one
func (c *AgentConfig) Merge(other *AgentConfig) {
	if other == nil {
		return
	}

	// Merge agent settings
	if other.Agent.MaxSteps > 0 {
		c.Agent.MaxSteps = other.Agent.MaxSteps
	}
	if other.Agent.DefaultModel != "" {
		c.Agent.DefaultModel = other.Agent.DefaultModel
	}
	if other.Agent.DefaultTemp > 0 {
		c.Agent.DefaultTemp = other.Agent.DefaultTemp
	}
	if other.Agent.ExecutionTimeout > 0 {
		c.Agent.ExecutionTimeout = other.Agent.ExecutionTimeout
	}
	if other.Agent.MaxConcurrent > 0 {
		c.Agent.MaxConcurrent = other.Agent.MaxConcurrent
	}

	// Merge reasoning settings
	if other.Reasoning.Model != "" {
		c.Reasoning.Model = other.Reasoning.Model
	}
	if other.Reasoning.Temperature > 0 {
		c.Reasoning.Temperature = other.Reasoning.Temperature
	}
	if other.Reasoning.MaxTokens > 0 {
		c.Reasoning.MaxTokens = other.Reasoning.MaxTokens
	}

	// Merge custom patterns
	if other.Parser.CustomPatterns != nil {
		if c.Parser.CustomPatterns == nil {
			c.Parser.CustomPatterns = make(map[string]string)
		}
		for k, v := range other.Parser.CustomPatterns {
			c.Parser.CustomPatterns[k] = v
		}
	}
}

// Clone creates a deep copy of the configuration
func (c *AgentConfig) Clone() *AgentConfig {
	data, _ := json.Marshal(c)
	var clone AgentConfig
	json.Unmarshal(data, &clone)
	return &clone
}

// Helper functions

func isJSONFile(filepath string) bool {
	return isJSON(filepath)
}

func isYAMLFile(filepath string) bool {
	return isYAML(filepath)
}

func isJSON(filepath string) bool {
	return len(filepath) > 5 && filepath[len(filepath)-5:] == ".json"
}

func isYAML(filepath string) bool {
	return (len(filepath) > 5 && filepath[len(filepath)-5:] == ".yaml") ||
		(len(filepath) > 4 && filepath[len(filepath)-4:] == ".yml")
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func parseFloat32(s string) (float32, error) {
	var f float32
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
