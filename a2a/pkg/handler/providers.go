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

package handler

import (
	"context"

	"seata-go-ai-a2a/pkg/types"
)

// SimpleAgentCardProvider simple static card provider
type SimpleAgentCardProvider struct {
	card *types.AgentCard
}

// NewSimpleAgentCardProvider creates a new simple agent card provider with static card
func NewSimpleAgentCardProvider(card *types.AgentCard) *SimpleAgentCardProvider {
	return &SimpleAgentCardProvider{card: card}
}

// GetAgentCard get agent card
func (p *SimpleAgentCardProvider) GetAgentCard(ctx context.Context) (*types.AgentCard, error) {
	return p.card, nil
}

// GetAgentCardForUser get user-specific agent card (returns same card by default, can be overridden)
func (p *SimpleAgentCardProvider) GetAgentCardForUser(ctx context.Context, userInfo *UserInfo) (*types.AgentCard, error) {
	return p.card, nil // Default returns same card, can be customized for different users
}

// DynamicAgentCardProvider dynamic agent card provider that supports runtime customization
type DynamicAgentCardProvider struct {
	baseCard      *types.AgentCard
	cardGenerator func(ctx context.Context, userInfo *UserInfo) (*types.AgentCard, error)
	userCardCache map[string]*types.AgentCard
}

// NewDynamicAgentCardProvider creates a new dynamic agent card provider
func NewDynamicAgentCardProvider(baseCard *types.AgentCard, generator func(ctx context.Context, userInfo *UserInfo) (*types.AgentCard, error)) *DynamicAgentCardProvider {
	return &DynamicAgentCardProvider{
		baseCard:      baseCard,
		cardGenerator: generator,
		userCardCache: make(map[string]*types.AgentCard),
	}
}

// GetAgentCard get base agent card
func (p *DynamicAgentCardProvider) GetAgentCard(ctx context.Context) (*types.AgentCard, error) {
	return p.baseCard, nil
}

// GetAgentCardForUser get user-specific agent card with customization
func (p *DynamicAgentCardProvider) GetAgentCardForUser(ctx context.Context, userInfo *UserInfo) (*types.AgentCard, error) {
	if userInfo == nil {
		return p.baseCard, nil
	}

	// Check cache first
	if cachedCard, exists := p.userCardCache[userInfo.UserID]; exists {
		return cachedCard, nil
	}

	// Generate customized card
	if p.cardGenerator != nil {
		customCard, err := p.cardGenerator(ctx, userInfo)
		if err != nil {
			return nil, err
		}

		// Cache the result
		p.userCardCache[userInfo.UserID] = customCard
		return customCard, nil
	}

	return p.baseCard, nil
}
