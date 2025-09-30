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

package agent

import (
	"fmt"
	"seata-go-ai-workflow-agent/pkg/agent/config"
	"seata-go-ai-workflow-agent/pkg/agent/core"
	"seata-go-ai-workflow-agent/pkg/agent/errors"
	"seata-go-ai-workflow-agent/pkg/agent/logging"
	"seata-go-ai-workflow-agent/pkg/agent/parser"
	"seata-go-ai-workflow-agent/pkg/agent/reasoning"
	"seata-go-ai-workflow-agent/pkg/agent/state"
	"seata-go-ai-workflow-agent/pkg/agent/tools"
	"seata-go-ai-workflow-agent/pkg/agent/types"
	"seata-go-ai-workflow-agent/pkg/session"

	"github.com/firebase/genkit/go/ai"
)

// Builder helps construct ReAct agents with proper configuration
type Builder struct {
	config      *config.AgentConfig
	model       ai.Model
	logger      *logging.Logger
	customTools []*types.Tool
}

// NewBuilder creates a new agent builder
func NewBuilder() *Builder {
	return &Builder{
		config:      config.DefaultConfig(),
		customTools: make([]*types.Tool, 0),
	}
}

// WithConfig sets the agent configuration
func (b *Builder) WithConfig(cfg *config.AgentConfig) *Builder {
	b.config = cfg
	return b
}

// WithModel sets the AI model for reasoning
func (b *Builder) WithModel(model ai.Model) *Builder {
	b.model = model
	return b
}

// WithLogger sets the logger
func (b *Builder) WithLogger(logger *logging.Logger) *Builder {
	b.logger = logger
	return b
}

// WithTool adds a custom tool
func (b *Builder) WithTool(tool *types.Tool) *Builder {
	b.customTools = append(b.customTools, tool)
	return b
}

// Build creates and configures the ReAct agent
func (b *Builder) Build() (types.Agent, error) {
	// Validate configuration
	if err := b.config.Validate(); err != nil {
		return nil, errors.NewInvalidConfigError(err.Error())
	}

	// Initialize logging if not provided
	if b.logger == nil {
		logConfig := &logging.Config{
			Level:     b.config.Logging.Level,
			Format:    b.config.Logging.Format,
			Output:    b.config.Logging.Output,
			Component: "agent",
		}
		logger, err := logging.NewLogger(logConfig)
		if err != nil {
			return nil, errors.Wrap(err, errors.ComponentConfig,
				errors.ErrCodeConfigLoadFailed, "failed to initialize logging")
		}
		b.logger = logger
	}

	// Initialize session management if configured
	if b.config.Session.UseExisting && session.DefaultManager == nil {
		return nil, errors.NewInvalidConfigError("session manager not initialized but required")
	}

	// Create tool registry
	toolConfig := &tools.RegistryConfig{
		MaxExecutionTime: b.config.Tools.MaxExecutionTime,
		EnableMetrics:    b.config.Tools.EnableMetrics,
		EnableValidation: b.config.Tools.EnableValidation,
	}
	toolRegistry := tools.NewRegistry(toolConfig)

	// Register built-in tools if enabled
	if b.config.Tools.EnableBuiltin {
		if err := tools.RegisterBuiltinTools(toolRegistry); err != nil {
			return nil, errors.Wrap(err, errors.ComponentTools,
				errors.ErrCodeToolRegistrationFailed, "failed to register built-in tools")
		}
	}

	// Register custom tools
	for _, tool := range b.customTools {
		if err := toolRegistry.Register(tool); err != nil {
			return nil, errors.Wrap(err, errors.ComponentTools,
				errors.ErrCodeToolRegistrationFailed,
				fmt.Sprintf("failed to register custom tool: %s", tool.Name))
		}
	}

	// Create state manager
	stateConfig := &state.ManagerConfig{
		StateKeyPrefix:  b.config.State.KeyPrefix,
		StateTTL:        b.config.State.TTL,
		EnableCleanup:   b.config.State.EnableCleanup,
		CleanupInterval: b.config.State.CleanupInterval,
	}

	var stateManager types.StateManager
	switch b.config.State.Backend {
	case "session":
		stateManager = state.NewManagerWithDefault(stateConfig)
	default:
		return nil, errors.NewInvalidConfigError(
			fmt.Sprintf("unsupported state backend: %s", b.config.State.Backend))
	}

	// Create response parser (embedded in reasoner for now)
	_ = &parser.ParserConfig{
		EnableJSON:     b.config.Parser.EnableJSON,
		EnableMarkdown: b.config.Parser.EnableMarkdown,
		EnableXML:      b.config.Parser.EnableXML,
		EnableRegex:    b.config.Parser.EnableRegex,
		StrictMode:     b.config.Parser.StrictMode,
		FallbackToText: b.config.Parser.FallbackToText,
		CustomPatterns: b.config.Parser.CustomPatterns,
	}

	// Create reasoner
	if b.model == nil {
		return nil, errors.NewInvalidConfigError("AI model is required")
	}

	reasonerConfig := &reasoning.ReasonerConfig{
		Model:           b.config.Reasoning.Model,
		Temperature:     b.config.Reasoning.Temperature,
		MaxTokens:       b.config.Reasoning.MaxTokens,
		SystemPrompt:    b.config.Reasoning.SystemPrompt,
		ReasoningPrompt: b.config.Reasoning.ReasoningPrompt,
		ActionPrompt:    b.config.Reasoning.ActionPrompt,
		FinalPrompt:     b.config.Reasoning.FinalPrompt,
	}
	reasoner := reasoning.NewLLMReasoner(b.model, reasonerConfig)

	// Create core agent configuration
	coreConfig := &core.Config{
		MaxSteps:         b.config.Agent.MaxSteps,
		DefaultModel:     b.config.Agent.DefaultModel,
		DefaultTemp:      b.config.Agent.DefaultTemp,
		ExecutionTimeout: b.config.Agent.ExecutionTimeout,
		MaxConcurrent:    b.config.Agent.MaxConcurrent,
	}

	// Create the agent
	agent, err := core.NewReActAgent(reasoner, toolRegistry, stateManager, coreConfig)
	if err != nil {
		return nil, errors.Wrap(err, errors.ComponentAgent,
			errors.ErrCodeSystemError, "failed to create ReAct agent")
	}

	b.logger.Info("ReAct agent initialized successfully")
	return agent, nil
}

// Convenience functions for common use cases

// NewSimpleAgent creates a basic agent with default configuration
func NewSimpleAgent(model ai.Model) (types.Agent, error) {
	return NewBuilder().WithModel(model).Build()
}

// NewAgentFromConfig creates an agent from configuration file
func NewAgentFromConfig(configPath string, model ai.Model) (types.Agent, error) {
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return nil, err
	}

	return NewBuilder().
		WithConfig(cfg).
		WithModel(model).
		Build()
}

// NewAgentFromEnv creates an agent using environment variables
func NewAgentFromEnv(model ai.Model) (types.Agent, error) {
	cfg := config.LoadFromEnv()

	return NewBuilder().
		WithConfig(cfg).
		WithModel(model).
		Build()
}
