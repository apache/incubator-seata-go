package models

import (
	"time"

	"agenthub/pkg/common"
)

// AgentProvider represents an agent provider
type AgentProvider struct {
	Organization string `json:"organization"`
	URL          string `json:"url"`
}

// AgentCapabilities represents agent capabilities
type AgentCapabilities struct {
	Streaming              bool `json:"streaming,omitempty"`
	PushNotifications      bool `json:"pushNotifications,omitempty"`
	StateTransitionHistory bool `json:"stateTransitionHistory,omitempty"`
}

// AgentSkill represents an agent skill
type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Examples    []string `json:"examples,omitempty"`
	InputModes  []string `json:"inputModes,omitempty"`
	OutputModes []string `json:"outputModes,omitempty"`
}

// AgentCard represents the agent card specification
type AgentCard struct {
	Name                              string                           `json:"name"`
	Description                       string                           `json:"description"`
	URL                               string                           `json:"url"`
	IconURL                           string                           `json:"iconUrl,omitempty"`
	Provider                          *AgentProvider                   `json:"provider,omitempty"`
	Version                           string                           `json:"version"`
	DocumentationURL                  string                           `json:"documentationUrl,omitempty"`
	Capabilities                      AgentCapabilities                `json:"capabilities"`
	SecuritySchemes                   map[string]common.SecurityScheme `json:"securitySchemes,omitempty"`
	Security                          []map[string][]string            `json:"security,omitempty"`
	DefaultInputModes                 []string                         `json:"defaultInputModes"`
	DefaultOutputModes                []string                         `json:"defaultOutputModes"`
	Skills                            []AgentSkill                     `json:"skills"`
	SupportsAuthenticatedExtendedCard bool                             `json:"supportsAuthenticatedExtendedCard,omitempty"`
}

// RegisteredAgent represents a registered agent with metadata
type RegisteredAgent struct {
	*common.BaseResource `json:",inline"`
	AgentCard            AgentCard `json:"agent_card"`
	Host                 string    `json:"host"`
	Port                 int       `json:"port"`
	Status               string    `json:"status"`
	LastSeen             time.Time `json:"last_seen"`
	RegisteredAt         time.Time `json:"registered_at"`
}

// NewRegisteredAgent creates a new registered agent
func NewRegisteredAgent(card AgentCard, host string, port int) *RegisteredAgent {
	now := time.Now()
	return &RegisteredAgent{
		BaseResource: &common.BaseResource{
			ID:        card.Name,
			Kind:      "RegisteredAgent",
			Version:   "v1",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  make(map[string]interface{}),
		},
		AgentCard:    card,
		Host:         host,
		Port:         port,
		Status:       "active",
		LastSeen:     now,
		RegisteredAt: now,
	}
}

// UpdateLastSeen updates the last seen timestamp
func (r *RegisteredAgent) UpdateLastSeen() {
	r.LastSeen = time.Now()
	r.BaseResource.UpdatedAt = time.Now()
}

// SetStatus sets the agent status
func (r *RegisteredAgent) SetStatus(status string) {
	r.Status = status
	r.BaseResource.UpdatedAt = time.Now()
}

// IsActive returns true if the agent is active
func (r *RegisteredAgent) IsActive() bool {
	return r.Status == "active"
}

// GetFullAddress returns the full address of the agent
func (r *RegisteredAgent) GetFullAddress() string {
	return r.Host + ":" + string(rune(r.Port))
}

// Request and Response types
type DiscoverRequest struct {
	Query string `json:"query"`
}

type DiscoverResponse struct {
	Agents []AgentCard `json:"agents"`
}

type RegisterRequest struct {
	AgentCard AgentCard `json:"agent_card"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
}

type RegisterResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	AgentID string `json:"agent_id"`
}
