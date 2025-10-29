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

package client

import (
	"fmt"
	"time"
)

// AgentSkill represents a skill that an agent can perform
type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Examples    []string `json:"examples,omitempty"`
	InputModes  []string `json:"inputModes,omitempty"`
	OutputModes []string `json:"outputModes,omitempty"`
}

// AgentCapabilities represents the capabilities of an agent
type AgentCapabilities struct {
	Streaming              bool `json:"streaming,omitempty"`
	PushNotifications      bool `json:"pushNotifications,omitempty"`
	StateTransitionHistory bool `json:"stateTransitionHistory,omitempty"`
}

// AgentCard represents the agent's metadata and capabilities
type AgentCard struct {
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	URL                string            `json:"url"`
	IconURL            string            `json:"iconUrl,omitempty"`
	Version            string            `json:"version"`
	DocumentationURL   string            `json:"documentationUrl,omitempty"`
	Capabilities       AgentCapabilities `json:"capabilities"`
	DefaultInputModes  []string          `json:"defaultInputModes"`
	DefaultOutputModes []string          `json:"defaultOutputModes"`
	Skills             []AgentSkill      `json:"skills"`
}

// RegisteredAgent represents a registered agent in the system
type RegisteredAgent struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	Version      string    `json:"version"`
	AgentCard    AgentCard `json:"agent_card"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	Status       string    `json:"status"`
	LastSeen     time.Time `json:"last_seen"`
	RegisteredAt time.Time `json:"registered_at"`
}

// RegisterRequest represents a request to register an agent
type RegisterRequest struct {
	AgentCard AgentCard `json:"agent_card"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
}

// RegisterResponse represents the response from agent registration
type RegisterResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	AgentID string `json:"agent_id,omitempty"`
}

// DiscoverRequest represents a request to discover agents
type DiscoverRequest struct {
	Query string `json:"query"`
}

// DiscoverResponse represents the response from agent discovery
type DiscoverResponse struct {
	Agents []AgentCard `json:"agents"`
}

// ContextAnalysisRequest represents a request for dynamic context analysis
type ContextAnalysisRequest struct {
	NeedDescription string `json:"need_description"`
	UserContext     string `json:"user_context,omitempty"`
}

// RouteResult represents the result of routing to an agent
type RouteResult struct {
	AgentURL      string     `json:"agent_url"`
	SkillID       string     `json:"skill_id"`
	SkillName     string     `json:"skill_name"`
	AgentInfo     *AgentCard `json:"agent_info,omitempty"`
	AgentResponse any        `json:"agent_response,omitempty"`
}

// AnalysisResult represents the result of context analysis
type AnalysisResult struct {
	RequiredSkills    []string `json:"required_skills,omitempty"`
	ContextTags       []string `json:"context_tags,omitempty"`
	SuggestedWorkflow string   `json:"suggested_workflow,omitempty"`
}

// ContextAnalysisResponse represents the response from context analysis
type ContextAnalysisResponse struct {
	Success        bool            `json:"success"`
	Message        string          `json:"message"`
	MatchedSkills  []AgentSkill    `json:"matched_skills,omitempty"`
	RouteResult    *RouteResult    `json:"route_result,omitempty"`
	AnalysisResult *AnalysisResult `json:"analysis_result,omitempty"`
}

// APIResponse represents a generic API response wrapper
type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    *T     `json:"data,omitempty"`
	Error   *struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	} `json:"error,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Success bool `json:"success"`
	Error   struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	} `json:"error"`
}

// APIError represents an API error
type APIError struct {
	StatusCode int
	Message    string
	Code       int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s (code: %d)", e.StatusCode, e.Message, e.Code)
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Component string `json:"component"`
	Status    string `json:"status"`
}

// AuthTokenResponse represents the authentication token response
type AuthTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

// Agent status constants
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusFailed   = "failed"
)

// Helper methods

// IsActive returns true if the agent is active
func (a *RegisteredAgent) IsActive() bool {
	return a.Status == StatusActive
}

// HasSkill returns true if the agent has the specified skill
func (a *RegisteredAgent) HasSkill(skillID string) bool {
	for _, skill := range a.AgentCard.Skills {
		if skill.ID == skillID {
			return true
		}
	}
	return false
}

// GetSkillByID returns the skill with the specified ID, or nil if not found
func (a *RegisteredAgent) GetSkillByID(skillID string) *AgentSkill {
	for _, skill := range a.AgentCard.Skills {
		if skill.ID == skillID {
			return &skill
		}
	}
	return nil
}

// HasTag returns true if the skill has the specified tag
func (s *AgentSkill) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}
