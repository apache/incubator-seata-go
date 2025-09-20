package storage

import (
	"fmt"
	"strings"

	"agenthub/pkg/models"
	"agenthub/pkg/utils"
)

// ServiceInstance represents the NamingServer service instance
type ServiceInstance struct {
	Addr string
	Port int
}

// SkillServiceMapper handles mapping between AgentCard skills and NamingServer services
type SkillServiceMapper struct {
	logger *utils.Logger
}

// NewSkillServiceMapper creates a new skill service mapper
func NewSkillServiceMapper() *SkillServiceMapper {
	return &SkillServiceMapper{
		logger: utils.WithField("component", "skill-mapper"),
	}
}

// SkillMapping represents a mapping between skill and service instance
type SkillMapping struct {
	SkillName       string
	HostUrl         string
	VGroup          string  // Independent vGroup for this skill
	ServiceInstance *ServiceInstance
	SkillInfo       models.AgentSkill
}

// ExtractSkillMappings extracts skill-URL mappings from AgentCard
func (m *SkillServiceMapper) ExtractSkillMappings(agentCard models.AgentCard) []SkillMapping {
	var mappings []SkillMapping
	
	for _, skill := range agentCard.Skills {
		// Extract port from URL
		port := m.extractPortFromURL(agentCard.URL)
		
		mapping := SkillMapping{
			SkillName: skill.ID,    // 使用skill.ID作为主键，保持一致性
			HostUrl:   agentCard.URL,
			VGroup:    m.getVGroupForSkill(skill.ID),  // Generate independent vGroup
			ServiceInstance: &ServiceInstance{
				Addr: agentCard.URL,
				Port: port,
			},
			SkillInfo: skill,
		}
		mappings = append(mappings, mapping)
		
		m.logger.Debug("Extracted skill mapping: %s -> %s", skill.Name, agentCard.URL)
	}
	
	return mappings
}

// extractPortFromURL extracts port from URL, returns default if not found
func (m *SkillServiceMapper) extractPortFromURL(url string) int {
	// Simple port extraction logic
	if strings.Contains(url, ":8080") {
		return 8080
	} else if strings.Contains(url, ":9000") {
		return 9000
	} else if strings.HasPrefix(url, "https") {
		return 443
	} else if strings.HasPrefix(url, "http") {
		return 80
	}
	return 80 // default port
}

// getVGroupForSkill generates independent vGroup name for each skill
func (m *SkillServiceMapper) getVGroupForSkill(skillName string) string {
	return fmt.Sprintf("skill-%s", skillName)
}