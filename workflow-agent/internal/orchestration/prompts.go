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

import "fmt"

// SystemPrompt defines the system role and constraints
const SystemPrompt = `You are an expert AI agent workflow orchestration assistant.

Your task is to:
1. Analyze user's scenario requirements
2. Identify required agent capabilities to accomplish the task
3. Orchestrate multi-agent collaboration workflows using available agents from the hub

The workflow will be executed using Seata Saga state machine as the execution engine, but your focus is on intelligent agent collaboration and task decomposition.

You MUST respond in valid JSON format only. No additional text outside JSON is allowed.`

// CapabilityAnalysisPrompt generates prompt for analyzing required capabilities
func CapabilityAnalysisPrompt(scenario string, context map[string]interface{}) string {
	contextStr := ""
	if len(context) > 0 {
		contextStr = fmt.Sprintf("\n\nAdditional Context:\n%v", context)
	}

	return fmt.Sprintf(`Analyze the following user scenario and identify the required agent capabilities.

Scenario:
%s%s

Please identify and list ALL agent capabilities needed to accomplish this task. For each capability, provide:
1. A clear, concise name (2-4 words) describing what the agent should do
2. A detailed description of the agent's responsibility
3. Whether it's a core capability (required=true) or optional (required=false)

Core capabilities are essential agents WITHOUT which the task cannot be completed.
Optional capabilities are helpful agents that enhance the workflow but are not strictly necessary.

Respond in JSON format:
{
  "analysis": "Your analysis of the scenario",
  "capabilities": [
    {
      "name": "Capability Name",
      "description": "Detailed description of what this capability does",
      "required": true/false
    }
  ],
  "minimum_required_count": <number of core capabilities that MUST be satisfied>
}`, scenario, contextStr)
}

// MinimumRequirementCheckPrompt generates prompt for checking if minimum requirements are met
func MinimumRequirementCheckPrompt(scenario string, discoveredCapabilities []DiscoveredCapability, requiredCount int) string {
	capList := ""
	foundCount := 0
	for _, cap := range discoveredCapabilities {
		status := "NOT FOUND"
		if cap.Found {
			status = fmt.Sprintf("FOUND - Agent: %s", cap.Agent.Name)
			foundCount++
		}
		capList += fmt.Sprintf("\n- %s (%s): %s", cap.Requirement.Name, status, cap.Requirement.Description)
	}

	return fmt.Sprintf(`Evaluate whether the discovered capabilities meet the minimum requirements for the scenario.

Scenario:
%s

Required Capabilities Summary:
%s

Statistics:
- Total capabilities found: %d
- Minimum required: %d

Analyze whether the FOUND capabilities are sufficient to create a functional workflow.
Consider:
1. Can the core business logic be implemented with available capabilities?
2. Are there any critical missing capabilities that block the workflow?
3. Can missing optional capabilities be worked around?

Respond in JSON format:
{
  "sufficient": true/false,
  "reason": "Detailed explanation of why requirements are/aren't met",
  "missing_critical": ["list of critical missing capabilities"],
  "suggestions": "Suggestions for next steps if insufficient"
}`, scenario, capList, foundCount, requiredCount)
}

// WorkflowOrchestrationPrompt generates prompt for workflow orchestration
func WorkflowOrchestrationPrompt(scenario string, capabilities []DiscoveredCapability) string {
	capDetails := ""
	for _, cap := range capabilities {
		if cap.Found {
			capDetails += fmt.Sprintf(`
Capability: %s
Agent: %s
URL: %s
Description: %s
Skills: %v
`, cap.Requirement.Name, cap.Agent.Name, cap.Agent.URL, cap.Agent.Description, cap.Agent.Skills)
		}
	}

	return fmt.Sprintf(`You are an AI agent workflow orchestration expert. Create a multi-agent collaboration workflow for the following scenario.

Scenario:
%s

Available Agents:
%s

Create a complete workflow (using Seata Saga state machine format) that:
1. Orchestrates the available agents to accomplish the task
2. Includes proper error handling and compensation logic for resilience
3. Defines clear agent invocation sequences and data flow
4. Ensures fault tolerance with catch blocks and compensation triggers

IMPORTANT Guidelines:
- ServiceName should be the agent's service name
- ServiceMethod should be a reasonable method name based on the capability
- All ServiceTask states MUST have CompensateState defined
- Use standard parameters: Input: ["$.[businessKey]", "$.[amount]", "$.[params]"]
- Include proper Status mappings: {"#root == true": "SU", "#root == false": "FA", "$Exception{java.lang.Throwable}": "UN"}
- Add Catch blocks: [{"Exceptions": ["java.lang.Throwable"], "Next": "CompensationTrigger"}]
- Create compensation states for each service task
- Include CompensationTrigger and Fail terminal states
- End with Succeed terminal state

Respond with ONLY a valid JSON object (no markdown, no code blocks):
{
  "workflow_name": "Descriptive name for the workflow",
  "workflow_description": "Brief description",
  "seata_json": {
    "Name": "WorkflowName",
    "Comment": "Description",
    "StartState": "FirstStateName",
    "Version": "0.0.1",
    "States": {
      "StateName": {
        "Type": "ServiceTask",
        "ServiceName": "agent-service-name",
        "ServiceMethod": "methodName",
        "CompensateState": "CompensateStateName",
        "IsForUpdate": true,
        "Input": ["$.[businessKey]", "$.[amount]", "$.[params]"],
        "Output": {"result": "$.#root"},
        "Status": {
          "#root == true": "SU",
          "#root == false": "FA",
          "$Exception{java.lang.Throwable}": "UN"
        },
        "Catch": [
          {
            "Exceptions": ["java.lang.Throwable"],
            "Next": "CompensationTrigger"
          }
        ],
        "Next": "NextState"
      },
      "CompensateStateName": {
        "Type": "ServiceTask",
        "ServiceName": "agent-service-name",
        "ServiceMethod": "compensateMethodName",
        "Input": ["$.[businessKey]", "$.[params]"],
        "Output": {"compensateResult": "$.#root"},
        "Status": {
          "#root == true": "SU",
          "$Exception{java.lang.Throwable}": "FA"
        }
      },
      "CompensationTrigger": {
        "Type": "CompensationTrigger",
        "Next": "Fail"
      },
      "Succeed": {
        "Type": "Succeed"
      },
      "Fail": {
        "Type": "Fail",
        "ErrorCode": "WORKFLOW_FAILED",
        "Message": "Workflow execution failed"
      }
    }
  },
  "react_flow": {
    "nodes": [
      {
        "id": "start",
        "type": "input",
        "position": {"x": 0, "y": 100},
        "data": {"label": "Start"},
        "style": {"background": "#52C41A", "color": "white", "border": "1px solid #389e0d"}
      },
      {
        "id": "state1",
        "type": "default",
        "position": {"x": 200, "y": 100},
        "data": {"label": "State Name\n(Service.Method)"},
        "style": {"background": "#5B8FF9", "color": "white", "border": "1px solid #1d39c4"}
      },
      {
        "id": "compensate1",
        "type": "default",
        "position": {"x": 200, "y": 250},
        "data": {"label": "Compensate State\n(Service.CompensateMethod)"},
        "style": {"background": "#FF6B3B", "color": "white", "border": "1px solid #d4380d"}
      },
      {
        "id": "succeed",
        "type": "output",
        "position": {"x": 800, "y": 100},
        "data": {"label": "Succeed"},
        "style": {"background": "#52C41A", "color": "white", "border": "1px solid #389e0d"}
      },
      {
        "id": "compensationTrigger",
        "type": "default",
        "position": {"x": 600, "y": 250},
        "data": {"label": "Compensation Trigger"},
        "style": {"background": "#FFC53D", "color": "black", "border": "1px solid #d48806"}
      },
      {
        "id": "fail",
        "type": "output",
        "position": {"x": 800, "y": 250},
        "data": {"label": "Fail"},
        "style": {"background": "#FF4D4F", "color": "white", "border": "1px solid #cf1322"}
      }
    ],
    "edges": [
      {
        "id": "e-start-state1",
        "source": "start",
        "target": "state1",
        "type": "default",
        "label": "execute",
        "style": {"stroke": "#5B8FF9", "strokeWidth": 2}
      },
      {
        "id": "e-state1-succeed",
        "source": "state1",
        "target": "succeed",
        "type": "default",
        "label": "success",
        "style": {"stroke": "#5B8FF9", "strokeWidth": 2}
      },
      {
        "id": "e-state1-trigger",
        "source": "state1",
        "target": "compensationTrigger",
        "type": "straight",
        "label": "error",
        "style": {"stroke": "#FF4D4F", "strokeWidth": 2}
      },
      {
        "id": "e-trigger-compensate1",
        "source": "compensationTrigger",
        "target": "compensate1",
        "type": "step",
        "label": "compensate",
        "style": {"stroke": "#FF6B3B", "strokeWidth": 2}
      },
      {
        "id": "e-trigger-fail",
        "source": "compensationTrigger",
        "target": "fail",
        "type": "default",
        "label": "after compensation",
        "style": {"stroke": "#FF4D4F", "strokeWidth": 2}
      }
    ]
  }
}`, scenario, capDetails)
}

// ReActThinkingPrompt generates prompt for ReAct reasoning
func ReActThinkingPrompt(scenario string, currentStep int, previousSteps []ReActStep) string {
	history := ""
	for i, step := range previousSteps {
		history += fmt.Sprintf("\nStep %d:\n  Thought: %s\n  Action: %s\n  Observation: %s\n",
			i+1, step.Thought, step.Action, step.Observation)
	}

	return fmt.Sprintf(`You are using ReAct (Reasoning + Acting) pattern to orchestrate a workflow.

Scenario: %s

Previous Steps:%s

Current Step: %d

Think about the next action you should take. Choose from:
1. "analyze_capabilities" - Analyze what capabilities are needed
2. "discover_agents" - Discover agents that provide required capabilities
3. "verify_requirements" - Check if discovered capabilities meet minimum requirements
4. "retry_discovery" - Refine capability descriptions and discover again
5. "orchestrate_workflow" - Create the final workflow
6. "complete" - Task completed successfully
7. "fail" - Task cannot be completed

Respond in JSON format:
{
  "thought": "Your reasoning about current situation and what to do next",
  "action": "One of the actions above",
  "rationale": "Why you chose this action"
}`, scenario, history, currentStep)
}

// RefinementPrompt generates prompt for refining capability descriptions when discovery fails
func RefinementPrompt(capability CapabilityRequirement, previousDescription string) string {
	return fmt.Sprintf(`The capability "%s" was not found with the current description.

Current Description: %s

Please refine the description to be more specific and searchable. Consider:
1. Using more common technical terms
2. Breaking down complex capabilities into simpler ones
3. Being more specific about the action/operation
4. Using industry-standard terminology

Respond in JSON format:
{
  "refined_name": "Improved capability name",
  "refined_description": "More specific and searchable description",
  "alternative_names": ["alternative1", "alternative2"],
  "reasoning": "Why you made these changes"
}`, capability.Name, previousDescription)
}
