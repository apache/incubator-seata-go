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

package tools

import (
	"context"
	"fmt"
	"seata-go-ai-workflow-agent/pkg/agent/types"
	"sync"
	"time"
)

// Registry implements the ToolRegistry interface
type Registry struct {
	tools   map[string]*types.Tool
	metrics map[string]*ToolMetrics
	mu      sync.RWMutex
	config  *RegistryConfig
}

// RegistryConfig holds tool registry configuration
type RegistryConfig struct {
	MaxExecutionTime time.Duration `json:"max_execution_time"`
	EnableMetrics    bool          `json:"enable_metrics"`
	EnableValidation bool          `json:"enable_validation"`
}

// ToolMetrics tracks tool usage statistics
type ToolMetrics struct {
	Name            string        `json:"name"`
	CallCount       int64         `json:"call_count"`
	SuccessCount    int64         `json:"success_count"`
	ErrorCount      int64         `json:"error_count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	LastUsed        time.Time     `json:"last_used"`
	LastError       string        `json:"last_error,omitempty"`
}

// DefaultRegistryConfig returns default registry configuration
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		MaxExecutionTime: 30 * time.Second,
		EnableMetrics:    true,
		EnableValidation: true,
	}
}

// NewRegistry creates a new tool registry
func NewRegistry(config *RegistryConfig) *Registry {
	if config == nil {
		config = DefaultRegistryConfig()
	}

	return &Registry{
		tools:   make(map[string]*types.Tool),
		metrics: make(map[string]*ToolMetrics),
		config:  config,
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool *types.Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if tool.Handler == nil {
		return fmt.Errorf("tool handler cannot be nil")
	}

	// Validate tool configuration if enabled
	if r.config.EnableValidation {
		if err := r.validateTool(tool); err != nil {
			return fmt.Errorf("tool validation failed: %w", err)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if tool already exists
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool '%s' already registered", tool.Name)
	}

	// Register the tool
	r.tools[tool.Name] = tool

	// Initialize metrics if enabled
	if r.config.EnableMetrics {
		r.metrics[tool.Name] = &ToolMetrics{
			Name:     tool.Name,
			LastUsed: time.Now(),
		}
	}

	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (*types.Tool, error) {
	if name == "" {
		return nil, fmt.Errorf("tool name cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return tool, nil
}

// List returns all registered tools
func (r *Registry) List() []*types.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*types.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// Unregister removes a tool from the registry
func (r *Registry) Unregister(name string) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}

	delete(r.tools, name)

	if r.config.EnableMetrics {
		delete(r.metrics, name)
	}

	return nil
}

// Execute runs a tool with given parameters
func (r *Registry) Execute(ctx context.Context, name string, input map[string]interface{}) (interface{}, error) {
	// Get the tool
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	// Validate input parameters if enabled
	if r.config.EnableValidation {
		if err := r.validateInput(tool, input); err != nil {
			return nil, fmt.Errorf("input validation failed: %w", err)
		}
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, r.config.MaxExecutionTime)
	defer cancel()

	// Track execution if metrics are enabled
	var startTime time.Time
	if r.config.EnableMetrics {
		startTime = time.Now()
		r.updateMetricsStart(name)
	}

	// Execute the tool
	result, err := tool.Handler(execCtx, input)

	// Update metrics if enabled
	if r.config.EnableMetrics {
		duration := time.Since(startTime)
		r.updateMetricsEnd(name, duration, err)
	}

	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// GetMetrics returns metrics for a specific tool
func (r *Registry) GetMetrics(name string) (*ToolMetrics, error) {
	if !r.config.EnableMetrics {
		return nil, fmt.Errorf("metrics not enabled")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics, exists := r.metrics[name]
	if !exists {
		return nil, fmt.Errorf("metrics for tool '%s' not found", name)
	}

	// Return a copy to avoid race conditions
	return &ToolMetrics{
		Name:            metrics.Name,
		CallCount:       metrics.CallCount,
		SuccessCount:    metrics.SuccessCount,
		ErrorCount:      metrics.ErrorCount,
		TotalDuration:   metrics.TotalDuration,
		AverageDuration: metrics.AverageDuration,
		LastUsed:        metrics.LastUsed,
		LastError:       metrics.LastError,
	}, nil
}

// GetAllMetrics returns metrics for all tools
func (r *Registry) GetAllMetrics() map[string]*ToolMetrics {
	if !r.config.EnableMetrics {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*ToolMetrics)
	for name, metrics := range r.metrics {
		result[name] = &ToolMetrics{
			Name:            metrics.Name,
			CallCount:       metrics.CallCount,
			SuccessCount:    metrics.SuccessCount,
			ErrorCount:      metrics.ErrorCount,
			TotalDuration:   metrics.TotalDuration,
			AverageDuration: metrics.AverageDuration,
			LastUsed:        metrics.LastUsed,
			LastError:       metrics.LastError,
		}
	}

	return result
}

// ResetMetrics resets metrics for a specific tool
func (r *Registry) ResetMetrics(name string) error {
	if !r.config.EnableMetrics {
		return fmt.Errorf("metrics not enabled")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.metrics[name]; !exists {
		return fmt.Errorf("metrics for tool '%s' not found", name)
	}

	r.metrics[name] = &ToolMetrics{
		Name:     name,
		LastUsed: time.Now(),
	}

	return nil
}

// validateTool validates tool configuration
func (r *Registry) validateTool(tool *types.Tool) error {
	if tool.Description == "" {
		return fmt.Errorf("tool description cannot be empty")
	}

	// Validate parameters
	for paramName, param := range tool.Parameters {
		if paramName == "" {
			return fmt.Errorf("parameter name cannot be empty")
		}

		if param.Type == "" {
			return fmt.Errorf("parameter '%s' type cannot be empty", paramName)
		}

		// Validate parameter type
		validTypes := map[string]bool{
			"string": true, "number": true, "integer": true,
			"boolean": true, "array": true, "object": true,
		}
		if !validTypes[param.Type] {
			return fmt.Errorf("parameter '%s' has invalid type '%s'", paramName, param.Type)
		}
	}

	return nil
}

// validateInput validates input parameters against tool schema
func (r *Registry) validateInput(tool *types.Tool, input map[string]interface{}) error {
	// Check required parameters
	for paramName, param := range tool.Parameters {
		if param.Required {
			if _, exists := input[paramName]; !exists {
				return fmt.Errorf("required parameter '%s' is missing", paramName)
			}
		}
	}

	// Validate parameter types and values
	for inputName, inputValue := range input {
		param, exists := tool.Parameters[inputName]
		if !exists {
			continue // Allow extra parameters
		}

		if err := r.validateParameterValue(inputName, inputValue, param); err != nil {
			return err
		}
	}

	return nil
}

// validateParameterValue validates a single parameter value
func (r *Registry) validateParameterValue(name string, value interface{}, param types.Parameter) error {
	switch param.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter '%s' must be a string", name)
		}

		// Check enum values
		if len(param.Enum) > 0 {
			strValue := value.(string)
			valid := false
			for _, enumValue := range param.Enum {
				if strValue == enumValue {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("parameter '%s' must be one of %v", name, param.Enum)
			}
		}

	case "number":
		switch value.(type) {
		case float64, float32, int, int64, int32:
			// Valid number types
		default:
			return fmt.Errorf("parameter '%s' must be a number", name)
		}

	case "integer":
		switch value.(type) {
		case int, int64, int32:
			// Valid integer types
		default:
			return fmt.Errorf("parameter '%s' must be an integer", name)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter '%s' must be a boolean", name)
		}

	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("parameter '%s' must be an array", name)
		}

	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("parameter '%s' must be an object", name)
		}
	}

	return nil
}

// updateMetricsStart updates metrics when tool execution starts
func (r *Registry) updateMetricsStart(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if metrics, exists := r.metrics[name]; exists {
		metrics.CallCount++
		metrics.LastUsed = time.Now()
	}
}

// updateMetricsEnd updates metrics when tool execution ends
func (r *Registry) updateMetricsEnd(name string, duration time.Duration, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	metrics, exists := r.metrics[name]
	if !exists {
		return
	}

	metrics.TotalDuration += duration

	if metrics.CallCount > 0 {
		metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.CallCount)
	}

	if err != nil {
		metrics.ErrorCount++
		metrics.LastError = err.Error()
	} else {
		metrics.SuccessCount++
		metrics.LastError = ""
	}
}

// ToolExists checks if a tool is registered
func (r *Registry) ToolExists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tools[name]
	return exists
}

// GetToolNames returns all registered tool names
func (r *Registry) GetToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	return names
}

// Clear removes all tools from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]*types.Tool)

	if r.config.EnableMetrics {
		r.metrics = make(map[string]*ToolMetrics)
	}
}
