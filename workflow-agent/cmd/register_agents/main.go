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
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var hubURL = flag.String("hub", "http://localhost:8080", "AgentHub URL")

type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Examples    []string `json:"examples"`
}

type AgentCard struct {
	Name               string       `json:"name"`
	Description        string       `json:"description"`
	URL                string       `json:"url"`
	Version            string       `json:"version"`
	Capabilities       interface{}  `json:"capabilities"`
	DefaultInputModes  []string     `json:"defaultInputModes"`
	DefaultOutputModes []string     `json:"defaultOutputModes"`
	Skills             []AgentSkill `json:"skills"`
}

type RegisterRequest struct {
	AgentCard AgentCard `json:"agent_card"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
}

type RegisterResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	AgentID string `json:"agent_id"`
}

func main() {
	flag.Parse()

	// Use the client-based registration
	RegisterViaClient()
	return

	fmt.Println("=== Registering Test Agents to AgentHub ===")
	fmt.Printf("Hub URL: %s\n\n", *hubURL)

	agents := []RegisterRequest{
		{
			AgentCard: AgentCard{
				Name:               "web-search-agent",
				Description:        "Web search agent that can search and retrieve information from the internet",
				URL:                "http://localhost:9001",
				Version:            "1.0.0",
				Capabilities:       map[string]interface{}{},
				DefaultInputModes:  []string{"text"},
				DefaultOutputModes: []string{"text", "json"},
				Skills: []AgentSkill{
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
			AgentCard: AgentCard{
				Name:               "data-analysis-agent",
				Description:        "Data analysis agent that can process, analyze and summarize data",
				URL:                "http://localhost:9002",
				Version:            "1.0.0",
				Capabilities:       map[string]interface{}{},
				DefaultInputModes:  []string{"text", "json"},
				DefaultOutputModes: []string{"text", "json"},
				Skills: []AgentSkill{
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
			AgentCard: AgentCard{
				Name:               "document-gen-agent",
				Description:        "Document generation agent that creates well-formatted documents and reports",
				URL:                "http://localhost:9003",
				Version:            "1.0.0",
				Capabilities:       map[string]interface{}{},
				DefaultInputModes:  []string{"text", "json"},
				DefaultOutputModes: []string{"document", "pdf", "markdown"},
				Skills: []AgentSkill{
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

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	successCount := 0
	for _, agent := range agents {
		fmt.Printf("Registering agent: %s\n", agent.AgentCard.Name)

		data, err := json.Marshal(agent)
		if err != nil {
			fmt.Printf("  ✗ Failed to marshal request: %v\n\n", err)
			continue
		}

		resp, err := client.Post(
			*hubURL+"/api/v1/agents/register",
			"application/json",
			bytes.NewBuffer(data),
		)
		if err != nil {
			fmt.Printf("  ✗ Failed to send request: %v\n\n", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result RegisterResponse
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("  ✗ Failed to parse response: %v\n", err)
			fmt.Printf("  Response body: %s\n\n", string(body))
			continue
		}

		if result.Success {
			fmt.Printf("  ✓ Successfully registered (ID: %s)\n", result.AgentID)
			fmt.Printf("    Message: %s\n\n", result.Message)
			successCount++
		} else {
			fmt.Printf("  ✗ Registration failed: %s\n\n", result.Message)
		}
	}

	fmt.Println("=== Registration Complete ===")
	fmt.Printf("Successfully registered: %d/%d agents\n\n", successCount, len(agents))

	if successCount > 0 {
		fmt.Println("To verify registered agents:")
		fmt.Printf("  curl %s/api/v1/agents\n\n", *hubURL)
		fmt.Println("To discover agents by capability:")
		fmt.Printf("  curl -X POST %s/api/v1/agents/discover -H 'Content-Type: application/json' -d '{\"query\": \"search\"}'\n", *hubURL)
	}

	if successCount < len(agents) {
		os.Exit(1)
	}
}
