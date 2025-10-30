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
	"fmt"

	"seata-go-ai-a2a/pkg/types"
)

// AgentCardBuilder builder for creating AgentCard with fluent API
type AgentCardBuilder struct {
	card *types.AgentCard
}

// NewAgentCardBuilder creates a new AgentCard builder
func NewAgentCardBuilder() *AgentCardBuilder {
	return &AgentCardBuilder{
		card: &types.AgentCard{
			ProtocolVersion: "2024-01-01",
			Capabilities:    &types.AgentCapabilities{},
			Skills:          []*types.AgentSkill{},
			SecuritySchemes: make(map[string]types.SecurityScheme),
			Security:        []*types.Security{},
		},
	}
}

// WithBasicInfo sets basic information for the agent
func (b *AgentCardBuilder) WithBasicInfo(name, description, version string) *AgentCardBuilder {
	b.card.Name = name
	b.card.Description = description
	b.card.Version = version
	return b
}

// WithURL sets the agent URL
func (b *AgentCardBuilder) WithURL(url string) *AgentCardBuilder {
	b.card.URL = url
	return b
}

// WithProvider sets the provider information
func (b *AgentCardBuilder) WithProvider(org, url string) *AgentCardBuilder {
	b.card.Provider = &types.AgentProvider{
		Organization: org,
		URL:          url,
	}
	return b
}

// WithCapabilities sets agent capabilities
func (b *AgentCardBuilder) WithCapabilities(streaming, pushNotifications bool) *AgentCardBuilder {
	b.card.Capabilities.Streaming = streaming
	b.card.Capabilities.PushNotifications = pushNotifications
	return b
}

// WithDefaultInputModes sets default input modes
func (b *AgentCardBuilder) WithDefaultInputModes(modes []string) *AgentCardBuilder {
	b.card.DefaultInputModes = modes
	return b
}

// WithDefaultOutputModes sets default output modes
func (b *AgentCardBuilder) WithDefaultOutputModes(modes []string) *AgentCardBuilder {
	b.card.DefaultOutputModes = modes
	return b
}

// WithSkill adds a skill to the agent
func (b *AgentCardBuilder) WithSkill(id, name, description string, inputModes, outputModes []string) *AgentCardBuilder {
	skill := &types.AgentSkill{
		ID:          id,
		Name:        name,
		Description: description,
		InputModes:  inputModes,
		OutputModes: outputModes,
	}

	b.card.Skills = append(b.card.Skills, skill)
	return b
}

// WithDetailedSkill adds a skill with detailed configuration
func (b *AgentCardBuilder) WithDetailedSkill(skill *types.AgentSkill) *AgentCardBuilder {
	b.card.Skills = append(b.card.Skills, skill)
	return b
}

// WithDocumentationURL sets documentation URL
func (b *AgentCardBuilder) WithDocumentationURL(url string) *AgentCardBuilder {
	b.card.DocumentationURL = url
	return b
}

// WithIconURL sets icon URL
func (b *AgentCardBuilder) WithIconURL(url string) *AgentCardBuilder {
	b.card.IconURL = url
	return b
}

// WithAPIKeySecurity adds API key security scheme
func (b *AgentCardBuilder) WithAPIKeySecurity(name, description, location, paramName string) *AgentCardBuilder {
	scheme := &types.APIKeySecurityScheme{
		Description: description,
		Location:    location,
		Name:        paramName,
	}

	b.card.SecuritySchemes[name] = scheme
	return b
}

// WithHTTPAuthSecurity adds HTTP authentication security scheme
func (b *AgentCardBuilder) WithHTTPAuthSecurity(name, description, scheme, bearerFormat string) *AgentCardBuilder {
	securityScheme := &types.HTTPAuthSecurityScheme{
		Description:  description,
		Scheme:       scheme,
		BearerFormat: bearerFormat,
	}

	b.card.SecuritySchemes[name] = securityScheme
	return b
}

// WithSecurityRequirement adds a security requirement
func (b *AgentCardBuilder) WithSecurityRequirement(schemeName string, scopes []string) *AgentCardBuilder {
	security := &types.Security{
		Schemes: map[string][]string{
			schemeName: scopes,
		},
	}

	b.card.Security = append(b.card.Security, security)
	return b
}

// WithExtension adds an agent extension
func (b *AgentCardBuilder) WithExtension(uri, description string, required bool, params map[string]any) *AgentCardBuilder {
	extension := &types.AgentExtension{
		URI:         uri,
		Description: description,
		Required:    required,
		Params:      params,
	}

	b.card.Capabilities.Extensions = append(b.card.Capabilities.Extensions, extension)
	return b
}

// WithPreferredTransport sets preferred transport
func (b *AgentCardBuilder) WithPreferredTransport(transport string) *AgentCardBuilder {
	b.card.PreferredTransport = transport
	return b
}

// WithAdditionalInterface adds additional interface
func (b *AgentCardBuilder) WithAdditionalInterface(url, transport string) *AgentCardBuilder {
	interface_ := &types.AgentInterface{
		URL:       url,
		Transport: transport,
	}

	b.card.AdditionalInterfaces = append(b.card.AdditionalInterfaces, interface_)
	return b
}

// WithSignature adds a signature to the agent card
func (b *AgentCardBuilder) WithSignature(protected, signature string, header map[string]any) *AgentCardBuilder {
	sig := &types.AgentCardSignature{
		Protected: protected,
		Signature: signature,
		Header:    header,
	}

	b.card.Signatures = append(b.card.Signatures, sig)
	return b
}

// Build creates the final AgentCard
func (b *AgentCardBuilder) Build() *types.AgentCard {
	return b.card
}

// BuildWithValidation creates the final AgentCard with validation
func (b *AgentCardBuilder) BuildWithValidation() (*types.AgentCard, error) {
	if b.card.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	if b.card.ProtocolVersion == "" {
		return nil, fmt.Errorf("protocol version is required")
	}

	return b.card, nil
}
