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

package orchestration

import (
	"agenthub/pkg/client"
)

// ScenarioRequest represents user's scenario requirement
type ScenarioRequest struct {
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// CapabilityRequirement represents a required capability for the scenario
type CapabilityRequirement struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// DiscoveredCapability represents a capability discovered from hub
type DiscoveredCapability struct {
	Requirement CapabilityRequirement `json:"requirement"`
	Agent       *client.AgentCard     `json:"agent,omitempty"`
	Found       bool                  `json:"found"`
}

// OrchestrationResult represents the final orchestration result
type OrchestrationResult struct {
	Success       bool                   `json:"success"`
	Message       string                 `json:"message"`
	Capabilities  []DiscoveredCapability `json:"capabilities"`
	SeataWorkflow *SeataStateMachine     `json:"seata_workflow,omitempty"`
	ReactFlow     *ReactFlowGraph        `json:"react_flow,omitempty"`
	RetryCount    int                    `json:"retry_count"`
}

// SeataStateMachine represents Seata Saga state machine JSON
type SeataStateMachine struct {
	Name       string               `json:"Name"`
	Comment    string               `json:"Comment"`
	StartState string               `json:"StartState"`
	Version    string               `json:"Version"`
	States     map[string]StateNode `json:"States"`
}

// StateNode represents a state in Seata state machine
type StateNode struct {
	Type            string            `json:"Type"`
	ServiceName     string            `json:"ServiceName,omitempty"`
	ServiceMethod   string            `json:"ServiceMethod,omitempty"`
	CompensateState string            `json:"CompensateState,omitempty"`
	IsForUpdate     bool              `json:"IsForUpdate,omitempty"`
	Input           []string          `json:"Input,omitempty"`
	Output          map[string]string `json:"Output,omitempty"`
	Status          map[string]string `json:"Status,omitempty"`
	Catch           []CatchBlock      `json:"Catch,omitempty"`
	Next            string            `json:"Next,omitempty"`
	Choices         []ChoiceRule      `json:"Choices,omitempty"`
	Default         string            `json:"Default,omitempty"`
	ErrorCode       string            `json:"ErrorCode,omitempty"`
	Message         string            `json:"Message,omitempty"`
}

// CatchBlock represents error catching configuration
type CatchBlock struct {
	Exceptions []string `json:"Exceptions"`
	Next       string   `json:"Next"`
}

// ChoiceRule represents a choice rule in Choice state
type ChoiceRule struct {
	Expression string `json:"Expression"`
	Next       string `json:"Next"`
}

// ReactFlowGraph represents React Flow visualization JSON
type ReactFlowGraph struct {
	Nodes []ReactFlowNode `json:"nodes"`
	Edges []ReactFlowEdge `json:"edges"`
}

// ReactFlowNode represents a node in React Flow
type ReactFlowNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position Position               `json:"position"`
	Data     map[string]interface{} `json:"data"`
	Style    map[string]interface{} `json:"style,omitempty"`
}

// ReactFlowEdge represents an edge in React Flow
type ReactFlowEdge struct {
	ID     string                 `json:"id"`
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Type   string                 `json:"type,omitempty"`
	Label  string                 `json:"label,omitempty"`
	Style  map[string]interface{} `json:"style,omitempty"`
}

// Position represents x,y coordinates
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ReActStep represents a step in ReAct reasoning process
type ReActStep struct {
	Thought     string `json:"thought"`
	Action      string `json:"action"`
	Observation string `json:"observation"`
}

// ReActTrace represents the complete reasoning trace
type ReActTrace struct {
	Steps      []ReActStep `json:"steps"`
	Conclusion string      `json:"conclusion"`
}
