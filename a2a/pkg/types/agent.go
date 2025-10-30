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

package types

import (
	"fmt"

	pb "seata-go-ai-a2a/pkg/proto/v1"
)

// AgentCard conveys key information about an agent
type AgentCard struct {
	ProtocolVersion                   string                    `json:"protocolVersion"`
	Name                              string                    `json:"name"`
	Description                       string                    `json:"description"`
	URL                               string                    `json:"url"`
	PreferredTransport                string                    `json:"preferredTransport"`
	AdditionalInterfaces              []*AgentInterface         `json:"additionalInterfaces,omitempty"`
	Provider                          *AgentProvider            `json:"provider,omitempty"`
	Version                           string                    `json:"version"`
	DocumentationURL                  string                    `json:"documentationUrl,omitempty"`
	Capabilities                      *AgentCapabilities        `json:"capabilities,omitempty"`
	SecuritySchemes                   map[string]SecurityScheme `json:"securitySchemes,omitempty"`
	Security                          []*Security               `json:"security,omitempty"`
	DefaultInputModes                 []string                  `json:"defaultInputModes,omitempty"`
	DefaultOutputModes                []string                  `json:"defaultOutputModes,omitempty"`
	Skills                            []*AgentSkill             `json:"skills,omitempty"`
	SupportsAuthenticatedExtendedCard bool                      `json:"supportsAuthenticatedExtendedCard,omitempty"`
	Signatures                        []*AgentCardSignature     `json:"signatures,omitempty"`
	IconURL                           string                    `json:"iconUrl,omitempty"`
}

// AgentInterface defines additional transport information for the agent
type AgentInterface struct {
	URL       string `json:"url"`
	Transport string `json:"transport"`
}

// AgentProvider represents information about the service provider of an agent
type AgentProvider struct {
	URL          string `json:"url"`
	Organization string `json:"organization"`
}

// AgentCapabilities defines the A2A feature set supported by the agent
type AgentCapabilities struct {
	Streaming              bool              `json:"streaming"`
	PushNotifications      bool              `json:"pushNotifications"`
	StateTransitionHistory bool              `json:"stateTransitionHistory"`
	Extensions             []*AgentExtension `json:"extensions,omitempty"`
}

// AgentExtension represents a declaration of an extension supported by an Agent
type AgentExtension struct {
	URI         string         `json:"uri"`
	Description string         `json:"description"`
	Required    bool           `json:"required"`
	Params      map[string]any `json:"params,omitempty"`
}

// AgentSkill represents a unit of action/solution that the agent can perform
type AgentSkill struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Tags        []string    `json:"tags,omitempty"`
	Examples    []string    `json:"examples,omitempty"`
	InputModes  []string    `json:"inputModes,omitempty"`
	OutputModes []string    `json:"outputModes,omitempty"`
	Security    []*Security `json:"security,omitempty"`
}

// AgentCardSignature represents a JWS signature of an AgentCard
type AgentCardSignature struct {
	Protected string         `json:"protected"`
	Signature string         `json:"signature"`
	Header    map[string]any `json:"header,omitempty"`
}

// AgentCardToProto converts AgentCard to pb.AgentCard
func AgentCardToProto(card *AgentCard) (*pb.AgentCard, error) {
	if card == nil {
		return nil, nil
	}

	additionalInterfaces := make([]*pb.AgentInterface, len(card.AdditionalInterfaces))
	for i, iface := range card.AdditionalInterfaces {
		additionalInterfaces[i] = AgentInterfaceToProto(iface)
	}

	var provider *pb.AgentProvider
	if card.Provider != nil {
		provider = AgentProviderToProto(card.Provider)
	}

	var capabilities *pb.AgentCapabilities
	var err error
	if card.Capabilities != nil {
		capabilities, err = AgentCapabilitiesToProto(card.Capabilities)
		if err != nil {
			return nil, fmt.Errorf("converting capabilities: %w", err)
		}
	}

	securitySchemes := make(map[string]*pb.SecurityScheme)
	for name, scheme := range card.SecuritySchemes {
		pbScheme, err := SecuritySchemeToProto(scheme)
		if err != nil {
			return nil, fmt.Errorf("converting security scheme %s: %w", name, err)
		}
		securitySchemes[name] = pbScheme
	}

	security := make([]*pb.Security, len(card.Security))
	for i, sec := range card.Security {
		security[i] = SecurityToProto(sec)
	}

	skills := make([]*pb.AgentSkill, len(card.Skills))
	for i, skill := range card.Skills {
		skills[i] = AgentSkillToProto(skill)
	}

	signatures := make([]*pb.AgentCardSignature, len(card.Signatures))
	for i, sig := range card.Signatures {
		var err error
		signatures[i], err = AgentCardSignatureToProto(sig)
		if err != nil {
			return nil, fmt.Errorf("converting signature %d: %w", i, err)
		}
	}

	return &pb.AgentCard{
		ProtocolVersion:                   card.ProtocolVersion,
		Name:                              card.Name,
		Description:                       card.Description,
		Url:                               card.URL,
		PreferredTransport:                card.PreferredTransport,
		AdditionalInterfaces:              additionalInterfaces,
		Provider:                          provider,
		Version:                           card.Version,
		DocumentationUrl:                  card.DocumentationURL,
		Capabilities:                      capabilities,
		SecuritySchemes:                   securitySchemes,
		Security:                          security,
		DefaultInputModes:                 card.DefaultInputModes,
		DefaultOutputModes:                card.DefaultOutputModes,
		Skills:                            skills,
		SupportsAuthenticatedExtendedCard: card.SupportsAuthenticatedExtendedCard,
		Signatures:                        signatures,
		IconUrl:                           card.IconURL,
	}, nil
}

// AgentCardFromProto converts pb.AgentCard to AgentCard
func AgentCardFromProto(card *pb.AgentCard) (*AgentCard, error) {
	if card == nil {
		return nil, nil
	}

	additionalInterfaces := make([]*AgentInterface, len(card.AdditionalInterfaces))
	for i, iface := range card.AdditionalInterfaces {
		additionalInterfaces[i] = AgentInterfaceFromProto(iface)
	}

	var provider *AgentProvider
	if card.Provider != nil {
		provider = AgentProviderFromProto(card.Provider)
	}

	var capabilities *AgentCapabilities
	var err error
	if card.Capabilities != nil {
		capabilities, err = AgentCapabilitiesFromProto(card.Capabilities)
		if err != nil {
			return nil, fmt.Errorf("converting capabilities: %w", err)
		}
	}

	securitySchemes := make(map[string]SecurityScheme)
	for name, scheme := range card.SecuritySchemes {
		typesScheme, err := SecuritySchemeFromProto(scheme)
		if err != nil {
			return nil, fmt.Errorf("converting security scheme %s: %w", name, err)
		}
		securitySchemes[name] = typesScheme
	}

	security := make([]*Security, len(card.Security))
	for i, sec := range card.Security {
		security[i] = SecurityFromProto(sec)
	}

	skills := make([]*AgentSkill, len(card.Skills))
	for i, skill := range card.Skills {
		skills[i] = AgentSkillFromProto(skill)
	}

	signatures := make([]*AgentCardSignature, len(card.Signatures))
	for i, sig := range card.Signatures {
		var err error
		signatures[i], err = AgentCardSignatureFromProto(sig)
		if err != nil {
			return nil, fmt.Errorf("converting signature %d: %w", i, err)
		}
	}

	return &AgentCard{
		ProtocolVersion:                   card.ProtocolVersion,
		Name:                              card.Name,
		Description:                       card.Description,
		URL:                               card.Url,
		PreferredTransport:                card.PreferredTransport,
		AdditionalInterfaces:              additionalInterfaces,
		Provider:                          provider,
		Version:                           card.Version,
		DocumentationURL:                  card.DocumentationUrl,
		Capabilities:                      capabilities,
		SecuritySchemes:                   securitySchemes,
		Security:                          security,
		DefaultInputModes:                 card.DefaultInputModes,
		DefaultOutputModes:                card.DefaultOutputModes,
		Skills:                            skills,
		SupportsAuthenticatedExtendedCard: card.SupportsAuthenticatedExtendedCard,
		Signatures:                        signatures,
		IconURL:                           card.IconUrl,
	}, nil
}

// AgentInterfaceToProto converts AgentInterface to pb.AgentInterface
func AgentInterfaceToProto(iface *AgentInterface) *pb.AgentInterface {
	if iface == nil {
		return nil
	}

	return &pb.AgentInterface{
		Url:       iface.URL,
		Transport: iface.Transport,
	}
}

// AgentInterfaceFromProto converts pb.AgentInterface to AgentInterface
func AgentInterfaceFromProto(iface *pb.AgentInterface) *AgentInterface {
	if iface == nil {
		return nil
	}

	return &AgentInterface{
		URL:       iface.Url,
		Transport: iface.Transport,
	}
}

// AgentProviderToProto converts AgentProvider to pb.AgentProvider
func AgentProviderToProto(provider *AgentProvider) *pb.AgentProvider {
	if provider == nil {
		return nil
	}

	return &pb.AgentProvider{
		Url:          provider.URL,
		Organization: provider.Organization,
	}
}

// AgentProviderFromProto converts pb.AgentProvider to AgentProvider
func AgentProviderFromProto(provider *pb.AgentProvider) *AgentProvider {
	if provider == nil {
		return nil
	}

	return &AgentProvider{
		URL:          provider.Url,
		Organization: provider.Organization,
	}
}

// AgentCapabilitiesToProto converts AgentCapabilities to pb.AgentCapabilities
func AgentCapabilitiesToProto(capabilities *AgentCapabilities) (*pb.AgentCapabilities, error) {
	if capabilities == nil {
		return nil, nil
	}

	extensions := make([]*pb.AgentExtension, len(capabilities.Extensions))
	for i, ext := range capabilities.Extensions {
		var err error
		extensions[i], err = AgentExtensionToProto(ext)
		if err != nil {
			return nil, fmt.Errorf("converting extension %d: %w", i, err)
		}
	}

	return &pb.AgentCapabilities{
		Streaming:              capabilities.Streaming,
		PushNotifications:      capabilities.PushNotifications,
		StateTransitionHistory: capabilities.StateTransitionHistory,
		Extensions:             extensions,
	}, nil
}

// AgentCapabilitiesFromProto converts pb.AgentCapabilities to AgentCapabilities
func AgentCapabilitiesFromProto(capabilities *pb.AgentCapabilities) (*AgentCapabilities, error) {
	if capabilities == nil {
		return nil, nil
	}

	extensions := make([]*AgentExtension, len(capabilities.Extensions))
	for i, ext := range capabilities.Extensions {
		var err error
		extensions[i], err = AgentExtensionFromProto(ext)
		if err != nil {
			return nil, fmt.Errorf("converting extension %d: %w", i, err)
		}
	}

	return &AgentCapabilities{
		Streaming:              capabilities.Streaming,
		PushNotifications:      capabilities.PushNotifications,
		StateTransitionHistory: capabilities.StateTransitionHistory,
		Extensions:             extensions,
	}, nil
}

// AgentExtensionToProto converts AgentExtension to pb.AgentExtension
func AgentExtensionToProto(extension *AgentExtension) (*pb.AgentExtension, error) {
	if extension == nil {
		return nil, nil
	}

	params, err := MapToStruct(extension.Params)
	if err != nil {
		return nil, fmt.Errorf("converting extension params: %w", err)
	}

	return &pb.AgentExtension{
		Uri:         extension.URI,
		Description: extension.Description,
		Required:    extension.Required,
		Params:      params,
	}, nil
}

// AgentExtensionFromProto converts pb.AgentExtension to AgentExtension
func AgentExtensionFromProto(extension *pb.AgentExtension) (*AgentExtension, error) {
	if extension == nil {
		return nil, nil
	}

	params, err := StructToMap(extension.Params)
	if err != nil {
		return nil, fmt.Errorf("converting extension params: %w", err)
	}

	return &AgentExtension{
		URI:         extension.Uri,
		Description: extension.Description,
		Required:    extension.Required,
		Params:      params,
	}, nil
}

// AgentSkillToProto converts AgentSkill to pb.AgentSkill
func AgentSkillToProto(skill *AgentSkill) *pb.AgentSkill {
	if skill == nil {
		return nil
	}

	security := make([]*pb.Security, len(skill.Security))
	for i, sec := range skill.Security {
		security[i] = SecurityToProto(sec)
	}

	return &pb.AgentSkill{
		Id:          skill.ID,
		Name:        skill.Name,
		Description: skill.Description,
		Tags:        skill.Tags,
		Examples:    skill.Examples,
		InputModes:  skill.InputModes,
		OutputModes: skill.OutputModes,
		Security:    security,
	}
}

// AgentSkillFromProto converts pb.AgentSkill to AgentSkill
func AgentSkillFromProto(skill *pb.AgentSkill) *AgentSkill {
	if skill == nil {
		return nil
	}

	security := make([]*Security, len(skill.Security))
	for i, sec := range skill.Security {
		security[i] = SecurityFromProto(sec)
	}

	return &AgentSkill{
		ID:          skill.Id,
		Name:        skill.Name,
		Description: skill.Description,
		Tags:        skill.Tags,
		Examples:    skill.Examples,
		InputModes:  skill.InputModes,
		OutputModes: skill.OutputModes,
		Security:    security,
	}
}

// AgentCardSignatureToProto converts AgentCardSignature to pb.AgentCardSignature
func AgentCardSignatureToProto(signature *AgentCardSignature) (*pb.AgentCardSignature, error) {
	if signature == nil {
		return nil, nil
	}

	header, err := MapToStruct(signature.Header)
	if err != nil {
		return nil, fmt.Errorf("converting signature header: %w", err)
	}

	return &pb.AgentCardSignature{
		Protected: signature.Protected,
		Signature: signature.Signature,
		Header:    header,
	}, nil
}

// AgentCardSignatureFromProto converts pb.AgentCardSignature to AgentCardSignature
func AgentCardSignatureFromProto(signature *pb.AgentCardSignature) (*AgentCardSignature, error) {
	if signature == nil {
		return nil, nil
	}

	header, err := StructToMap(signature.Header)
	if err != nil {
		return nil, fmt.Errorf("converting signature header: %w", err)
	}

	return &AgentCardSignature{
		Protected: signature.Protected,
		Signature: signature.Signature,
		Header:    header,
	}, nil
}
