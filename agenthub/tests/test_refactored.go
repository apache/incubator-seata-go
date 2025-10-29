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

package tests

import (
	"context"
	"fmt"
	"log"

	"agenthub/pkg/models"
	"agenthub/pkg/services"
	"agenthub/pkg/storage"
	"agenthub/pkg/utils"
)

func main() {
	// Test the refactored components
	log.Println("Testing refactored AgentHub components...")

	// Initialize storage
	memStorage := storage.NewMemoryStorage()
	defer memStorage.Close()

	// Initialize logger
	logger := utils.WithField("component", "test")

	// Initialize agent service
	agentService := services.NewAgentService(services.AgentServiceConfig{
		Storage: memStorage,
		Logger:  logger,
	})

	// Test agent registration
	ctx := context.Background()
	registerReq := &models.RegisterRequest{
		AgentCard: models.AgentCard{
			Name:        "test-agent",
			Description: "A test agent for verification",
			URL:         "http://localhost:8080",
			Version:     "1.0.0",
			Skills: []models.AgentSkill{
				{
					ID:          "skill1",
					Name:        "Test Skill",
					Description: "A test skill",
					Tags:        []string{"test", "demo"},
				},
			},
			DefaultInputModes:  []string{"text"},
			DefaultOutputModes: []string{"text"},
		},
		Host: "localhost",
		Port: 8080,
	}

	// Register agent
	registerResp, err := agentService.RegisterAgent(ctx, registerReq)
	if err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}

	if !registerResp.Success {
		log.Fatalf("Agent registration failed: %s", registerResp.Message)
	}

	log.Printf("✓ Agent registration successful: %s", registerResp.AgentID)

	// Test agent discovery
	discoverReq := &models.DiscoverRequest{
		Query: "test",
	}

	discoverResp, err := agentService.DiscoverAgents(ctx, discoverReq)
	if err != nil {
		log.Fatalf("Failed to discover agents: %v", err)
	}

	if len(discoverResp.Agents) == 0 {
		log.Fatalf("No agents discovered")
	}

	log.Printf("✓ Agent discovery successful: found %d agents", len(discoverResp.Agents))

	// Test agent retrieval
	agent, err := agentService.GetAgent(ctx, "test-agent")
	if err != nil {
		log.Fatalf("Failed to get agent: %v", err)
	}

	if agent.AgentCard.Name != "test-agent" {
		log.Fatalf("Retrieved wrong agent: %s", agent.AgentCard.Name)
	}

	log.Printf("✓ Agent retrieval successful: %s", agent.AgentCard.Name)

	// Test agent status update
	err = agentService.UpdateAgentStatus(ctx, "test-agent", "inactive")
	if err != nil {
		log.Fatalf("Failed to update agent status: %v", err)
	}

	// Verify status update
	updatedAgent, err := agentService.GetAgent(ctx, "test-agent")
	if err != nil {
		log.Fatalf("Failed to get updated agent: %v", err)
	}

	if updatedAgent.Status != "inactive" {
		log.Fatalf("Agent status not updated: %s", updatedAgent.Status)
	}

	log.Printf("✓ Agent status update successful: %s", updatedAgent.Status)

	// Test heartbeat update
	err = agentService.UpdateAgentHeartbeat(ctx, "test-agent")
	if err != nil {
		log.Fatalf("Failed to update heartbeat: %v", err)
	}

	log.Printf("✓ Agent heartbeat update successful")

	// Test list all agents
	allAgents, err := agentService.ListAllAgents(ctx)
	if err != nil {
		log.Fatalf("Failed to list all agents: %v", err)
	}

	if len(allAgents) == 0 {
		log.Fatalf("No agents in list")
	}

	log.Printf("✓ List all agents successful: %d agents", len(allAgents))

	// Test agent removal
	err = agentService.RemoveAgent(ctx, "test-agent")
	if err != nil {
		log.Fatalf("Failed to remove agent: %v", err)
	}

	// Verify removal
	_, err = agentService.GetAgent(ctx, "test-agent")
	if err == nil {
		log.Fatalf("Agent still exists after removal")
	}

	log.Printf("✓ Agent removal successful")

	// Test utils functions
	testUtils()

	log.Println("✓ All tests passed! Refactoring is successful and functional.")
}

func testUtils() {
	// Test string utilities
	if !utils.ContainsIgnoreCase("Test String", "test") {
		log.Fatal("String matching failed")
	}

	// Test environment utilities
	defaultVal := utils.GetString("NONEXISTENT_VAR", "default")
	if defaultVal != "default" {
		log.Fatal("Environment utils failed")
	}

	// Test crypto utilities
	key, err := utils.GenerateSecretKey()
	if err != nil || len(key) == 0 {
		log.Fatal("Crypto utils failed")
	}

	// Test JSON utilities
	testData := map[string]interface{}{
		"name": "test",
		"id":   123,
	}

	jsonBytes, err := utils.MarshalJSON(testData)
	if err != nil {
		log.Fatal("JSON marshal failed")
	}

	var result map[string]interface{}
	err = utils.UnmarshalJSON(jsonBytes, &result)
	if err != nil {
		log.Fatal("JSON unmarshal failed")
	}

	fmt.Println("✓ All utility functions working correctly")
}
