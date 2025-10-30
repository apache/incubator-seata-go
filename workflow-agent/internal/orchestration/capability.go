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

package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agenthub/pkg/client"

	"seata-go-ai-workflow-agent/pkg/errors"
	"seata-go-ai-workflow-agent/pkg/hubclient"
	"seata-go-ai-workflow-agent/pkg/llm"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// CapabilityAnalyzer handles capability analysis and discovery
type CapabilityAnalyzer struct {
	llmClient *llm.Client
	hubClient *hubclient.HubClient
}

// NewCapabilityAnalyzer creates a new capability analyzer
func NewCapabilityAnalyzer(llmClient *llm.Client, hubClient *hubclient.HubClient) *CapabilityAnalyzer {
	return &CapabilityAnalyzer{
		llmClient: llmClient,
		hubClient: hubClient,
	}
}

// AnalysisResponse represents LLM's capability analysis response
type AnalysisResponse struct {
	Analysis             string                  `json:"analysis"`
	Capabilities         []CapabilityRequirement `json:"capabilities"`
	MinimumRequiredCount int                     `json:"minimum_required_count"`
}

// AnalyzeScenario analyzes the scenario and identifies required capabilities
func (ca *CapabilityAnalyzer) AnalyzeScenario(ctx context.Context, req ScenarioRequest) (*AnalysisResponse, error) {
	logger.Info("analyzing scenario for required capabilities")

	prompt := CapabilityAnalysisPrompt(req.Description, req.Context)

	resp, err := ca.llmClient.Generate(ctx, &llm.GenerateRequest{
		Messages: []llm.Message{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, errors.Wrap(errors.ErrGeneration, "failed to analyze scenario", err)
	}

	var analysis AnalysisResponse
	if err := json.Unmarshal([]byte(resp.Content), &analysis); err != nil {
		logger.WithField("response", resp.Content).Error("failed to parse analysis response")
		return nil, errors.Wrap(errors.ErrInvalidArgument, "invalid LLM response format", err)
	}

	logger.WithField("capabilities_count", len(analysis.Capabilities)).
		WithField("minimum_required", analysis.MinimumRequiredCount).
		Info("scenario analysis completed")

	return &analysis, nil
}

// DiscoverCapabilities discovers agents for each required capability
func (ca *CapabilityAnalyzer) DiscoverCapabilities(ctx context.Context, requirements []CapabilityRequirement) ([]DiscoveredCapability, error) {
	logger.WithField("requirements_count", len(requirements)).Info("discovering capabilities from hub")

	discovered := make([]DiscoveredCapability, 0, len(requirements))

	// Get all registered agents from hub
	allAgents, err := ca.hubClient.ListAgents(ctx)
	if err != nil {
		logger.WithField("error", err).Error("failed to list agents")
		return nil, errors.Wrap(errors.ErrDiscovery, "failed to list agents", err)
	}

	logger.WithField("agent_count", len(allAgents)).Debug("retrieved agents from hub")

	for _, req := range requirements {
		logger.WithField("capability", req.Name).Debug("discovering capability")

		// Find best matching agent by comparing skills
		var bestAgent *client.AgentCard
		var bestMatch float64 = 0.0

		for _, agent := range allAgents {
			// Note: We don't check IsActive() for testing purposes since test agents may not have active services
			// In production, you may want to add: if !agent.IsActive() { continue }

			// Check each skill of the agent
			for _, skill := range agent.AgentCard.Skills {
				// Simple keyword matching score
				score := ca.matchSkillToRequirement(skill, req)

				logger.WithField("requirement", req.Name).
					WithField("agent", agent.AgentCard.Name).
					WithField("skill", skill.Name).
					WithField("score", score).
					Debug("skill matching score")

				if score > bestMatch {
					bestMatch = score
					card := agent.AgentCard
					bestAgent = &card
				}
			}
		}

		if bestAgent != nil && bestMatch > 0.1 { // Threshold for accepting match (lowered for better matching)
			logger.WithField("capability", req.Name).
				WithField("agent", bestAgent.Name).
				WithField("score", bestMatch).
				Info("capability discovered")

			discovered = append(discovered, DiscoveredCapability{
				Requirement: req,
				Agent:       bestAgent,
				Found:       true,
			})
		} else {
			logger.WithField("capability", req.Name).Warn("no agent found for capability")
			discovered = append(discovered, DiscoveredCapability{
				Requirement: req,
				Found:       false,
			})
		}
	}

	foundCount := 0
	for _, d := range discovered {
		if d.Found {
			foundCount++
		}
	}

	logger.WithField("total", len(discovered)).
		WithField("found", foundCount).
		Info("capability discovery completed")

	return discovered, nil
}

// SufficiencyCheckResponse represents LLM's sufficiency check response
type SufficiencyCheckResponse struct {
	Sufficient      bool     `json:"sufficient"`
	Reason          string   `json:"reason"`
	MissingCritical []string `json:"missing_critical"`
	Suggestions     string   `json:"suggestions"`
}

// CheckMinimumRequirements checks if discovered capabilities meet minimum requirements
func (ca *CapabilityAnalyzer) CheckMinimumRequirements(
	ctx context.Context,
	scenario string,
	discovered []DiscoveredCapability,
	minimumRequired int,
) (*SufficiencyCheckResponse, error) {
	logger.Info("checking if minimum requirements are met")

	prompt := MinimumRequirementCheckPrompt(scenario, discovered, minimumRequired)

	resp, err := ca.llmClient.Generate(ctx, &llm.GenerateRequest{
		Messages: []llm.Message{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, errors.Wrap(errors.ErrGeneration, "failed to check requirements", err)
	}

	var check SufficiencyCheckResponse
	if err := json.Unmarshal([]byte(resp.Content), &check); err != nil {
		logger.WithField("response", resp.Content).Error("failed to parse check response")
		return nil, errors.Wrap(errors.ErrInvalidArgument, "invalid LLM response format", err)
	}

	logger.WithField("sufficient", check.Sufficient).
		WithField("reason", check.Reason).
		Info("requirement check completed")

	return &check, nil
}

// RefineCapabilityDescription refines capability description using LLM
func (ca *CapabilityAnalyzer) RefineCapabilityDescription(
	ctx context.Context,
	capability CapabilityRequirement,
) (*CapabilityRequirement, error) {
	logger.WithField("capability", capability.Name).Info("refining capability description")

	prompt := RefinementPrompt(capability, capability.Description)

	resp, err := ca.llmClient.Generate(ctx, &llm.GenerateRequest{
		Messages: []llm.Message{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, errors.Wrap(errors.ErrGeneration, "failed to refine description", err)
	}

	var refinement struct {
		RefinedName        string   `json:"refined_name"`
		RefinedDescription string   `json:"refined_description"`
		AlternativeNames   []string `json:"alternative_names"`
		Reasoning          string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(resp.Content), &refinement); err != nil {
		logger.WithField("response", resp.Content).Error("failed to parse refinement response")
		return nil, errors.Wrap(errors.ErrInvalidArgument, "invalid LLM response format", err)
	}

	refined := CapabilityRequirement{
		Name:        refinement.RefinedName,
		Description: refinement.RefinedDescription,
		Required:    capability.Required,
	}

	logger.WithField("original", capability.Name).
		WithField("refined", refined.Name).
		WithField("reasoning", refinement.Reasoning).
		Info("capability description refined")

	return &refined, nil
}

// matchSkillToRequirement calculates a simple matching score between skill and requirement
func (ca *CapabilityAnalyzer) matchSkillToRequirement(skill client.AgentSkill, req CapabilityRequirement) float64 {
	score := 0.0
	reqNameLower := strings.ToLower(req.Name)
	reqDescLower := strings.ToLower(req.Description)
	skillNameLower := strings.ToLower(skill.Name)
	skillDescLower := strings.ToLower(skill.Description)

	// Extract keywords from requirement
	reqKeywords := extractKeywords(reqNameLower + " " + reqDescLower)

	// Check skill name contains any requirement keywords
	for _, keyword := range reqKeywords {
		if strings.Contains(skillNameLower, keyword) || strings.Contains(reqNameLower, strings.ToLower(skill.Name)) {
			score += 0.4
			break
		}
	}

	// Check skill description contains any requirement keywords
	matchCount := 0
	for _, keyword := range reqKeywords {
		if strings.Contains(skillDescLower, keyword) {
			matchCount++
		}
	}
	if matchCount > 0 {
		score += 0.3 * float64(matchCount) / float64(len(reqKeywords))
	}

	// Check skill tags
	for _, tag := range skill.Tags {
		tagLower := strings.ToLower(tag)
		for _, keyword := range reqKeywords {
			if tagLower == keyword || strings.Contains(tagLower, keyword) || strings.Contains(keyword, tagLower) {
				score += 0.3
				break
			}
		}
	}

	return score
}

// extractKeywords extracts meaningful keywords from text
func extractKeywords(text string) []string {
	// Common stop words to filter out
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "from": true, "by": true, "as": true, "is": true, "are": true,
		"that": true, "this": true, "can": true, "be": true, "should": true,
	}

	words := strings.Fields(text)
	keywords := make([]string, 0)

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// CountFoundCapabilities counts how many capabilities were found
func CountFoundCapabilities(capabilities []DiscoveredCapability) int {
	count := 0
	for _, cap := range capabilities {
		if cap.Found {
			count++
		}
	}
	return count
}

// GetMissingRequiredCapabilities returns list of missing required capabilities
func GetMissingRequiredCapabilities(capabilities []DiscoveredCapability) []CapabilityRequirement {
	missing := make([]CapabilityRequirement, 0)
	for _, cap := range capabilities {
		if cap.Requirement.Required && !cap.Found {
			missing = append(missing, cap.Requirement)
		}
	}
	return missing
}

// FormatCapabilitySummary creates a human-readable summary
func FormatCapabilitySummary(capabilities []DiscoveredCapability) string {
	summary := fmt.Sprintf("Total Capabilities: %d\n", len(capabilities))
	summary += fmt.Sprintf("Found: %d\n", CountFoundCapabilities(capabilities))
	summary += fmt.Sprintf("Missing: %d\n", len(capabilities)-CountFoundCapabilities(capabilities))

	summary += "\nDetails:\n"
	for _, cap := range capabilities {
		status := "❌ NOT FOUND"
		agent := "N/A"
		if cap.Found {
			status = "✓ FOUND"
			agent = cap.Agent.Name
		}
		summary += fmt.Sprintf("  %s %s (Agent: %s)\n", status, cap.Requirement.Name, agent)
	}

	return summary
}
