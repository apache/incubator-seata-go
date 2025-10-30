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
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"seata-go-ai-a2a/pkg/transport/grpc"
	"seata-go-ai-a2a/pkg/transport/rest"
	"seata-go-ai-a2a/pkg/types"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: client [grpc|rest|interactive]")
		fmt.Println("  grpc       - Test gRPC client")
		fmt.Println("  rest       - Test REST client")
		fmt.Println("  interactive - Interactive chat mode (REST)")
		os.Exit(1)
	}

	mode := os.Args[1]

	switch mode {
	case "grpc":
		testGRPCClient()
	case "rest":
		testRESTClient()
	case "interactive":
		interactiveChat()
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
		os.Exit(1)
	}
}

// testGRPCClient demonstrates gRPC client usage
func testGRPCClient() {
	log.Println("🔌 Testing gRPC Client...")

	// Create gRPC client
	client, err := grpc.NewClient(&grpc.ClientConfig{
		Address: "localhost:9090",
	})
	if err != nil {
		log.Fatalf("Failed to create gRPC client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test 1: Get Agent Card
	log.Println("\n📋 Getting Agent Card...")
	card, err := client.GetAgentCard(ctx)
	if err != nil {
		log.Fatalf("Failed to get agent card: %v", err)
	}
	fmt.Printf("Agent: %s v%s\n", card.Name, card.Version)
	fmt.Printf("Description: %s\n", card.Description)
	fmt.Printf("Skills: %d available\n", len(card.Skills))

	// Test 2: Send Messages
	messages := []string{
		"Hello there!",
		"What time is it?",
		"Can you show me some code?",
		"Tell me a joke",
		"Thank you!",
	}

	for i, content := range messages {
		log.Printf("\n💬 Test %d: Sending message...", i+1)

		req := &types.SendMessageRequest{
			Request: &types.Message{
				Role: types.RoleUser,
				Parts: []types.Part{
					&types.TextPart{Text: content},
				},
			},
			Configuration: &types.SendMessageConfiguration{
				Blocking: true,
			},
		}

		task, err := client.SendMessage(ctx, req)
		if err != nil {
			log.Printf("❌ Failed to send message: %v", err)
			continue
		}

		fmt.Printf("📨 User: %s\n", content)
		fmt.Printf("🤖 Assistant: %s\n", getLastAssistantMessage(task))
		if len(task.Artifacts) > 0 {
			fmt.Printf("📎 Artifacts: %d attached\n", len(task.Artifacts))
			for _, artifact := range task.Artifacts {
				fmt.Printf("   - %s\n", artifact.Name)
			}
		}
	}

	log.Println("\n✅ gRPC client test completed!")
}

// testRESTClient demonstrates REST client usage
func testRESTClient() {
	log.Println("🌐 Testing REST Client...")

	// Create REST client
	client := rest.NewClient(&rest.ClientConfig{
		BaseURL: "http://localhost:8080",
	})

	ctx := context.Background()

	// Test 1: Get Agent Card
	log.Println("\n📋 Getting Agent Card...")
	card, err := client.GetAgentCard(ctx)
	if err != nil {
		log.Fatalf("Failed to get agent card: %v", err)
	}
	fmt.Printf("Agent: %s v%s\n", card.Name, card.Version)
	fmt.Printf("Description: %s\n", card.Description)

	// Test 2: Send Message
	log.Println("\n💬 Sending test message...")

	req := &types.SendMessageRequest{
		Request: &types.Message{
			Role: types.RoleUser,
			Parts: []types.Part{
				&types.TextPart{Text: "Hello from REST client! What can you do?"},
			},
		},
		Configuration: &types.SendMessageConfiguration{
			Blocking: true,
		},
	}

	response, err := client.SendMessage(ctx, req)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Printf("📨 User: %s\n", getTextFromMessage(req.Request))
	fmt.Printf("🤖 Assistant: %s\n", getLastAssistantMessage(response.Task))

	log.Println("\n✅ REST client test completed!")
}

// interactiveChat provides an interactive chat experience
func interactiveChat() {
	log.Println("💬 Starting Interactive Chat (REST)...")
	log.Println("Type 'quit' or 'exit' to stop")
	log.Println("---")

	// Create REST client
	client := rest.NewClient(&rest.ClientConfig{
		BaseURL: "http://localhost:8080",
	})
	ctx := context.Background()

	// Get agent info
	card, err := client.GetAgentCard(ctx)
	if err != nil {
		log.Fatalf("Failed to get agent card: %v", err)
	}
	fmt.Printf("🤖 Connected to: %s\n", card.Name)
	fmt.Printf("💡 %s\n\n", card.Description)

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "quit" || input == "exit" {
			break
		}

		// Send message
		req := &types.SendMessageRequest{
			Request: &types.Message{
				Role: types.RoleUser,
				Parts: []types.Part{
					&types.TextPart{Text: input},
				},
			},
			Configuration: &types.SendMessageConfiguration{
				Blocking: true,
			},
		}

		response, err := client.SendMessage(ctx, req)
		if err != nil {
			fmt.Printf("❌ Error: %v\n\n", err)
			continue
		}

		fmt.Printf("Assistant: %s\n", getLastAssistantMessage(response.Task))

		if len(response.Task.Artifacts) > 0 {
			fmt.Printf("📎 %d artifact(s) attached:\n", len(response.Task.Artifacts))
			for _, artifact := range response.Task.Artifacts {
				fmt.Printf("   📄 %s:\n", artifact.Name)
				content := getTextFromMessage(&types.Message{Parts: artifact.Parts})
				if len(content) < 200 {
					fmt.Printf("   %s\n", content)
				} else {
					fmt.Printf("   %s...\n", content[:200])
				}
			}
		}
		fmt.Println()
	}

	log.Println("👋 Chat ended. Goodbye!")
}

// getLastAssistantMessage extracts the last assistant message from task history
func getLastAssistantMessage(task *types.Task) string {
	if task == nil || len(task.History) == 0 {
		return "No response"
	}

	// Find the last agent message
	for i := len(task.History) - 1; i >= 0; i-- {
		if task.History[i].Role == types.RoleAgent {
			return getTextFromMessage(task.History[i])
		}
	}

	return "No assistant response found"
}

// getTextFromMessage extracts text content from message parts
func getTextFromMessage(msg *types.Message) string {
	if msg == nil || len(msg.Parts) == 0 {
		return ""
	}

	for _, part := range msg.Parts {
		if textPart, ok := part.(*types.TextPart); ok {
			return textPart.Text
		}
	}
	return ""
}
