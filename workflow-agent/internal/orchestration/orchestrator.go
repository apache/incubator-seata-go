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

	"seata-go-ai-workflow-agent/pkg/errors"
	"seata-go-ai-workflow-agent/pkg/hubclient"
	"seata-go-ai-workflow-agent/pkg/llm"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// OrchestratorConfig holds orchestration configuration
type OrchestratorConfig struct {
	MaxRetryCount int `json:"max_retry_count"`
}

// Orchestrator implements ReAct pattern for workflow orchestration
type Orchestrator struct {
	config             OrchestratorConfig
	llmClient          *llm.Client
	hubClient          *hubclient.HubClient
	capabilityAnalyzer *CapabilityAnalyzer
	workflowGenerator  *WorkflowGenerator
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(
	config OrchestratorConfig,
	llmClient *llm.Client,
	hubClient *hubclient.HubClient,
) *Orchestrator {
	return &Orchestrator{
		config:             config,
		llmClient:          llmClient,
		hubClient:          hubClient,
		capabilityAnalyzer: NewCapabilityAnalyzer(llmClient, hubClient),
		workflowGenerator:  NewWorkflowGenerator(llmClient),
	}
}

// ReActResponse represents a ReAct step response from LLM
type ReActResponse struct {
	Thought   string `json:"thought"`
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
}

// Orchestrate performs complete workflow orchestration using ReAct pattern
func (o *Orchestrator) Orchestrate(ctx context.Context, req ScenarioRequest) (*OrchestrationResult, error) {
	logger.WithField("scenario", req.Description).Info("starting workflow orchestration")

	result := &OrchestrationResult{
		Success:    false,
		RetryCount: 0,
	}

	// ReAct loop
	var trace ReActTrace
	var analysisResp *AnalysisResponse
	var discoveredCapabilities []DiscoveredCapability
	var minimumRequired int

	maxSteps := 20 // Prevent infinite loops
	for step := 1; step <= maxSteps; step++ {
		logger.WithField("step", step).Info("ReAct step")

		// Get next action from LLM
		action, err := o.getNextAction(ctx, req.Description, step, trace.Steps)
		if err != nil {
			logger.WithField("error", err).Error("failed to get next action")
			result.Message = fmt.Sprintf("failed to determine next action: %v", err)
			return result, err
		}

		logger.WithField("action", action.Action).
			WithField("thought", action.Thought).
			Info("next action determined")

		var observation string

		switch action.Action {
		case "analyze_capabilities":
			// Step 1: Analyze scenario to identify required capabilities
			analysisResp, err = o.capabilityAnalyzer.AnalyzeScenario(ctx, req)
			if err != nil {
				observation = fmt.Sprintf("Failed to analyze scenario: %v", err)
				result.Message = observation
				return result, err
			}
			minimumRequired = analysisResp.MinimumRequiredCount
			observation = fmt.Sprintf("Identified %d capabilities (%d required minimum). Analysis: %s",
				len(analysisResp.Capabilities), minimumRequired, analysisResp.Analysis)

		case "discover_agents":
			// Step 2: Discover agents for required capabilities
			if analysisResp == nil {
				observation = "Error: Must analyze capabilities first"
				break
			}
			discoveredCapabilities, err = o.capabilityAnalyzer.DiscoverCapabilities(ctx, analysisResp.Capabilities)
			if err != nil {
				observation = fmt.Sprintf("Failed to discover agents: %v", err)
				break
			}
			foundCount := CountFoundCapabilities(discoveredCapabilities)
			observation = fmt.Sprintf("Discovered %d/%d capabilities from hub",
				foundCount, len(discoveredCapabilities))
			result.Capabilities = discoveredCapabilities

		case "verify_requirements":
			// Step 3: Check if minimum requirements are met
			if len(discoveredCapabilities) == 0 {
				observation = "Error: Must discover agents first"
				break
			}
			checkResp, err := o.capabilityAnalyzer.CheckMinimumRequirements(
				ctx, req.Description, discoveredCapabilities, minimumRequired)
			if err != nil {
				observation = fmt.Sprintf("Failed to check requirements: %v", err)
				break
			}
			if checkResp.Sufficient {
				observation = fmt.Sprintf("Requirements met! Reason: %s", checkResp.Reason)
			} else {
				observation = fmt.Sprintf("Requirements NOT met. Reason: %s. Missing: %v. Suggestions: %s",
					checkResp.Reason, checkResp.MissingCritical, checkResp.Suggestions)
			}

		case "retry_discovery":
			// Step 4: Retry discovery with refined descriptions
			if result.RetryCount >= o.config.MaxRetryCount {
				observation = fmt.Sprintf("Max retry count (%d) reached. Cannot satisfy requirements.",
					o.config.MaxRetryCount)
				result.Message = observation
				trace.Steps = append(trace.Steps, ReActStep{
					Thought:     action.Thought,
					Action:      action.Action,
					Observation: observation,
				})
				trace.Conclusion = "Failed: Maximum retries exceeded without satisfying requirements"
				return result, errors.New(errors.ErrMaxRetriesExceeded, observation)
			}

			result.RetryCount++
			logger.WithField("retry_count", result.RetryCount).Info("retrying discovery")

			// Refine missing capabilities
			missing := GetMissingRequiredCapabilities(discoveredCapabilities)
			if len(missing) > 0 {
				refined, err := o.capabilityAnalyzer.RefineCapabilityDescription(ctx, missing[0])
				if err != nil {
					observation = fmt.Sprintf("Failed to refine capability: %v", err)
					break
				}

				// Update requirement and rediscover
				for i := range analysisResp.Capabilities {
					if analysisResp.Capabilities[i].Name == missing[0].Name {
						analysisResp.Capabilities[i] = *refined
						break
					}
				}

				discoveredCapabilities, err = o.capabilityAnalyzer.DiscoverCapabilities(ctx, analysisResp.Capabilities)
				if err != nil {
					observation = fmt.Sprintf("Failed to rediscover agents: %v", err)
					break
				}
				foundCount := CountFoundCapabilities(discoveredCapabilities)
				observation = fmt.Sprintf("Retry %d: Discovered %d/%d capabilities after refinement",
					result.RetryCount, foundCount, len(discoveredCapabilities))
				result.Capabilities = discoveredCapabilities
			} else {
				observation = "No missing required capabilities to refine"
			}

		case "orchestrate_workflow":
			// Step 5: Generate workflow
			if len(discoveredCapabilities) == 0 {
				observation = "Error: No capabilities available for workflow generation"
				break
			}

			workflow, err := o.workflowGenerator.GenerateWorkflow(ctx, req.Description, discoveredCapabilities)
			if err != nil {
				observation = fmt.Sprintf("Failed to generate workflow: %v", err)
				result.Message = observation
				break
			}

			result.SeataWorkflow = workflow.SeataJSON
			result.ReactFlow = workflow.ReactFlow
			observation = fmt.Sprintf("Workflow '%s' generated successfully with %d states and %d nodes",
				workflow.WorkflowName, len(workflow.SeataJSON.States), len(workflow.ReactFlow.Nodes))

		case "complete":
			// Task completed successfully
			result.Success = true
			result.Message = "Workflow orchestration completed successfully"
			observation = result.Message
			trace.Steps = append(trace.Steps, ReActStep{
				Thought:     action.Thought,
				Action:      action.Action,
				Observation: observation,
			})
			trace.Conclusion = "Success: Workflow orchestrated and generated"

			logger.WithField("retry_count", result.RetryCount).
				WithField("steps", len(trace.Steps)).
				Info("orchestration completed successfully")
			return result, nil

		case "fail":
			// Task failed
			result.Success = false
			result.Message = "Workflow orchestration failed: " + action.Thought
			observation = result.Message
			trace.Steps = append(trace.Steps, ReActStep{
				Thought:     action.Thought,
				Action:      action.Action,
				Observation: observation,
			})
			trace.Conclusion = "Failed: " + action.Thought
			return result, errors.New(errors.ErrOrchestration, result.Message)

		default:
			observation = fmt.Sprintf("Unknown action: %s", action.Action)
		}

		// Record step
		trace.Steps = append(trace.Steps, ReActStep{
			Thought:     action.Thought,
			Action:      action.Action,
			Observation: observation,
		})

		logger.WithField("observation", observation).Debug("step observation")
	}

	// Max steps reached
	result.Message = "Max orchestration steps reached without completion"
	trace.Conclusion = "Failed: Maximum steps exceeded"
	return result, errors.New(errors.ErrOrchestration, result.Message)
}

// getNextAction determines the next action using ReAct reasoning
func (o *Orchestrator) getNextAction(
	ctx context.Context,
	scenario string,
	currentStep int,
	previousSteps []ReActStep,
) (*ReActResponse, error) {
	prompt := ReActThinkingPrompt(scenario, currentStep, previousSteps)

	resp, err := o.llmClient.Generate(ctx, &llm.GenerateRequest{
		Messages: []llm.Message{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: floatPtr(0.3), // Lower temperature for more deterministic reasoning
	})
	if err != nil {
		return nil, errors.Wrap(errors.ErrGeneration, "failed to get next action", err)
	}

	var action ReActResponse
	if err := json.Unmarshal([]byte(resp.Content), &action); err != nil {
		logger.WithField("response", resp.Content).Error("failed to parse ReAct response")
		return nil, errors.Wrap(errors.ErrInvalidArgument, "invalid ReAct response format", err)
	}

	return &action, nil
}

// GetCapabilityAnalyzer returns the capability analyzer (for testing)
func (o *Orchestrator) GetCapabilityAnalyzer() *CapabilityAnalyzer {
	return o.capabilityAnalyzer
}

// GetWorkflowGenerator returns the workflow generator (for testing)
func (o *Orchestrator) GetWorkflowGenerator() *WorkflowGenerator {
	return o.workflowGenerator
}
