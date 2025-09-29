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

package core

import (
	"context"
	"fmt"
	"seata-go-ai-workflow-agent/pkg/agent/types"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ReActAgent implements the Agent interface
type ReActAgent struct {
	reasoner     types.Reasoner
	toolRegistry types.ToolRegistry
	stateManager types.StateManager
	config       *Config
	mu           sync.RWMutex
	executions   map[string]*execution
	shutdownCh   chan struct{}
	wg           sync.WaitGroup
}

// Config holds agent configuration
type Config struct {
	MaxSteps         int           `json:"max_steps"`
	DefaultModel     string        `json:"default_model"`
	DefaultTemp      float32       `json:"default_temperature"`
	ExecutionTimeout time.Duration `json:"execution_timeout"`
	MaxConcurrent    int           `json:"max_concurrent"`
}

// execution tracks a single agent execution
type execution struct {
	ctx       context.Context
	cancel    context.CancelFunc
	request   *types.AgentRequest
	state     *types.AgentState
	startTime time.Time
	done      chan struct{}
}

// DefaultConfig returns default agent configuration
func DefaultConfig() *Config {
	return &Config{
		MaxSteps:         10,
		DefaultModel:     "gpt-4",
		DefaultTemp:      0.7,
		ExecutionTimeout: 5 * time.Minute,
		MaxConcurrent:    10,
	}
}

// NewReActAgent creates a new ReAct agent
func NewReActAgent(
	reasoner types.Reasoner,
	toolRegistry types.ToolRegistry,
	stateManager types.StateManager,
	config *Config,
) (*ReActAgent, error) {
	if config == nil {
		config = DefaultConfig()
	}

	agent := &ReActAgent{
		reasoner:     reasoner,
		toolRegistry: toolRegistry,
		stateManager: stateManager,
		config:       config,
		executions:   make(map[string]*execution),
		shutdownCh:   make(chan struct{}),
	}

	return agent, nil
}

// Process executes the agent workflow
func (a *ReActAgent) Process(ctx context.Context, req *types.AgentRequest) (*types.AgentResponse, error) {
	// Validate request
	if err := a.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check concurrent execution limit
	if err := a.checkConcurrencyLimit(); err != nil {
		return nil, err
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, a.config.ExecutionTimeout)
	defer cancel()

	// Track execution
	exec := &execution{
		ctx:       execCtx,
		cancel:    cancel,
		request:   req,
		startTime: time.Now(),
		done:      make(chan struct{}),
	}

	a.mu.Lock()
	a.executions[req.SessionID] = exec
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.executions, req.SessionID)
		a.mu.Unlock()
		close(exec.done)
	}()

	// Execute the workflow directly
	response, err := a.executeFlow(execCtx, req)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	return response, nil
}

// executeFlow implements the core ReAct workflow
func (a *ReActAgent) executeFlow(ctx context.Context, req *types.AgentRequest) (*types.AgentResponse, error) {
	startTime := time.Now()

	// Initialize agent state
	state, err := a.initializeState(req)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state: %w", err)
	}

	// Save initial state
	if err := a.stateManager.SaveState(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to save initial state: %w", err)
	}

	// Execute ReAct loop
	response, err := a.reactLoop(ctx, req, state)
	if err != nil {
		state.Status = types.StatusFailed
		a.stateManager.SaveState(ctx, state)
		return nil, err
	}

	// Update final state
	state.Status = types.StatusCompleted
	state.LastActivity = time.Now()
	if err := a.stateManager.SaveState(ctx, state); err != nil {
		// Log error but don't fail the response
		fmt.Printf("Warning: failed to save final state: %v\n", err)
	}

	// Build response
	response.SessionID = req.SessionID
	response.Steps = state.Steps
	response.Status = types.StatusCompleted
	response.ExecutionTime = types.Duration(time.Since(startTime))
	response.Metadata = types.Metadata{
		"steps_taken":    len(state.Steps),
		"model_used":     req.Model,
		"temperature":    req.Temperature,
		"execution_time": time.Since(startTime).String(),
	}

	return response, nil
}

// reactLoop implements the main Reason-Act-Observe loop
func (a *ReActAgent) reactLoop(ctx context.Context, req *types.AgentRequest, state *types.AgentState) (*types.AgentResponse, error) {
	maxSteps := req.MaxSteps
	if maxSteps == 0 {
		maxSteps = a.config.MaxSteps
	}

	var finalResponse string
	var toolCalls []types.ToolCall

	for step := 0; step < maxSteps; step++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		state.CurrentStep = step
		state.LastActivity = time.Now()

		// Reasoning step
		reasoningResult, err := a.performReasoning(ctx, req, state)
		if err != nil {
			return nil, fmt.Errorf("reasoning failed at step %d: %w", step, err)
		}

		// Create reasoning step
		reasoningStep := types.NewStep(
			generateStepID(),
			types.StepTypeReasoning,
			fmt.Sprintf("Step %d reasoning", step+1),
		)
		reasoningStep.Output = reasoningResult.Thought
		reasoningStep.Reasoning = reasoningResult.Thought
		reasoningStep.Duration = time.Since(reasoningStep.Timestamp)

		state.Steps = append(state.Steps, *reasoningStep)

		// Update state
		if err := a.stateManager.UpdateStep(ctx, state.SessionID, reasoningStep); err != nil {
			fmt.Printf("Warning: failed to update reasoning step: %v\n", err)
		}

		// Check if this is the final answer
		if reasoningResult.Final {
			finalResponse = reasoningResult.Thought
			break
		}

		// Action step (if action is specified)
		if reasoningResult.Action != "" {
			actionResult, toolCall, err := a.performAction(ctx, reasoningResult, state)
			if err != nil {
				return nil, fmt.Errorf("action failed at step %d: %w", step, err)
			}

			// Create action step
			actionStep := types.NewStep(
				generateStepID(),
				types.StepTypeAction,
				fmt.Sprintf("Execute %s", reasoningResult.Action),
			)
			actionStep.Output = fmt.Sprintf("%v", actionResult)
			actionStep.ToolCall = toolCall
			actionStep.Duration = time.Since(actionStep.Timestamp)

			state.Steps = append(state.Steps, *actionStep)
			toolCalls = append(toolCalls, *toolCall)

			// Update context with action result
			state.Context[fmt.Sprintf("action_%d_result", step)] = actionResult

			// Update state
			if err := a.stateManager.UpdateStep(ctx, state.SessionID, actionStep); err != nil {
				fmt.Printf("Warning: failed to update action step: %v\n", err)
			}
		}

		// Save intermediate state
		if err := a.stateManager.SaveState(ctx, state); err != nil {
			fmt.Printf("Warning: failed to save intermediate state: %v\n", err)
		}
	}

	// If no final answer was reached, use the last reasoning result
	if finalResponse == "" {
		if len(state.Steps) > 0 {
			lastStep := state.Steps[len(state.Steps)-1]
			finalResponse = lastStep.Output
		} else {
			finalResponse = "No response generated"
		}
	}

	return &types.AgentResponse{
		Response:  finalResponse,
		ToolCalls: toolCalls,
	}, nil
}

// performReasoning executes the reasoning step
func (a *ReActAgent) performReasoning(ctx context.Context, req *types.AgentRequest, state *types.AgentState) (*types.ReasoningResult, error) {
	// Build context with conversation history and current state
	contextData := make(map[string]interface{})

	// Add request context
	for k, v := range req.Context {
		contextData[k] = v
	}

	// Add state context
	for k, v := range state.Context {
		contextData[k] = v
	}

	// Add step history
	contextData["previous_steps"] = state.Steps
	contextData["current_step"] = state.CurrentStep
	contextData["max_steps"] = state.MaxSteps

	// Add available tools
	tools := a.toolRegistry.List()
	toolDescriptions := make([]string, len(tools))
	for i, tool := range tools {
		toolDescriptions[i] = fmt.Sprintf("%s: %s", tool.Name, tool.Description)
	}
	contextData["available_tools"] = toolDescriptions

	// Perform reasoning
	result, err := a.reasoner.Reason(ctx, req.Input, contextData)
	if err != nil {
		return nil, fmt.Errorf("reasoning failed: %w", err)
	}

	return result, nil
}

// performAction executes a tool action
func (a *ReActAgent) performAction(ctx context.Context, reasoning *types.ReasoningResult, state *types.AgentState) (interface{}, *types.ToolCall, error) {
	toolCall := types.NewToolCall(
		generateToolCallID(),
		reasoning.Action,
		reasoning.Parameters,
	)

	startTime := time.Now()

	result, err := a.toolRegistry.Execute(ctx, reasoning.Action, reasoning.Parameters)

	toolCall.Duration = time.Since(startTime)

	if err != nil {
		toolCall.Error = &types.ToolError{
			Code:      "execution_failed",
			Message:   err.Error(),
			Retryable: false,
		}
		return nil, toolCall, fmt.Errorf("tool execution failed: %w", err)
	}

	toolCall.Output = result
	return result, toolCall, nil
}

// initializeState creates initial agent state
func (a *ReActAgent) initializeState(req *types.AgentRequest) (*types.AgentState, error) {
	maxSteps := req.MaxSteps
	if maxSteps == 0 {
		maxSteps = a.config.MaxSteps
	}

	state := &types.AgentState{
		SessionID:    req.SessionID,
		CurrentStep:  0,
		MaxSteps:     maxSteps,
		Status:       types.StatusRunning,
		Steps:        make([]types.Step, 0),
		Context:      make(map[string]interface{}),
		Memory:       make([]types.MemoryItem, 0),
		StartTime:    time.Now(),
		LastActivity: time.Now(),
	}

	// Copy request context
	for k, v := range req.Context {
		state.Context[k] = v
	}

	return state, nil
}

// GetState returns current agent state
func (a *ReActAgent) GetState(ctx context.Context, sessionID string) (*types.AgentState, error) {
	return a.stateManager.LoadState(ctx, sessionID)
}

// Cancel cancels an ongoing execution
func (a *ReActAgent) Cancel(ctx context.Context, sessionID string) error {
	a.mu.RLock()
	exec, exists := a.executions[sessionID]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no active execution found for session %s", sessionID)
	}

	exec.cancel()

	// Update state to cancelled
	state, err := a.stateManager.LoadState(ctx, sessionID)
	if err == nil {
		state.Status = types.StatusCancelled
		state.LastActivity = time.Now()
		a.stateManager.SaveState(ctx, state)
	}

	return nil
}

// RegisterTool registers a new tool
func (a *ReActAgent) RegisterTool(tool *types.Tool) error {
	return a.toolRegistry.Register(tool)
}

// ListTools returns available tools
func (a *ReActAgent) ListTools() []*types.Tool {
	return a.toolRegistry.List()
}

// Close shuts down the agent
func (a *ReActAgent) Close() error {
	close(a.shutdownCh)

	// Cancel all ongoing executions
	a.mu.RLock()
	executions := make([]*execution, 0, len(a.executions))
	for _, exec := range a.executions {
		executions = append(executions, exec)
	}
	a.mu.RUnlock()

	for _, exec := range executions {
		exec.cancel()
	}

	// Wait for all executions to complete
	for _, exec := range executions {
		<-exec.done
	}

	a.wg.Wait()
	return nil
}

// validateRequest validates the agent request
func (a *ReActAgent) validateRequest(req *types.AgentRequest) error {
	if req.SessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	if req.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if req.Input == "" {
		return fmt.Errorf("input is required")
	}
	return nil
}

// checkConcurrencyLimit checks if we can start a new execution
func (a *ReActAgent) checkConcurrencyLimit() error {
	a.mu.RLock()
	count := len(a.executions)
	a.mu.RUnlock()

	if count >= a.config.MaxConcurrent {
		return fmt.Errorf("maximum concurrent executions reached: %d", a.config.MaxConcurrent)
	}
	return nil
}

// generateStepID generates a unique step ID
func generateStepID() string {
	return "step_" + uuid.New().String()
}

// generateToolCallID generates a unique tool call ID
func generateToolCallID() string {
	return "call_" + uuid.New().String()
}
