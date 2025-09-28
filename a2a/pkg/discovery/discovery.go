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

package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"seata-go-ai-a2a/pkg/types"
)

// Service provides Agent Card discovery functionality
type Service struct {
	agentCard *types.AgentCard
}

// NewService creates a new discovery service with the given agent card
func NewService(agentCard *types.AgentCard) *Service {
	return &Service{
		agentCard: agentCard,
	}
}

// RegisterWellKnownEndpoints registers the well-known Agent Card discovery endpoints
// This implements the A2A specification for Agent Card discovery
func (s *Service) RegisterWellKnownEndpoints(mux *http.ServeMux) {
	// Register the well-known agent card endpoint
	// As per A2A specification: https://{domain}/.well-known/agent-card.json
	mux.HandleFunc("/.well-known/agent-card.json", s.handleAgentCardDiscovery)

	// Additional discovery endpoint as fallback
	mux.HandleFunc("/.well-known/agent-card", s.handleAgentCardDiscovery)
}

// handleAgentCardDiscovery handles requests for the agent card discovery endpoint
func (s *Service) handleAgentCardDiscovery(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Header().Set("Access-Control-Allow-Origin", "*")      // Allow CORS for discovery

	// Serialize and return the agent card
	if err := json.NewEncoder(w).Encode(s.agentCard); err != nil {
		http.Error(w, "Failed to encode agent card", http.StatusInternalServerError)
		return
	}
}

// UpdateAgentCard updates the agent card served by the discovery service
func (s *Service) UpdateAgentCard(agentCard *types.AgentCard) {
	s.agentCard = agentCard
}

// GetAgentCard returns the current agent card
func (s *Service) GetAgentCard() *types.AgentCard {
	return s.agentCard
}

// DiscoverAgentCard discovers an agent card from a given domain
// This is a client-side function to discover remote agent cards
func DiscoverAgentCard(ctx context.Context, domain string) (*types.AgentCard, error) {
	// Try the primary well-known URI first
	url := fmt.Sprintf("https://%s/.well-known/agent-card.json", domain)

	agentCard, err := fetchAgentCard(ctx, url)
	if err == nil {
		return agentCard, nil
	}

	// Try fallback without .json extension
	url = fmt.Sprintf("https://%s/.well-known/agent-card", domain)
	agentCard, err = fetchAgentCard(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to discover agent card for domain %s: %w", domain, err)
	}

	return agentCard, nil
}

// fetchAgentCard fetches an agent card from the given URL
func fetchAgentCard(ctx context.Context, url string) (*types.AgentCard, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "A2A-Discovery/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agent card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var agentCard types.AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&agentCard); err != nil {
		return nil, fmt.Errorf("failed to decode agent card: %w", err)
	}

	return &agentCard, nil
}

// ValidateAgentCard validates that an agent card conforms to A2A specification
func ValidateAgentCard(agentCard *types.AgentCard) error {
	if agentCard == nil {
		return fmt.Errorf("agent card cannot be nil")
	}

	if agentCard.ProtocolVersion == "" {
		return fmt.Errorf("protocol version is required")
	}

	if agentCard.Name == "" {
		return fmt.Errorf("name is required")
	}

	if agentCard.Description == "" {
		return fmt.Errorf("description is required")
	}

	if agentCard.URL == "" {
		return fmt.Errorf("URL is required")
	}

	if agentCard.PreferredTransport == "" {
		return fmt.Errorf("preferred transport is required")
	}

	if agentCard.Version == "" {
		return fmt.Errorf("version is required")
	}

	// Validate preferred transport is one of the supported values
	validTransports := map[string]bool{
		"grpc":      true,
		"jsonrpc":   true,
		"http+json": true,
		"rest":      true,
	}

	if !validTransports[agentCard.PreferredTransport] {
		return fmt.Errorf("invalid preferred transport: %s", agentCard.PreferredTransport)
	}

	return nil
}
