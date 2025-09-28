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
