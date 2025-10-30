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

package tools

import (
	"context"
)

import (
	"agenthub/pkg/client"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

import (
	"seata-go-ai-workflow-agent/pkg/errors"
	"seata-go-ai-workflow-agent/pkg/hubclient"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// DiscoveryInput represents the input for discovering agents from natural language needs
type DiscoveryInput struct {
	NeedDescription string `json:"need_description"`
	UserContext     string `json:"user_context,omitempty"`
}

// DiscoveryOutput represents discovered agents matching the needs
type DiscoveryOutput struct {
	Agents  []client.AgentCard `json:"agents"`
	Message string             `json:"message,omitempty"`
}

// DefineAgentDiscoveryTool registers the agent discovery tool with genkit
func DefineAgentDiscoveryTool(g *genkit.Genkit, hub *hubclient.HubClient) (ai.Tool, error) {
	if g == nil {
		return nil, errors.ConfigError("genkit instance cannot be nil")
	}

	if hub == nil {
		return nil, errors.ConfigError("hub client cannot be nil")
	}

	logger.Info("defining agent discovery tool")

	var tool = genkit.DefineTool[*DiscoveryInput, *DiscoveryOutput](
		g,
		"discover-agents",
		"Discovers available agents from the hub based on natural language capability needs. "+
			"Analyzes the described requirements and returns agents that can fulfill those capabilities.",
		func(ctx *ai.ToolContext, input *DiscoveryInput) (*DiscoveryOutput, error) {
			return discoverAgents(ctx, hub, input)
		},
	)
	logger.Info("agent discovery tool registered")

	return tool, nil
}

// discoverAgents performs the discovery workflow
func discoverAgents(ctx context.Context, hub *hubclient.HubClient, input *DiscoveryInput) (*DiscoveryOutput, error) {
	if input == nil || input.NeedDescription == "" {
		return nil, errors.InvalidArgumentError("need_description is required")
	}

	logger.WithField("needDescription", input.NeedDescription).Info("discovering agents")

	// Analyze context to identify required skills
	analysisResp, err := hub.AnalyzeContext(ctx, input.NeedDescription, input.UserContext)
	if err != nil {
		logger.WithField("error", err).Error("context analysis failed")
		return nil, errors.Wrap(errors.ErrConnection, "context analysis failed", err)
	}

	if !analysisResp.Success {
		logger.WithField("message", analysisResp.Message).Warn("analysis unsuccessful")
		return &DiscoveryOutput{
			Agents:  []client.AgentCard{},
			Message: analysisResp.Message,
		}, nil
	}

	matchedSkills := analysisResp.MatchedSkills
	if len(matchedSkills) == 0 {
		logger.Info("no skills matched")
		return &DiscoveryOutput{
			Agents:  []client.AgentCard{},
			Message: "No matching skills found",
		}, nil
	}

	logger.WithField("skillCount", len(matchedSkills)).Info("skills identified")

	// Discover agents for each matched skill
	agentMap := make(map[string]client.AgentCard)

	for _, skill := range matchedSkills {
		logger.WithField("skill", skill.Name).Debug("discovering agents")

		agents, err := hub.Discover(ctx, skill.Name)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"skill": skill.Name,
				"error": err,
			}).Warn("discovery failed for skill")
			continue
		}

		for _, agent := range agents {
			agentMap[agent.Name] = agent
		}
	}

	// Convert map to slice
	agents := make([]client.AgentCard, 0, len(agentMap))
	for _, agent := range agentMap {
		agents = append(agents, agent)
	}

	logger.WithField("agentCount", len(agents)).Info("discovery completed")

	message := analysisResp.Message
	if analysisResp.AnalysisResult != nil && analysisResp.AnalysisResult.SuggestedWorkflow != "" {
		message = analysisResp.AnalysisResult.SuggestedWorkflow
	}

	return &DiscoveryOutput{
		Agents:  agents,
		Message: message,
	}, nil
}
