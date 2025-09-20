package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"agenthub/pkg/client"
)

func main() {
	// Create client
	c := client.NewClient(client.ClientConfig{
		BaseURL: "http://localhost:8080",
		Timeout: 30 * time.Second,
	})

	ctx := context.Background()

	// Health check
	fmt.Println("=== Health Check ===")
	health, err := c.HealthCheck(ctx)
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("Health status: %s\n\n", health.Status)

	// Register an agent
	fmt.Println("=== Registering Agent ===")
	registerReq := &client.RegisterRequest{
		AgentCard: client.AgentCard{
			Name:        "example-agent",
			Description: "Example agent for demonstration",
			URL:         "http://localhost:3001",
			Version:     "1.0.0",
			Capabilities: client.AgentCapabilities{
				Streaming:         true,
				PushNotifications: false,
			},
			DefaultInputModes:  []string{"text"},
			DefaultOutputModes: []string{"text", "json"},
			Skills: []client.AgentSkill{
				{
					ID:          "text-processing",
					Name:        "文本处理",
					Description: "处理和分析各种文本内容",
					Tags:        []string{"nlp", "text", "processing"},
					Examples:    []string{"情感分析", "关键词提取", "文本摘要"},
				},
				{
					ID:          "data-analysis",
					Name:        "数据分析",
					Description: "分析各种格式的数据",
					Tags:        []string{"data", "analysis", "statistics"},
					Examples:    []string{"数据统计", "趋势分析", "报表生成"},
				},
			},
		},
		Host: "localhost",
		Port: 3001,
	}

	registerResp, err := c.RegisterAgent(ctx, registerReq)
	if err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}
	fmt.Printf("Agent registered: %s (ID: %s)\n\n", registerResp.Message, registerResp.AgentID)

	// List all agents
	fmt.Println("=== Listing Agents ===")
	agents, err := c.ListAgents(ctx)
	if err != nil {
		log.Fatalf("Failed to list agents: %v", err)
	}
	fmt.Printf("Found %d agents:\n", len(agents))
	for _, agent := range agents {
		fmt.Printf("  - %s (%s): %s\n", agent.ID, agent.Status, agent.AgentCard.Description)
	}
	fmt.Println()

	// Discover agents
	fmt.Println("=== Discovering Agents ===")
	discoverReq := &client.DiscoverRequest{
		Query: "文本处理",
	}
	discoverResp, err := c.DiscoverAgents(ctx, discoverReq)
	if err != nil {
		log.Fatalf("Failed to discover agents: %v", err)
	}
	fmt.Printf("Found %d matching agents:\n", len(discoverResp.Agents))
	for _, agent := range discoverResp.Agents {
		fmt.Printf("  - %s: %s\n", agent.Name, agent.Description)
	}
	fmt.Println()

	// Dynamic context analysis
	fmt.Println("=== Context Analysis ===")
	contextReq := &client.ContextAnalysisRequest{
		NeedDescription: "我需要分析用户评论的情感倾向，并提取其中的关键词",
		UserContext:     "电商平台用户反馈分析",
	}
	contextResp, err := c.AnalyzeContext(ctx, contextReq)
	if err != nil {
		log.Printf("Context analysis failed: %v", err)
	} else {
		fmt.Printf("Analysis successful: %s\n", contextResp.Message)
		if len(contextResp.MatchedSkills) > 0 {
			fmt.Printf("Matched %d skills:\n", len(contextResp.MatchedSkills))
			for _, skill := range contextResp.MatchedSkills {
				fmt.Printf("  - %s: %s\n", skill.Name, skill.Description)
			}
		}
		if contextResp.RouteResult != nil {
			fmt.Printf("Route to: %s (skill: %s)\n", contextResp.RouteResult.AgentURL, contextResp.RouteResult.SkillName)
		}
	}
	fmt.Println()

	// Send heartbeat
	fmt.Println("=== Sending Heartbeat ===")
	err = c.SendHeartbeat(ctx, "example-agent")
	if err != nil {
		log.Printf("Heartbeat failed: %v", err)
	} else {
		fmt.Println("Heartbeat sent successfully")
	}
	fmt.Println()

	// Get specific agent
	fmt.Println("=== Getting Agent Details ===")
	agent, err := c.GetAgent(ctx, "example-agent")
	if err != nil {
		log.Printf("Failed to get agent: %v", err)
	} else {
		fmt.Printf("Agent: %s\n", agent.ID)
		fmt.Printf("Status: %s\n", agent.Status)
		fmt.Printf("Skills: %d\n", len(agent.AgentCard.Skills))
		fmt.Printf("Last seen: %s\n", agent.LastSeen.Format(time.RFC3339))
	}
	fmt.Println()

	// Update agent status
	fmt.Println("=== Updating Agent Status ===")
	err = c.UpdateAgentStatus(ctx, "example-agent", "inactive")
	if err != nil {
		log.Printf("Failed to update status: %v", err)
	} else {
		fmt.Println("Agent status updated to inactive")
	}

	// Clean up - remove the agent
	fmt.Println("=== Removing Agent ===")
	err = c.RemoveAgent(ctx, "example-agent")
	if err != nil {
		log.Printf("Failed to remove agent: %v", err)
	} else {
		fmt.Println("Agent removed successfully")
	}

	fmt.Println("Example completed!")
}