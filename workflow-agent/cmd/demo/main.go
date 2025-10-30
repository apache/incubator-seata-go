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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"seata-go-ai-workflow-agent/internal/config"
	"seata-go-ai-workflow-agent/pkg/hubclient"
	"seata-go-ai-workflow-agent/pkg/llm"
	"seata-go-ai-workflow-agent/pkg/logger"
	"seata-go-ai-workflow-agent/pkg/orchestration"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
)

var configPath = flag.String("config", "configs/config.yaml", "path to config file")

func main() {
	flag.Parse()

	// Load configuration
	cfg := config.LoadOrDefault(*configPath)

	// Initialize logger
	if err := logger.Init(logger.Config{
		Level:      cfg.Logger.Level,
		Format:     cfg.Logger.Format,
		OutputPath: cfg.Logger.OutputPath,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("=== Workflow Orchestration Demo ===")

	ctx := context.Background()

	// Initialize OpenAI-compatible provider
	oaiProvider := &openai.OpenAI{
		APIKey: cfg.LLM.APIKey,
		Opts: []option.RequestOption{
			option.WithBaseURL(cfg.LLM.BaseURL),
		},
	}

	// Initialize genkit with OpenAI provider
	g := genkit.Init(ctx, genkit.WithPlugins(oaiProvider))

	// Define custom model
	oaiProvider.DefineModel(cfg.LLM.DefaultModel, ai.ModelOptions{})

	// Build full model name with provider prefix
	fullModelName := "openai/" + cfg.LLM.DefaultModel

	// Initialize LLM client
	llmClient, err := llm.NewClient(&llm.Config{
		Genkit:       g,
		DefaultModel: fullModelName,
		APIKey:       cfg.LLM.APIKey,
		BaseURL:      cfg.LLM.BaseURL,
		Config:       cfg.LLM.Config,
	})
	if err != nil {
		logger.Fatalf("failed to initialize llm client: %v", err)
	}
	logger.Info("LLM client initialized successfully")

	// Initialize Hub client
	hubClient := hubclient.New(&hubclient.Config{
		BaseURL: cfg.Hub.BaseURL,
		Timeout: cfg.GetHubTimeout(),
	})
	logger.Info("Hub client initialized successfully")

	// Test hub connectivity
	if err := hubClient.HealthCheck(ctx); err != nil {
		logger.Warnf("hub health check failed (hub may not be running): %v", err)
		logger.Info("Please start agenthub first: cd ../agenthub && go run cmd/hub/main.go")
	} else {
		logger.Info("Hub is healthy and ready")
	}

	// Initialize orchestrator
	orchestrator := orchestration.NewOrchestrator(
		orchestration.OrchestratorConfig{
			MaxRetryCount: cfg.Orchestration.MaxRetryCount,
		},
		llmClient,
		hubClient,
	)
	logger.Info("Orchestrator initialized successfully")

	// Test scenario: Research report generation workflow
	scenario := orchestration.ScenarioRequest{
		Description: "Create a multi-agent workflow to generate a comprehensive research report. " +
			"The workflow should: 1) Use a web search agent to gather information from multiple sources, " +
			"2) Use a data analysis agent to process and summarize the collected data, " +
			"3) Use a document generation agent to create a well-formatted report. " +
			"If any step fails, the workflow should handle errors gracefully and provide meaningful feedback.",
		Context: map[string]interface{}{
			"task_type":     "research",
			"collaboration": "multi-agent",
			"output":        "document",
		},
	}

	logger.Info("Starting orchestration for scenario:")
	logger.Infof("  %s", scenario.Description)

	// Perform orchestration
	result, err := orchestrator.Orchestrate(ctx, scenario)
	if err != nil {
		logger.Errorf("Orchestration failed: %v", err)
		printResult(result)
		os.Exit(1)
	}

	logger.Info("✓ Orchestration completed successfully!")
	printResult(result)

	// Save results to files
	saveResults(result)

	logger.Info("=== Demo Completed ===")
}

func printResult(result *orchestration.OrchestrationResult) {
	logger.Info("\n=== Orchestration Result ===")
	logger.Infof("Success: %v", result.Success)
	logger.Infof("Message: %s", result.Message)
	logger.Infof("Retry Count: %d", result.RetryCount)
	logger.Infof("Capabilities Found: %d/%d",
		orchestration.CountFoundCapabilities(result.Capabilities),
		len(result.Capabilities))

	if len(result.Capabilities) > 0 {
		logger.Info("\nDiscovered Capabilities:")
		for i, cap := range result.Capabilities {
			status := "❌ NOT FOUND"
			agent := "N/A"
			if cap.Found {
				status = "✓ FOUND"
				agent = cap.Agent.Name
			}
			logger.Infof("  %d. %s %s (Agent: %s)", i+1, status, cap.Requirement.Name, agent)
		}
	}

	if result.SeataWorkflow != nil {
		logger.Info("\n✓ Seata Saga Workflow Generated:")
		logger.Infof("  Name: %s", result.SeataWorkflow.Name)
		logger.Infof("  Version: %s", result.SeataWorkflow.Version)
		logger.Infof("  Start State: %s", result.SeataWorkflow.StartState)
		logger.Infof("  Total States: %d", len(result.SeataWorkflow.States))

		logger.Info("  States:")
		for name, state := range result.SeataWorkflow.States {
			logger.Infof("    - %s (Type: %s)", name, state.Type)
			if state.ServiceName != "" {
				logger.Infof("      Service: %s.%s", state.ServiceName, state.ServiceMethod)
			}
			if state.CompensateState != "" {
				logger.Infof("      Compensation: %s", state.CompensateState)
			}
		}
	}

	if result.ReactFlow != nil {
		logger.Info("\n✓ React Flow Visualization Generated:")
		logger.Infof("  Nodes: %d", len(result.ReactFlow.Nodes))
		logger.Infof("  Edges: %d", len(result.ReactFlow.Edges))
	}
}

func saveResults(result *orchestration.OrchestrationResult) {
	if result.SeataWorkflow != nil {
		data, err := json.MarshalIndent(result.SeataWorkflow, "", "  ")
		if err != nil {
			logger.Errorf("failed to marshal seata workflow: %v", err)
		} else {
			filename := "output_seata_workflow.json"
			if err := os.WriteFile(filename, data, 0644); err != nil {
				logger.Errorf("failed to write seata workflow: %v", err)
			} else {
				logger.Infof("✓ Seata workflow saved to: %s", filename)
			}
		}
	}

	if result.ReactFlow != nil {
		data, err := json.MarshalIndent(result.ReactFlow, "", "  ")
		if err != nil {
			logger.Errorf("failed to marshal react flow: %v", err)
		} else {
			filename := "output_react_flow.json"
			if err := os.WriteFile(filename, data, 0644); err != nil {
				logger.Errorf("failed to write react flow: %v", err)
			} else {
				logger.Infof("✓ React Flow visualization saved to: %s", filename)
			}
		}
	}

	// Save complete result
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Errorf("failed to marshal complete result: %v", err)
	} else {
		filename := "output_orchestration_result.json"
		if err := os.WriteFile(filename, data, 0644); err != nil {
			logger.Errorf("failed to write complete result: %v", err)
		} else {
			logger.Infof("✓ Complete result saved to: %s", filename)
		}
	}
}
