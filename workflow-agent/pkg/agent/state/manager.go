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

package state

import (
	"context"
	"encoding/json"
	"fmt"
	"seata-go-ai-workflow-agent/pkg/agent/types"
	"seata-go-ai-workflow-agent/pkg/session"
	sessionTypes "seata-go-ai-workflow-agent/pkg/session/types"
	"time"
)

// Manager implements the StateManager interface using session storage
type Manager struct {
	sessionManager sessionTypes.SessionManager
	config         *ManagerConfig
}

// ManagerConfig holds state manager configuration
type ManagerConfig struct {
	StateKeyPrefix  string        `json:"state_key_prefix"`
	StateTTL        time.Duration `json:"state_ttl"`
	EnableCleanup   bool          `json:"enable_cleanup"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		StateKeyPrefix:  "agent_state:",
		StateTTL:        24 * time.Hour,
		EnableCleanup:   true,
		CleanupInterval: time.Hour,
	}
}

// NewManager creates a new state manager
func NewManager(sessionManager sessionTypes.SessionManager, config *ManagerConfig) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	return &Manager{
		sessionManager: sessionManager,
		config:         config,
	}
}

// NewManagerWithDefault creates a new state manager using the default session manager
func NewManagerWithDefault(config *ManagerConfig) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	return &Manager{
		sessionManager: session.DefaultManager,
		config:         config,
	}
}

// SaveState persists agent state
func (m *Manager) SaveState(ctx context.Context, state *types.AgentState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	if state.SessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// Serialize state to JSON
	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// Store in session context with TTL
	key := m.getStateKey(state.SessionID)

	if err := m.sessionManager.SetContext(ctx, state.SessionID, key, string(stateData)); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Set TTL if configured
	if m.config.StateTTL > 0 {
		if err := m.sessionManager.ExtendTTL(ctx, state.SessionID, m.config.StateTTL); err != nil {
			// Log warning but don't fail the operation
			fmt.Printf("Warning: failed to set state TTL: %v\n", err)
		}
	}

	return nil
}

// LoadState retrieves agent state
func (m *Manager) LoadState(ctx context.Context, sessionID string) (*types.AgentState, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	key := m.getStateKey(sessionID)

	// Retrieve from session context
	stateData, err := m.sessionManager.GetContext(ctx, sessionID, key)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Handle case where state doesn't exist
	if stateData == nil {
		return nil, fmt.Errorf("state not found for session %s", sessionID)
	}

	stateJSON, ok := stateData.(string)
	if !ok {
		return nil, fmt.Errorf("invalid state data format")
	}

	// Deserialize state from JSON
	var state types.AgentState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, fmt.Errorf("failed to deserialize state: %w", err)
	}

	return &state, nil
}

// DeleteState removes agent state
func (m *Manager) DeleteState(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// Delete the session entirely to remove all associated state
	if err := m.sessionManager.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	return nil
}

// UpdateStep updates a specific step in the agent state
func (m *Manager) UpdateStep(ctx context.Context, sessionID string, step *types.Step) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if step == nil {
		return fmt.Errorf("step cannot be nil")
	}

	// Load current state
	state, err := m.LoadState(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load state for step update: %w", err)
	}

	// Find and update the step
	stepUpdated := false
	for i, existingStep := range state.Steps {
		if existingStep.ID == step.ID {
			state.Steps[i] = *step
			stepUpdated = true
			break
		}
	}

	// If step not found, append it
	if !stepUpdated {
		state.Steps = append(state.Steps, *step)
	}

	// Update last activity
	state.LastActivity = time.Now()

	// Save updated state
	return m.SaveState(ctx, state)
}

// AddMemoryItem adds a memory item to the agent state
func (m *Manager) AddMemoryItem(ctx context.Context, sessionID string, item *types.MemoryItem) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if item == nil {
		return fmt.Errorf("memory item cannot be nil")
	}

	// Load current state
	state, err := m.LoadState(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load state for memory update: %w", err)
	}

	// Add memory item
	state.Memory = append(state.Memory, *item)

	// Update last activity
	state.LastActivity = time.Now()

	// Save updated state
	return m.SaveState(ctx, state)
}

// GetMemoryItems retrieves memory items of a specific type
func (m *Manager) GetMemoryItems(ctx context.Context, sessionID string, memoryType types.MemoryType) ([]types.MemoryItem, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	// Load current state
	state, err := m.LoadState(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for memory retrieval: %w", err)
	}

	// Filter memory items by type
	var filtered []types.MemoryItem
	for _, item := range state.Memory {
		if memoryType == "" || item.Type == memoryType {
			// Check TTL
			if item.TTL != nil && time.Now().After(*item.TTL) {
				continue // Skip expired items
			}
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}

// CleanupExpiredMemory removes expired memory items
func (m *Manager) CleanupExpiredMemory(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// Load current state
	state, err := m.LoadState(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load state for memory cleanup: %w", err)
	}

	// Filter out expired memory items
	now := time.Now()
	var validMemory []types.MemoryItem

	for _, item := range state.Memory {
		if item.TTL == nil || now.Before(*item.TTL) {
			validMemory = append(validMemory, item)
		}
	}

	// Update state if any items were removed
	if len(validMemory) != len(state.Memory) {
		state.Memory = validMemory
		state.LastActivity = time.Now()
		return m.SaveState(ctx, state)
	}

	return nil
}

// UpdateContext updates the context in agent state
func (m *Manager) UpdateContext(ctx context.Context, sessionID string, key string, value interface{}) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if key == "" {
		return fmt.Errorf("context key cannot be empty")
	}

	// Load current state
	state, err := m.LoadState(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load state for context update: %w", err)
	}

	// Initialize context if nil
	if state.Context == nil {
		state.Context = make(map[string]interface{})
	}

	// Update context
	state.Context[key] = value
	state.LastActivity = time.Now()

	// Save updated state
	return m.SaveState(ctx, state)
}

// GetContext retrieves a value from agent state context
func (m *Manager) GetContext(ctx context.Context, sessionID string, key string) (interface{}, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	if key == "" {
		return nil, fmt.Errorf("context key cannot be empty")
	}

	// Load current state
	state, err := m.LoadState(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for context retrieval: %w", err)
	}

	// Retrieve value
	value, exists := state.Context[key]
	if !exists {
		return nil, fmt.Errorf("context key '%s' not found", key)
	}

	return value, nil
}

// StateExists checks if state exists for a session
func (m *Manager) StateExists(ctx context.Context, sessionID string) (bool, error) {
	if sessionID == "" {
		return false, fmt.Errorf("session ID cannot be empty")
	}

	// Check if session exists
	exists, err := m.sessionManager.SessionExists(ctx, sessionID)
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	if !exists {
		return false, nil
	}

	// Check if state key exists in session context
	key := m.getStateKey(sessionID)
	_, err = m.sessionManager.GetContext(ctx, sessionID, key)
	if err != nil {
		return false, nil // State doesn't exist
	}

	return true, nil
}

// ListStates returns all active states (session IDs with agent state)
func (m *Manager) ListStates(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Get user sessions
	sessions, err := m.sessionManager.ListSessions(ctx, userID, 0, 100) // Limit to 100
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var stateSessionIDs []string
	for _, session := range sessions {
		// Check if session has agent state
		key := m.getStateKey(session.ID)
		if _, err := m.sessionManager.GetContext(ctx, session.ID, key); err == nil {
			stateSessionIDs = append(stateSessionIDs, session.ID)
		}
	}

	return stateSessionIDs, nil
}

// GetStateSummary returns a summary of the agent state
func (m *Manager) GetStateSummary(ctx context.Context, sessionID string) (*StateSummary, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	state, err := m.LoadState(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	summary := &StateSummary{
		SessionID:    state.SessionID,
		Status:       state.Status,
		CurrentStep:  state.CurrentStep,
		MaxSteps:     state.MaxSteps,
		StepCount:    len(state.Steps),
		MemoryCount:  len(state.Memory),
		ContextKeys:  len(state.Context),
		StartTime:    state.StartTime,
		LastActivity: state.LastActivity,
	}

	// Calculate execution time
	if state.Status == types.StatusCompleted || state.Status == types.StatusFailed {
		summary.ExecutionTime = state.LastActivity.Sub(state.StartTime)
	} else {
		summary.ExecutionTime = time.Since(state.StartTime)
	}

	return summary, nil
}

// StateSummary provides a summary view of agent state
type StateSummary struct {
	SessionID     string        `json:"session_id"`
	Status        types.Status  `json:"status"`
	CurrentStep   int           `json:"current_step"`
	MaxSteps      int           `json:"max_steps"`
	StepCount     int           `json:"step_count"`
	MemoryCount   int           `json:"memory_count"`
	ContextKeys   int           `json:"context_keys"`
	StartTime     time.Time     `json:"start_time"`
	LastActivity  time.Time     `json:"last_activity"`
	ExecutionTime time.Duration `json:"execution_time"`
}

// getStateKey generates the session context key for agent state
func (m *Manager) getStateKey(sessionID string) string {
	return m.config.StateKeyPrefix + sessionID
}
