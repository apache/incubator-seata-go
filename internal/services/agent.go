package services

import (
	"context"
	"fmt"
	"time"

	"agenthub/pkg/models"
	"agenthub/pkg/storage"
	"agenthub/pkg/utils"
)

// AgentService provides agent management functionality following K8s patterns
type AgentService struct {
	storage         storage.Storage
	logger          *utils.Logger
	matcher         *utils.StringMatcher
	contextAnalyzer *ContextAnalyzer // Dynamic context analyzer
}

// AgentServiceConfig holds agent service configuration
type AgentServiceConfig struct {
	Storage         storage.Storage
	Logger          *utils.Logger
	ContextAnalyzer *ContextAnalyzer // Optional dynamic context analyzer
}

// NewAgentService creates a new agent service
func NewAgentService(config AgentServiceConfig) *AgentService {
	logger := config.Logger
	if logger == nil {
		logger = utils.WithField("component", "agent-service")
	}
	
	return &AgentService{
		storage:         config.Storage,
		logger:          logger,
		matcher:         utils.NewStringMatcher(),
		contextAnalyzer: config.ContextAnalyzer,
	}
}

// RegisterAgent registers a new agent
func (s *AgentService) RegisterAgent(ctx context.Context, req *models.RegisterRequest) (*models.RegisterResponse, error) {
	s.logger.Info("Registering agent: %s", req.AgentCard.Name)
	
	if err := s.validateRegisterRequest(req); err != nil {
		s.logger.Warn("Invalid register request: %v", err)
		return &models.RegisterResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid request: %s", err.Error()),
			AgentID: "",
		}, nil
	}

	registeredAgent := models.NewRegisteredAgent(req.AgentCard, req.Host, req.Port)
	
	if err := s.storage.Create(ctx, registeredAgent); err != nil {
		s.logger.Error("Error registering agent: %v", err)
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	s.logger.Info("Agent registered successfully: %s", req.AgentCard.Name)

	return &models.RegisterResponse{
		Success: true,
		Message: "Agent registered successfully",
		AgentID: req.AgentCard.Name,
	}, nil
}

// DiscoverAgents discovers agents based on query (now supports skill-based discovery)
func (s *AgentService) DiscoverAgents(ctx context.Context, req *models.DiscoverRequest) (*models.DiscoverResponse, error) {
	s.logger.Info("Discovering agents with skill: %s", req.Query)

	// Check if using skill-based NamingServer storage
	if skillStorage, ok := s.storage.(*storage.SkillBasedNamingServerStorage); ok {
		return s.discoverAgentsBySkillFromNaming(ctx, req, skillStorage)
	}
	
	// Fallback to legacy discovery
	return s.legacyDiscoverAgents(ctx, req)
}

// discoverAgentsBySkillFromNaming discovers agents using NamingServer skill-based lookup
func (s *AgentService) discoverAgentsBySkillFromNaming(ctx context.Context, req *models.DiscoverRequest, storage *storage.SkillBasedNamingServerStorage) (*models.DiscoverResponse, error) {
	skillName := req.Query // Query parameter is now used as skill name
	
	hostUrl, err := storage.DiscoverUrlBySkill(ctx, skillName)
	if err != nil {
		s.logger.Warn("Skill %s not found in NamingServer", skillName)
		return &models.DiscoverResponse{
			Agents: []models.AgentCard{}, // Return empty list
		}, nil
	}
	
	// Get AgentCard information from storage cache
	agentCard := s.getAgentCardByUrl(storage, hostUrl, skillName)
	
	s.logger.Info("Found agent for skill %s: %s", skillName, hostUrl)
	return &models.DiscoverResponse{
		Agents: []models.AgentCard{agentCard},
	}, nil
}

// getAgentCardByUrl retrieves AgentCard information based on URL and skill
func (s *AgentService) getAgentCardByUrl(storage *storage.SkillBasedNamingServerStorage, hostUrl string, skillName string) models.AgentCard {
	// Try to get complete AgentCard from cache
	if agentId, exists := storage.GetSkillToAgent()[skillName]; exists {
		if agent, exists := storage.GetAgentCache()[agentId]; exists {
			return agent.AgentCard
		}
	}
	
	// If not found in cache, construct a simplified AgentCard
	return models.AgentCard{
		Name: fmt.Sprintf("agent-with-%s", skillName),
		URL:  hostUrl,
		Skills: []models.AgentSkill{
			{
				ID:   skillName,
				Name: skillName,
			},
		},
	}
}

// legacyDiscoverAgents performs traditional agent discovery (fallback)
func (s *AgentService) legacyDiscoverAgents(ctx context.Context, req *models.DiscoverRequest) (*models.DiscoverResponse, error) {
	s.logger.Info("Using legacy discovery for query: %s", req.Query)
	
	// Get all registered agents
	resources, err := s.storage.List(ctx, map[string]interface{}{
		"kind": "RegisteredAgent",
	})
	if err != nil {
		s.logger.Error("Error listing agents: %v", err)
		return nil, fmt.Errorf("failed to discover agents: %w", err)
	}

	var matchingCards []models.AgentCard
	for _, resource := range resources {
		if agent, ok := resource.(*models.RegisteredAgent); ok && agent.IsActive() {
			if s.matchesQuery(agent, req.Query) {
				matchingCards = append(matchingCards, agent.AgentCard)
			}
		}
	}

	s.logger.Info("Found %d agents matching query: %s", len(matchingCards), req.Query)

	return &models.DiscoverResponse{
		Agents: matchingCards,
	}, nil
}

// GetAgent retrieves an agent by ID
func (s *AgentService) GetAgent(ctx context.Context, id string) (*models.RegisteredAgent, error) {
	s.logger.Debug("Getting agent: %s", id)
	
	resource, err := s.storage.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	
	agent, ok := resource.(*models.RegisteredAgent)
	if !ok {
		return nil, fmt.Errorf("invalid agent type")
	}
	
	return agent, nil
}

// UpdateAgentStatus updates an agent's status
func (s *AgentService) UpdateAgentStatus(ctx context.Context, id string, status string) error {
	s.logger.Debug("Updating agent status: %s -> %s", id, status)
	
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return err
	}
	
	agent.SetStatus(status)
	
	return s.storage.Update(ctx, agent)
}

// RemoveAgent removes an agent
func (s *AgentService) RemoveAgent(ctx context.Context, id string) error {
	s.logger.Info("Removing agent: %s", id)
	
	return s.storage.Delete(ctx, id)
}

// ListAllAgents lists all registered agents
func (s *AgentService) ListAllAgents(ctx context.Context) ([]*models.RegisteredAgent, error) {
	s.logger.Debug("Listing all agents")
	
	resources, err := s.storage.List(ctx, map[string]interface{}{
		"kind": "RegisteredAgent",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	
	var agents []*models.RegisteredAgent
	for _, resource := range resources {
		if agent, ok := resource.(*models.RegisteredAgent); ok {
			agents = append(agents, agent)
		}
	}
	
	return agents, nil
}

// UpdateAgentHeartbeat updates an agent's last seen timestamp
func (s *AgentService) UpdateAgentHeartbeat(ctx context.Context, id string) error {
	s.logger.Debug("Updating agent heartbeat: %s", id)
	
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return err
	}
	
	agent.UpdateLastSeen()
	
	return s.storage.Update(ctx, agent)
}

// HealthCheck checks the health of inactive agents
func (s *AgentService) HealthCheck(ctx context.Context, timeout time.Duration) error {
	s.logger.Debug("Performing health check with timeout: %v", timeout)
	
	agents, err := s.ListAllAgents(ctx)
	if err != nil {
		return err
	}
	
	cutoff := time.Now().Add(-timeout)
	for _, agent := range agents {
		if agent.LastSeen.Before(cutoff) && agent.IsActive() {
			s.logger.Warn("Agent %s is inactive, last seen: %v", agent.GetID(), agent.LastSeen)
			agent.SetStatus("inactive")
			if err := s.storage.Update(ctx, agent); err != nil {
				s.logger.Error("Failed to update agent status: %v", err)
			}
		}
	}
	
	return nil
}

// validateRegisterRequest validates the registration request
func (s *AgentService) validateRegisterRequest(req *models.RegisterRequest) error {
	if utils.IsEmpty(req.AgentCard.Name) {
		return fmt.Errorf("agent name is required")
	}

	if utils.IsEmpty(req.AgentCard.Version) {
		return fmt.Errorf("agent version is required")
	}

	if utils.IsEmpty(req.AgentCard.URL) {
		return fmt.Errorf("agent URL is required")
	}

	if utils.IsEmpty(req.Host) {
		return fmt.Errorf("host is required")
	}

	if req.Port <= 0 || req.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", req.Port)
	}

	if len(req.AgentCard.Skills) == 0 {
		return fmt.Errorf("at least one skill is required")
	}

	for _, skill := range req.AgentCard.Skills {
		if utils.IsEmpty(skill.ID) || utils.IsEmpty(skill.Name) {
			return fmt.Errorf("skill ID and name are required")
		}
	}

	return nil
}

// matchesQuery checks if an agent matches the search query
func (s *AgentService) matchesQuery(agent *models.RegisteredAgent, query string) bool {
	normalizedQuery := utils.NormalizeString(query)
	if normalizedQuery == "" {
		return true
	}
	
	// Exact match agent name
	if utils.ExactMatchIgnoreCase(agent.AgentCard.Name, normalizedQuery) {
		return true
	}
	
	// Partial match agent name
	if utils.ContainsIgnoreCase(agent.AgentCard.Name, normalizedQuery) {
		return true
	}
	
	// Match description
	if utils.ContainsIgnoreCase(agent.AgentCard.Description, normalizedQuery) {
		return true
	}
	
	// Match skills
	for _, skill := range agent.AgentCard.Skills {
		if utils.MatchFields(normalizedQuery, skill.Name, skill.Description) {
			return true
		}
		if utils.MatchTags(normalizedQuery, skill.Tags) {
			return true
		}
	}
	
	return false
}

// AnalyzeContext performs dynamic context analysis and routing
func (s *AgentService) AnalyzeContext(ctx context.Context, req *models.ContextAnalysisRequest) (*models.ContextAnalysisResponse, error) {
	s.logger.Info("Performing dynamic context analysis for: %s", req.NeedDescription)
	
	// Check if context analyzer is available
	if s.contextAnalyzer == nil {
		s.logger.Warn("Context analyzer not configured, dynamic context analysis disabled")
		return &models.ContextAnalysisResponse{
			Success: false,
			Message: "Dynamic context analysis is not enabled",
		}, nil
	}
	
	// Get initial analysis from context analyzer
	response, err := s.contextAnalyzer.AnalyzeAndRoute(ctx, req)
	if err != nil {
		return nil, err
	}
	
	// If routing was successful, enhance with Hub discovery
	if response.Success && response.RouteResult != nil {
		enhancedResult, err := s.enhanceRouteResultWithHubDiscovery(ctx, response.RouteResult)
		if err != nil {
			s.logger.Warn("Hub discovery failed, using basic route result: %v", err)
			// Continue with basic result rather than fail completely
		} else {
			response.RouteResult = enhancedResult
			s.logger.Info("Enhanced route result with Hub discovery")
		}
	}
	
	return response, nil
}

// enhanceRouteResultWithHubDiscovery enhances route result with Hub agent discovery
func (s *AgentService) enhanceRouteResultWithHubDiscovery(ctx context.Context, routeResult *models.RouteResult) (*models.RouteResult, error) {
	s.logger.Debug("Enhancing route result with Hub discovery for skill: %s", routeResult.SkillID)
	
	// Use Hub's discover agents functionality to get complete agent info
	discoverReq := &models.DiscoverRequest{
		Query: routeResult.SkillID, // Use skill ID as query
	}
	
	discoverResponse, err := s.DiscoverAgents(ctx, discoverReq)
	if err != nil {
		return nil, fmt.Errorf("Hub discovery failed: %w", err)
	}
	
	if len(discoverResponse.Agents) == 0 {
		return nil, fmt.Errorf("no agents found for skill %s", routeResult.SkillID)
	}
	
	// Find the agent that matches our route result
	var matchedAgent *models.AgentCard
	for i, agent := range discoverResponse.Agents {
		// Check if agent has the skill we're looking for
		for _, skill := range agent.Skills {
			if skill.ID == routeResult.SkillID {
				matchedAgent = &discoverResponse.Agents[i]
				break
			}
		}
		if matchedAgent != nil {
			break
		}
	}
	
	if matchedAgent == nil {
		// If no exact match, use the first discovered agent
		matchedAgent = &discoverResponse.Agents[0]
	}
	
	// Create enhanced route result
	enhancedResult := &models.RouteResult{
		AgentURL:      routeResult.AgentURL,
		SkillID:       routeResult.SkillID,
		SkillName:     routeResult.SkillName,
		AgentInfo:     matchedAgent, // Complete agent information from Hub
		AgentResponse: nil,          // Will be populated when actual call is made
	}
	
	s.logger.Debug("Successfully enhanced route result with agent info: %s", matchedAgent.Name)
	return enhancedResult, nil
}

// IsContextAnalysisEnabled returns true if dynamic context analysis is enabled
func (s *AgentService) IsContextAnalysisEnabled() bool {
	return s.contextAnalyzer != nil
}