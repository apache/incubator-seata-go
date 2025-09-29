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

package config

type ProjectConfig struct {
	Name       string `json:"name"`
	ModuleName string `json:"moduleName"`
	Mode       string `json:"mode"` // "default" or "provider"
	OutputDir  string `json:"outputDir"`
}

type AgentCard struct {
	Name                              string                    `json:"name"`
	Description                       string                    `json:"description"`
	URL                               string                    `json:"url"`
	Provider                          *AgentProvider            `json:"provider,omitempty"`
	Version                           string                    `json:"version"`
	DocumentationURL                  *string                   `json:"documentationUrl,omitempty"`
	Capabilities                      AgentCapabilities         `json:"capabilities"`
	SecuritySchemes                   map[string]SecurityScheme `json:"securitySchemes,omitempty"`
	Security                          []map[string][]string     `json:"security,omitempty"`
	DefaultInputModes                 []string                  `json:"defaultInputModes"`
	DefaultOutputModes                []string                  `json:"defaultOutputModes"`
	Skills                            []AgentSkill              `json:"skills"`
	SupportsAuthenticatedExtendedCard *bool                     `json:"supportsAuthenticatedExtendedCard,omitempty"`
}

type AgentProvider struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type AgentCapabilities struct {
	SupportsStreaming bool     `json:"supportsStreaming"`
	SupportedModes    []string `json:"supportedModes"`
}

type SecurityScheme struct {
	Type         string            `json:"type"`
	Scheme       string            `json:"scheme,omitempty"`
	BearerFormat string            `json:"bearerFormat,omitempty"`
	Description  string            `json:"description,omitempty"`
	Flows        map[string]string `json:"flows,omitempty"`
}

type AgentSkill struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type TemplateData struct {
	ProjectName string
	ModuleName  string
	Mode        string
	Timestamp   string
	AgentCard   AgentCard
}
