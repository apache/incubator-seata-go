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
	"sync"
)

// DefaultToolRegistry implements the ToolRegistry interface
type DefaultToolRegistry struct {
	mu        sync.RWMutex
	tools     map[string]Tool
	executors map[string]ToolExecutor
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() ToolRegistry {
	return &DefaultToolRegistry{
		tools:     make(map[string]Tool),
		executors: make(map[string]ToolExecutor),
	}
}

// Register registers a tool with its executor
func (r *DefaultToolRegistry) Register(tool Tool, executor ToolExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools[tool.Function.Name] = tool
	r.executors[tool.Function.Name] = executor
}

// GetTool retrieves a tool by name
func (r *DefaultToolRegistry) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// GetExecutor retrieves an executor by tool name
func (r *DefaultToolRegistry) GetExecutor(name string) (ToolExecutor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executor, exists := r.executors[name]
	return executor, exists
}

// ListTools returns all registered tools
func (r *DefaultToolRegistry) ListTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []Tool
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}
