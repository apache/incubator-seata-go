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

	"grpc-a2a/pkg/handler"
	"grpc-a2a/pkg/types"
)

// ChatAgentCardProvider provides dynamic agent card information
type ChatAgentCardProvider struct {
	baseCard *types.AgentCard
}

// NewChatAgentCardProvider creates a new agent card provider
func NewChatAgentCardProvider() *ChatAgentCardProvider {
	// Create base agent card using the builder
	card := handler.NewAgentCardBuilder().
		WithBasicInfo(
			"Simple Chat Assistant",
			"A friendly A2A chat assistant that can help with various tasks including conversation, time queries, code examples, and jokes.",
			"1.0.0",
		).
		WithURL("http://localhost:8080").
		WithProvider("A2A Examples", "https://github.com/anthropics/grpc-a2a").
		WithCapabilities(true, false). // streaming=true, pushNotifications=false
		WithDefaultInputModes([]string{"text"}).
		WithDefaultOutputModes([]string{"text", "artifact"}).
		WithSkill(
			"general-conversation",
			"General Conversation",
			"Engage in natural conversation and provide helpful responses",
			[]string{"text"},
			[]string{"text"},
		).
		WithSkill(
			"time-info",
			"Time Information",
			"Provide current time and date information",
			[]string{"text"},
			[]string{"text"},
		).
		WithSkill(
			"code-generation",
			"Code Generation",
			"Generate simple code examples and programming snippets",
			[]string{"text"},
			[]string{"text", "artifact"},
		).
		WithSkill(
			"joke-telling",
			"Joke Telling",
			"Tell jokes and provide light entertainment",
			[]string{"text"},
			[]string{"text"},
		).
		WithDocumentationURL("https://github.com/anthropics/grpc-a2a/blob/main/examples/README.md").
		WithIconURL("https://example.com/chat-icon.png").
		Build()

	return &ChatAgentCardProvider{
		baseCard: card,
	}
}

// GetAgentCard returns the standard agent card
func (p *ChatAgentCardProvider) GetAgentCard(ctx context.Context) (*types.AgentCard, error) {
	return p.baseCard, nil
}

// GetAgentCardForUser returns a user-specific agent card (could be customized based on user)
func (p *ChatAgentCardProvider) GetAgentCardForUser(ctx context.Context, userInfo *handler.UserInfo) (*types.AgentCard, error) {
	// For this example, we return the same card for all users
	// In a real implementation, you might customize based on userInfo
	return p.baseCard, nil
}
