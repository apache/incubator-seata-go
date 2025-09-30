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
	"fmt"
	"log"
	"seata-go-ai-workflow-agent/pkg/agent"
	"seata-go-ai-workflow-agent/pkg/agent/types"
	"seata-go-ai-workflow-agent/pkg/session"
	sessionConfig "seata-go-ai-workflow-agent/pkg/session/config"

	"github.com/firebase/genkit/go/ai"
)

func main() {
	// Initialize session management
	sessionCfg := sessionConfig.DefaultConfig()
	sessionCfg.Store.Type = sessionConfig.StoreTypeMemory

	if err := session.Initialize(*sessionCfg); err != nil {
		log.Fatalf("Failed to initialize session manager: %v", err)
	}

	// Note: You would need to configure an actual AI model here
	// This is just an example structure
	var model ai.Model // This would be your actual model implementation

	// Create a simple agent
	reactAgent, err := agent.NewSimpleAgent(model)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Create a request
	request := types.NewAgentRequest("session_123", "user_456", "What is 2 + 2?")
	request.MaxSteps = 5

	// Process the request
	ctx := context.Background()
	response, err := reactAgent.Process(ctx, request)
	if err != nil {
		log.Fatalf("Failed to process request: %v", err)
	}

	// Print the response
	fmt.Printf("Agent Response: %s\n", response.Response)
	fmt.Printf("Steps taken: %d\n", len(response.Steps))
	fmt.Printf("Status: %s\n", response.Status)

	// List available tools
	tools := reactAgent.ListTools()
	fmt.Printf("Available tools: %d\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	// Clean up
	reactAgent.Close()
	session.Close()
}
