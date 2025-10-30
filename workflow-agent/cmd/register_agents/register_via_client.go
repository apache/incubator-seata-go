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
	"flag"
	"fmt"
	"time"

	"agenthub/pkg/client"

	"seata-go-ai-workflow-agent/pkg/hubclient"
)

var hubURL2 = flag.String("hub2", "http://localhost:8080", "AgentHub URL")

func RegisterViaClient() {
	fmt.Println("=== Registering Test Agents via HubClient ===")
	fmt.Printf("Hub URL: %s\n\n", *hubURL2)

	// Initialize hub client
	hub := hubclient.New(&hubclient.Config{
		BaseURL: *hubURL2,
		Timeout: 10 * time.Second,
	})

	ctx := context.Background()

	// Check hub health
	if err := hub.HealthCheck(ctx); err != nil {
		fmt.Printf("✗ Hub health check failed: %v\n", err)
		fmt.Println("Please ensure AgentHub is running on", *hubURL2)
		return
	}
	fmt.Println("✓ Hub is healthy\n")

	agents := []struct {
		Card client.AgentCard
		Host string
		Port int
	}{
		{
			Card: client.AgentCard{
				Name:               "web-search-agent",
				Description:        "Web search agent that can search and retrieve information from the internet",
				URL:                "http://localhost:9001",
				Version:            "1.0.0",
				Capabilities:       client.AgentCapabilities{},
				DefaultInputModes:  []string{"text"},
				DefaultOutputModes: []string{"text", "json"},
				Skills: []client.AgentSkill{
					{
						ID:          "web-search",
						Name:        "Web Search",
						Description: "Search the web for information on any topic",
						Tags:        []string{"search", "web", "information", "query", "internet"},
						Examples:    []string{"search the web", "find information", "look up", "query online"},
					},
					{
						ID:          "data-collection",
						Name:        "Data Collection",
						Description: "Collect and aggregate data from multiple web sources",
						Tags:        []string{"collect", "aggregate", "gather", "data", "sources"},
						Examples:    []string{"collect data", "gather information", "aggregate sources"},
					},
				},
			},
			Host: "localhost",
			Port: 9001,
		},
		{
			Card: client.AgentCard{
				Name:               "data-analysis-agent",
				Description:        "Data analysis agent that can process, analyze and summarize data",
				URL:                "http://localhost:9002",
				Version:            "1.0.0",
				Capabilities:       client.AgentCapabilities{},
				DefaultInputModes:  []string{"text", "json"},
				DefaultOutputModes: []string{"text", "json"},
				Skills: []client.AgentSkill{
					{
						ID:          "data-analysis",
						Name:        "Data Analysis",
						Description: "Analyze and process structured or unstructured data",
						Tags:        []string{"analysis", "process", "data", "analytics", "insights"},
						Examples:    []string{"analyze data", "process information", "extract insights"},
					},
					{
						ID:          "summarization",
						Name:        "Data Summarization",
						Description: "Summarize large amounts of data into key findings",
						Tags:        []string{"summarize", "summary", "condense", "key points"},
						Examples:    []string{"summarize data", "create summary", "condense information"},
					},
				},
			},
			Host: "localhost",
			Port: 9002,
		},
		{
			Card: client.AgentCard{
				Name:               "document-gen-agent",
				Description:        "Document generation agent that creates well-formatted documents and reports",
				URL:                "http://localhost:9003",
				Version:            "1.0.0",
				Capabilities:       client.AgentCapabilities{},
				DefaultInputModes:  []string{"text", "json"},
				DefaultOutputModes: []string{"document", "pdf", "markdown"},
				Skills: []client.AgentSkill{
					{
						ID:          "document-generation",
						Name:        "Document Generation",
						Description: "Generate professional documents from structured data",
						Tags:        []string{"document", "generate", "create", "format", "report"},
						Examples:    []string{"generate document", "create report", "format content"},
					},
					{
						ID:          "report-formatting",
						Name:        "Report Formatting",
						Description: "Format and structure reports with proper layouts",
						Tags:        []string{"format", "structure", "layout", "professional"},
						Examples:    []string{"format report", "structure document", "professional layout"},
					},
				},
			},
			Host: "localhost",
			Port: 9003,
		},
	}

	successCount := 0
	for _, agent := range agents {
		fmt.Printf("Registering agent: %s\n", agent.Card.Name)

		agentID, err := hub.Register(ctx, &client.RegisterRequest{
			AgentCard: agent.Card,
			Host:      agent.Host,
			Port:      agent.Port,
		})
		if err != nil {
			fmt.Printf("  ✗ Failed to register: %v\n\n", err)
			continue
		}

		fmt.Printf("  ✓ Successfully registered (ID: %s)\n\n", agentID)
		successCount++
	}

	fmt.Println("=== Registration Complete ===")
	fmt.Printf("Successfully registered: %d/%d agents\n\n", successCount, len(agents))

	if successCount > 0 {
		// Test discovery
		fmt.Println("Testing agent discovery...")
		agents, err := hub.Discover(ctx, "search")
		if err != nil {
			fmt.Printf("✗ Discovery test failed: %v\n", err)
		} else {
			fmt.Printf("✓ Found %d agents for 'search'\n", len(agents))
			for _, a := range agents {
				fmt.Printf("  - %s\n", a.Name)
			}
		}
	}
}
