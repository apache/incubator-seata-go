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

package storage

import (
	"context"
	"fmt"
	"sync"

	"agenthub/pkg/models"
	"agenthub/pkg/utils"
)

// NamingserverRegistry interface from seata-go
type NamingserverRegistry interface {
	Register(instance *ServiceInstance) error
	Deregister(instance *ServiceInstance) error
	doHealthCheck(addr string) bool
	RefreshToken(addr string) error
	RefreshGroup(vGroup string) error
	Watch(vGroup string) (bool, error)
	Lookup(key string) ([]*ServiceInstance, error)
}

// SkillBasedNamingServerStorage implements storage interface using NamingServer for skill-based discovery
type SkillBasedNamingServerStorage struct {
	registry NamingserverRegistry
	mapper   *SkillServiceMapper
	logger   *utils.Logger

	// Multi-dimensional cache
	skillToUrl    map[string]string                  // skill_name -> host_url
	skillToVGroup map[string]string                  // skill_name -> vGroup
	agentToSkills map[string][]string                // agent_id -> []skill_names
	skillToAgent  map[string]string                  // skill_name -> agent_id
	agentCache    map[string]*models.RegisteredAgent // agent_id -> agent

	// Global aggregated agent card containing all registered skills
	globalCard *models.AgentCard // Global agent card with all skills

	cacheMutex sync.RWMutex
}

// NewSkillBasedNamingServerStorage creates a new skill-based NamingServer storage
func NewSkillBasedNamingServerStorage(registry NamingserverRegistry) *SkillBasedNamingServerStorage {
	// Initialize global agent card
	globalCard := &models.AgentCard{
		Name:        "GlobalAgentCard",
		Description: "Aggregated agent card containing all registered agent skills",
		Version:     "1.0.0",
		Skills:      []models.AgentSkill{},
	}

	return &SkillBasedNamingServerStorage{
		registry:      registry,
		mapper:        NewSkillServiceMapper(),
		logger:        utils.WithField("component", "skill-namingserver-storage"),
		skillToUrl:    make(map[string]string),
		skillToVGroup: make(map[string]string),
		agentToSkills: make(map[string][]string),
		skillToAgent:  make(map[string]string),
		agentCache:    make(map[string]*models.RegisteredAgent),
		globalCard:    globalCard,
	}
}

// Create registers an agent and all its skills to NamingServer using independent vGroups
func (s *SkillBasedNamingServerStorage) Create(ctx context.Context, resource Resource) error {
	agent, ok := resource.(*models.RegisteredAgent)
	if !ok {
		return fmt.Errorf("unsupported resource type")
	}

	agentId := agent.GetID()
	s.logger.Info("Registering agent %s: mapping %d skills to %s",
		agentId, len(agent.AgentCard.Skills), agent.AgentCard.URL)

	// Extract skill mappings with independent vGroups
	skillMappings := s.mapper.ExtractSkillMappings(agent.AgentCard)
	if len(skillMappings) == 0 {
		return fmt.Errorf("agent %s has no skills to register", agentId)
	}

	// Track registered skills for rollback
	var registeredSkills []SkillMapping

	// Register each skill to its own vGroup in NamingServer
	for _, mapping := range skillMappings {
		s.logger.Debug("Registering skill '%s' -> %s to vGroup '%s'",
			mapping.SkillName, mapping.HostUrl, mapping.VGroup)

		// 为Mock设置vGroup上下文（在真实实现中，这会通过其他方式处理）
		if mockRegistry, ok := s.registry.(*MockNamingserverRegistry); ok {
			mockRegistry.SetVGroup(mapping.VGroup)
		}

		if err := s.registry.Register(mapping.ServiceInstance); err != nil {
			s.logger.Error("Failed to register skill %s to vGroup %s: %v",
				mapping.SkillName, mapping.VGroup, err)

			// Rollback registered skills
			s.rollbackSkillRegistrations(registeredSkills)
			return fmt.Errorf("failed to register skill %s to NamingServer: %w", mapping.SkillName, err)
		}

		registeredSkills = append(registeredSkills, mapping)
		s.logger.Debug("Successfully registered skill %s to vGroup %s", mapping.SkillName, mapping.VGroup)
	}

	// Update caches
	s.updateCaches(agent, skillMappings)

	s.logger.Info("Successfully registered agent %s with %d skills", agentId, len(skillMappings))
	return nil
}

// Get retrieves an agent by ID
func (s *SkillBasedNamingServerStorage) Get(ctx context.Context, id string) (Resource, error) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	if agent, exists := s.agentCache[id]; exists {
		return agent, nil
	}
	return nil, fmt.Errorf("agent %s not found", id)
}

// Update updates an agent intelligently (only recreate if skills changed)
func (s *SkillBasedNamingServerStorage) Update(ctx context.Context, resource Resource) error {
	if agent, ok := resource.(*models.RegisteredAgent); ok {
		return s.updateAgent(ctx, agent)
	}
	return fmt.Errorf("unsupported resource type")
}

// updateAgent performs intelligent agent update
func (s *SkillBasedNamingServerStorage) updateAgent(ctx context.Context, newAgent *models.RegisteredAgent) error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	agentId := newAgent.GetID()

	// Get existing agent from cache
	existingAgent, exists := s.agentCache[agentId]
	if !exists {
		// Agent doesn't exist, treat as create
		s.cacheMutex.Unlock()
		return s.Create(ctx, newAgent)
	}

	// Check if skills have changed (structural change)
	skillsChanged := s.hasSkillsChanged(existingAgent.AgentCard.Skills, newAgent.AgentCard.Skills)

	if skillsChanged {
		// Skills changed, need to re-register to NamingServer
		s.logger.Debug("Agent %s skills changed, re-registering to NamingServer", agentId)
		s.cacheMutex.Unlock()

		// Use the old delete+create approach for structural changes
		if err := s.Delete(ctx, agentId); err != nil {
			s.logger.Warn("Failed to delete agent for skill update: %v", err)
		}
		return s.Create(ctx, newAgent)
	}

	// Only metadata changed (like heartbeat), update in-place
	s.logger.Debug("Agent %s metadata updated, no NamingServer re-registration needed", agentId)
	s.agentCache[agentId] = newAgent

	return nil
}

// hasSkillsChanged compares two skill arrays to detect structural changes
func (s *SkillBasedNamingServerStorage) hasSkillsChanged(oldSkills, newSkills []models.AgentSkill) bool {
	if len(oldSkills) != len(newSkills) {
		return true
	}

	// Create maps for efficient comparison
	oldSkillMap := make(map[string]models.AgentSkill)
	for _, skill := range oldSkills {
		oldSkillMap[skill.ID] = skill
	}

	for _, newSkill := range newSkills {
		oldSkill, exists := oldSkillMap[newSkill.ID]
		if !exists {
			return true // New skill added
		}

		// Check if skill content changed (name, description, tags, etc.)
		if !s.skillsEqual(oldSkill, newSkill) {
			return true
		}
	}

	return false
}

// skillsEqual compares two skills for equality
func (s *SkillBasedNamingServerStorage) skillsEqual(skill1, skill2 models.AgentSkill) bool {
	if skill1.ID != skill2.ID ||
		skill1.Name != skill2.Name ||
		skill1.Description != skill2.Description {
		return false
	}

	// Compare tags
	if len(skill1.Tags) != len(skill2.Tags) {
		return false
	}

	tagMap := make(map[string]bool)
	for _, tag := range skill1.Tags {
		tagMap[tag] = true
	}

	for _, tag := range skill2.Tags {
		if !tagMap[tag] {
			return false
		}
	}

	return true
}

// Delete deregisters an agent and all its skills from NamingServer
func (s *SkillBasedNamingServerStorage) Delete(ctx context.Context, agentId string) error {
	s.logger.Info("Deregistering agent %s", agentId)

	// Get skills for this agent
	s.cacheMutex.RLock()
	skillNames, exists := s.agentToSkills[agentId]
	s.cacheMutex.RUnlock()

	if !exists {
		return fmt.Errorf("agent %s not found", agentId)
	}

	// Deregister each skill from its vGroup
	var errors []error
	for _, skillName := range skillNames {
		if err := s.deregisterSkill(skillName); err != nil {
			errors = append(errors, err)
		}
	}

	// Clean up caches
	s.cleanupCaches(agentId, skillNames)

	if len(errors) > 0 {
		s.logger.Warn("Some skills failed to deregister: %v", errors)
		return fmt.Errorf("partial deregistration failure: %v", errors)
	}

	s.logger.Info("Successfully deregistered agent %s with %d skills", agentId, len(skillNames))
	return nil
}

// List returns all cached agents
func (s *SkillBasedNamingServerStorage) List(ctx context.Context, filters map[string]interface{}) ([]Resource, error) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	var resources []Resource
	for _, agent := range s.agentCache {
		resources = append(resources, agent)
	}

	return resources, nil
}

// Watch is not implemented yet
func (s *SkillBasedNamingServerStorage) Watch(ctx context.Context, filters map[string]interface{}) (<-chan Event, error) {
	return nil, fmt.Errorf("Watch not implemented yet")
}

// Close closes the storage
func (s *SkillBasedNamingServerStorage) Close() error {
	s.logger.Info("Closing skill-based NamingServer storage")
	return nil
}

// DiscoverUrlBySkill discovers host URL by skill name using independent vGroup
func (s *SkillBasedNamingServerStorage) DiscoverUrlBySkill(ctx context.Context, skillName string) (string, error) {
	s.logger.Debug("Discovering host URL for skill: %s", skillName)

	// 1. Try lookup from NamingServer using skill-specific vGroup
	if hostUrl := s.lookupFromNamingServer(skillName); hostUrl != "" {
		s.logger.Info("Found skill %s in NamingServer: %s", skillName, hostUrl)
		return hostUrl, nil
	}

	// 2. Fallback to local cache
	s.cacheMutex.RLock()
	hostUrl, exists := s.skillToUrl[skillName]
	s.cacheMutex.RUnlock()

	if exists {
		s.logger.Info("Found skill %s in local cache: %s", skillName, hostUrl)
		return hostUrl, nil
	}

	s.logger.Warn("Skill %s not found anywhere", skillName)
	return "", fmt.Errorf("skill %s not found", skillName)
}

// lookupFromNamingServer looks up skill from NamingServer using independent vGroup
func (s *SkillBasedNamingServerStorage) lookupFromNamingServer(skillName string) string {
	// Use skill-specific vGroup
	vGroup := s.getVGroupForSkill(skillName)

	instances, err := s.registry.Lookup(vGroup)
	if err != nil {
		s.logger.Warn("Failed to lookup skill %s from vGroup %s: %v", skillName, vGroup, err)
		return ""
	}

	if len(instances) == 0 {
		s.logger.Debug("No instances found for skill %s in vGroup %s", skillName, vGroup)
		return ""
	}

	// Return the first available instance
	hostUrl := instances[0].Addr
	s.logger.Debug("Found skill %s in NamingServer vGroup %s: %s", skillName, vGroup, hostUrl)
	return hostUrl
}

// getVGroupForSkill generates vGroup name for a specific skill
func (s *SkillBasedNamingServerStorage) getVGroupForSkill(skillName string) string {
	return fmt.Sprintf("skill-%s", skillName)
}

// updateCaches updates all cache mappings
func (s *SkillBasedNamingServerStorage) updateCaches(agent *models.RegisteredAgent, mappings []SkillMapping) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	agentId := agent.GetID()
	var skillNames []string

	// Update all mappings
	for _, mapping := range mappings {
		s.skillToUrl[mapping.SkillName] = mapping.HostUrl
		s.skillToVGroup[mapping.SkillName] = mapping.VGroup
		s.skillToAgent[mapping.SkillName] = agentId
		skillNames = append(skillNames, mapping.SkillName)
	}

	s.agentToSkills[agentId] = skillNames
	s.agentCache[agentId] = agent

	// Update global agent card with new skills
	s.updateGlobalCard(agent.AgentCard.Skills)
}

// updateGlobalCard adds new skills to the global agent card
func (s *SkillBasedNamingServerStorage) updateGlobalCard(newSkills []models.AgentSkill) {
	// Note: cacheMutex is already locked by caller

	for _, newSkill := range newSkills {
		// Check if skill already exists (by ID)
		exists := false
		for i, existingSkill := range s.globalCard.Skills {
			if existingSkill.ID == newSkill.ID {
				// Update existing skill
				s.globalCard.Skills[i] = newSkill
				exists = true
				break
			}
		}

		if !exists {
			// Add new skill
			s.globalCard.Skills = append(s.globalCard.Skills, newSkill)
		}
	}

	s.logger.Debug("Global card updated with %d skills, total skills: %d",
		len(newSkills), len(s.globalCard.Skills))
}

// cleanupCaches removes cache entries
func (s *SkillBasedNamingServerStorage) cleanupCaches(agentId string, skillNames []string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Clean up all mappings
	for _, skillName := range skillNames {
		delete(s.skillToUrl, skillName)
		delete(s.skillToVGroup, skillName)
		delete(s.skillToAgent, skillName)
	}

	delete(s.agentToSkills, agentId)
	delete(s.agentCache, agentId)

	// Remove skills from global agent card
	s.removeFromGlobalCard(skillNames)
}

// removeFromGlobalCard removes skills from the global agent card
func (s *SkillBasedNamingServerStorage) removeFromGlobalCard(skillNames []string) {
	// Note: cacheMutex is already locked by caller

	// Create a map for quick lookup
	removeMap := make(map[string]bool)
	for _, skillName := range skillNames {
		removeMap[skillName] = true
	}

	// Filter out skills to be removed
	var remainingSkills []models.AgentSkill
	for _, skill := range s.globalCard.Skills {
		if !removeMap[skill.ID] {
			remainingSkills = append(remainingSkills, skill)
		}
	}

	s.globalCard.Skills = remainingSkills
	s.logger.Debug("Removed %d skills from global card, remaining: %d",
		len(skillNames), len(s.globalCard.Skills))
}

// rollbackSkillRegistrations rolls back skill registrations
func (s *SkillBasedNamingServerStorage) rollbackSkillRegistrations(registeredMappings []SkillMapping) {
	s.logger.Warn("Rolling back %d skill registrations", len(registeredMappings))

	for _, mapping := range registeredMappings {
		if err := s.registry.Deregister(mapping.ServiceInstance); err != nil {
			s.logger.Error("Failed to rollback skill %s: %v", mapping.SkillName, err)
		}
	}
}

// deregisterSkill deregisters a single skill from its vGroup
func (s *SkillBasedNamingServerStorage) deregisterSkill(skillName string) error {
	s.cacheMutex.RLock()
	hostUrl, exists := s.skillToUrl[skillName]
	s.cacheMutex.RUnlock()

	if !exists {
		return fmt.Errorf("skill %s not found in cache", skillName)
	}

	// Construct ServiceInstance for deregistration
	port := s.mapper.extractPortFromURL(hostUrl)
	instance := &ServiceInstance{
		Addr: hostUrl,
		Port: port,
	}

	return s.registry.Deregister(instance)
}

// refreshNamingServer refreshes data from NamingServer for a specific skill
func (s *SkillBasedNamingServerStorage) refreshNamingServer(skillName string) error {
	vGroup := s.getVGroupForSkill(skillName)
	return s.registry.RefreshGroup(vGroup)
}

// GetSkillToAgent returns the skill to agent mapping (for AgentService access)
func (s *SkillBasedNamingServerStorage) GetSkillToAgent() map[string]string {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]string)
	for k, v := range s.skillToAgent {
		result[k] = v
	}
	return result
}

// GetAgentCache returns the agent cache (for AgentService access)
func (s *SkillBasedNamingServerStorage) GetAgentCache() map[string]*models.RegisteredAgent {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*models.RegisteredAgent)
	for k, v := range s.agentCache {
		result[k] = v
	}
	return result
}

// GetGlobalAgentCard returns the global agent card containing all registered skills
func (s *SkillBasedNamingServerStorage) GetGlobalAgentCard() *models.AgentCard {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	// Return a copy to prevent external modification
	cardCopy := &models.AgentCard{
		Name:        s.globalCard.Name,
		Description: s.globalCard.Description,
		Version:     s.globalCard.Version,
		Skills:      make([]models.AgentSkill, len(s.globalCard.Skills)),
	}
	copy(cardCopy.Skills, s.globalCard.Skills)

	return cardCopy
}

// FindSkillsByQuery finds skills in the global card that match the given query
func (s *SkillBasedNamingServerStorage) FindSkillsByQuery(query string) []models.AgentSkill {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	var matchingSkills []models.AgentSkill
	normalizedQuery := utils.NormalizeString(query)

	for _, skill := range s.globalCard.Skills {
		// Match by skill ID, name, description, or tags
		if utils.ContainsIgnoreCase(skill.ID, normalizedQuery) ||
			utils.ContainsIgnoreCase(skill.Name, normalizedQuery) ||
			utils.ContainsIgnoreCase(skill.Description, normalizedQuery) ||
			utils.MatchTags(normalizedQuery, skill.Tags) {
			matchingSkills = append(matchingSkills, skill)
		}
	}

	return matchingSkills
}
