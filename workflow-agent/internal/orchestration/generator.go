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

	"seata-go-ai-workflow-agent/pkg/errors"
	"seata-go-ai-workflow-agent/pkg/llm"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// WorkflowGenerator generates Seata Saga and React Flow JSONs
type WorkflowGenerator struct {
	llmClient *llm.Client
}

// NewWorkflowGenerator creates a new workflow generator
func NewWorkflowGenerator(llmClient *llm.Client) *WorkflowGenerator {
	return &WorkflowGenerator{
		llmClient: llmClient,
	}
}

// WorkflowGenerationResponse represents LLM's workflow generation response
type WorkflowGenerationResponse struct {
	WorkflowName        string             `json:"workflow_name"`
	WorkflowDescription string             `json:"workflow_description"`
	SeataJSON           *SeataStateMachine `json:"seata_json"`
	ReactFlow           *ReactFlowGraph    `json:"react_flow"`
}

// GenerateWorkflow generates both Seata Saga and React Flow JSONs
func (wg *WorkflowGenerator) GenerateWorkflow(
	ctx context.Context,
	scenario string,
	capabilities []DiscoveredCapability,
) (*WorkflowGenerationResponse, error) {
	logger.Info("generating workflow from discovered capabilities")

	// Filter only found capabilities
	foundCapabilities := make([]DiscoveredCapability, 0)
	for _, cap := range capabilities {
		if cap.Found {
			foundCapabilities = append(foundCapabilities, cap)
		}
	}

	if len(foundCapabilities) == 0 {
		return nil, errors.New(errors.ErrInvalidArgument, "no capabilities found to generate workflow")
	}

	prompt := WorkflowOrchestrationPrompt(scenario, foundCapabilities)

	resp, err := wg.llmClient.Generate(ctx, &llm.GenerateRequest{
		Messages: []llm.Message{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: floatPtr(0.7), // More creative for workflow design
	})
	if err != nil {
		return nil, errors.Wrap(errors.ErrGeneration, "failed to generate workflow", err)
	}

	var workflow WorkflowGenerationResponse
	if err := json.Unmarshal([]byte(resp.Content), &workflow); err != nil {
		logger.WithField("response", resp.Content).Error("failed to parse workflow response")
		return nil, errors.Wrap(errors.ErrInvalidArgument, "invalid workflow generation response", err)
	}

	// Validate generated workflow
	if err := wg.validateWorkflow(&workflow); err != nil {
		return nil, errors.Wrap(errors.ErrInvalidArgument, "workflow validation failed", err)
	}

	logger.WithField("workflow_name", workflow.WorkflowName).
		WithField("states_count", len(workflow.SeataJSON.States)).
		WithField("nodes_count", len(workflow.ReactFlow.Nodes)).
		Info("workflow generation completed")

	return &workflow, nil
}

// validateWorkflow validates the generated workflow structure
func (wg *WorkflowGenerator) validateWorkflow(workflow *WorkflowGenerationResponse) error {
	if workflow.SeataJSON == nil {
		return errors.New(errors.ErrInvalidArgument, "seata_json is missing")
	}

	if workflow.ReactFlow == nil {
		return errors.New(errors.ErrInvalidArgument, "react_flow is missing")
	}

	// Validate Seata JSON
	if workflow.SeataJSON.Name == "" {
		return errors.New(errors.ErrInvalidArgument, "workflow name is empty")
	}

	if workflow.SeataJSON.StartState == "" {
		return errors.New(errors.ErrInvalidArgument, "start state is not defined")
	}

	if len(workflow.SeataJSON.States) == 0 {
		return errors.New(errors.ErrInvalidArgument, "no states defined in workflow")
	}

	// Check if StartState exists
	if _, exists := workflow.SeataJSON.States[workflow.SeataJSON.StartState]; !exists {
		return errors.Newf(errors.ErrInvalidArgument, "start state '%s' not found in states", workflow.SeataJSON.StartState)
	}

	// Validate that there's at least one terminal state (Succeed or Fail)
	hasTerminal := false
	for _, state := range workflow.SeataJSON.States {
		if state.Type == "Succeed" || state.Type == "Fail" {
			hasTerminal = true
			break
		}
	}
	if !hasTerminal {
		return errors.New(errors.ErrInvalidArgument, "workflow must have at least one terminal state")
	}

	// Validate React Flow
	if len(workflow.ReactFlow.Nodes) == 0 {
		return errors.New(errors.ErrInvalidArgument, "react flow has no nodes")
	}

	logger.Debug("workflow validation passed")
	return nil
}

// EnhanceWorkflowWithBestPractices applies best practices to the workflow
func (wg *WorkflowGenerator) EnhanceWorkflowWithBestPractices(workflow *SeataStateMachine) {
	// Ensure all ServiceTask states have compensation
	for name, state := range workflow.States {
		if state.Type == "ServiceTask" && state.CompensateState == "" {
			logger.WithField("state", name).Warn("ServiceTask without compensation detected")
		}

		// Ensure all ServiceTask have Catch blocks
		if state.Type == "ServiceTask" && len(state.Catch) == 0 {
			logger.WithField("state", name).Warn("ServiceTask without error handling detected")
		}
	}

	// Check for CompensationTrigger
	hasCompensationTrigger := false
	for _, state := range workflow.States {
		if state.Type == "CompensationTrigger" {
			hasCompensationTrigger = true
			break
		}
	}

	if !hasCompensationTrigger {
		logger.Warn("workflow missing CompensationTrigger state")
	}
}

// floatPtr returns pointer to float32
func floatPtr(f float32) *float32 {
	return &f
}

// FormatWorkflowSummary creates a human-readable workflow summary
func FormatWorkflowSummary(workflow *WorkflowGenerationResponse) string {
	summary := "=== Workflow Summary ===\n"
	summary += "Name: " + workflow.WorkflowName + "\n"
	summary += "Description: " + workflow.WorkflowDescription + "\n\n"

	if workflow.SeataJSON != nil {
		summary += "Seata Saga State Machine:\n"
		summary += "  Version: " + workflow.SeataJSON.Version + "\n"
		summary += "  Start State: " + workflow.SeataJSON.StartState + "\n"
		summary += "  Total States: " + string(rune(len(workflow.SeataJSON.States))) + "\n\n"

		summary += "  States:\n"
		for name, state := range workflow.SeataJSON.States {
			summary += "    - " + name + " (" + state.Type + ")\n"
			if state.ServiceName != "" {
				summary += "      Service: " + state.ServiceName + "." + state.ServiceMethod + "\n"
			}
			if state.CompensateState != "" {
				summary += "      Compensation: " + state.CompensateState + "\n"
			}
		}
	}

	if workflow.ReactFlow != nil {
		summary += "\nReact Flow Visualization:\n"
		summary += "  Nodes: " + string(rune(len(workflow.ReactFlow.Nodes))) + "\n"
		summary += "  Edges: " + string(rune(len(workflow.ReactFlow.Edges))) + "\n"
	}

	return summary
}
