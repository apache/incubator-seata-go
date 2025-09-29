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

package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"seata-go-ai-workflow-agent/pkg/agent/types"
	"strings"

	"github.com/firebase/genkit/go/ai"
)

// LLMReasoner implements reasoning using Genkit's AI capabilities
type LLMReasoner struct {
	model       ai.Model
	config      *ReasonerConfig
	promptCache map[string]string
}

// ReasonerConfig holds reasoning configuration
type ReasonerConfig struct {
	Model           string  `json:"model"`
	Temperature     float32 `json:"temperature"`
	MaxTokens       int     `json:"max_tokens"`
	SystemPrompt    string  `json:"system_prompt"`
	ReasoningPrompt string  `json:"reasoning_prompt"`
	ActionPrompt    string  `json:"action_prompt"`
	FinalPrompt     string  `json:"final_prompt"`
}

// DefaultReasonerConfig returns default configuration
func DefaultReasonerConfig() *ReasonerConfig {
	return &ReasonerConfig{
		Model:           "gpt-4",
		Temperature:     0.7,
		MaxTokens:       2048,
		SystemPrompt:    defaultSystemPrompt,
		ReasoningPrompt: defaultReasoningPrompt,
		ActionPrompt:    defaultActionPrompt,
		FinalPrompt:     defaultFinalPrompt,
	}
}

// NewLLMReasoner creates a new LLM-based reasoner
func NewLLMReasoner(model ai.Model, config *ReasonerConfig) *LLMReasoner {
	if config == nil {
		config = DefaultReasonerConfig()
	}

	return &LLMReasoner{
		model:       model,
		config:      config,
		promptCache: make(map[string]string),
	}
}

// Reason performs reasoning based on current context
func (r *LLMReasoner) Reason(ctx context.Context, input string, context map[string]interface{}) (*types.ReasoningResult, error) {
	// Build prompt with context
	prompt, err := r.buildReasoningPrompt(input, context)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Generate response using Genkit AI
	messages := []*ai.Message{
		ai.NewSystemTextMessage(r.config.SystemPrompt),
		ai.NewUserTextMessage(prompt),
	}

	request := &ai.ModelRequest{
		Messages: messages,
		Config: &ai.GenerationCommonConfig{
			Temperature:     float64(r.config.Temperature),
			MaxOutputTokens: r.config.MaxTokens,
		},
	}

	response, err := r.model.Generate(ctx, request, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	if response.Message == nil || len(response.Message.Content) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	responseText := response.Message.Content[0].Text

	// Parse the response into structured format
	parsed, err := r.ParseResponse(responseText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to reasoning result
	result := &types.ReasoningResult{
		Thought:    parsed.Content,
		Action:     parsed.Action,
		Parameters: parsed.Parameters,
		Final:      parsed.Final,
		Confidence: 0.8, // Default confidence
	}

	return result, nil
}

// ParseResponse parses LLM response into structured format
func (r *LLMReasoner) ParseResponse(response string) (*types.ParsedResponse, error) {
	response = strings.TrimSpace(response)

	// Try to parse as JSON first
	if jsonResult := r.tryParseJSON(response); jsonResult != nil {
		return jsonResult, nil
	}

	// Parse using regex patterns
	return r.parseWithPatterns(response)
}

// tryParseJSON attempts to parse response as JSON
func (r *LLMReasoner) tryParseJSON(response string) *types.ParsedResponse {
	var result types.ParsedResponse
	if err := json.Unmarshal([]byte(response), &result); err == nil {
		return &result
	}
	return nil
}

// parseWithPatterns parses response using regex patterns
func (r *LLMReasoner) parseWithPatterns(response string) (*types.ParsedResponse, error) {
	result := &types.ParsedResponse{
		Type:       types.ResponseTypeThought,
		Content:    response,
		Parameters: make(map[string]interface{}),
	}

	// Patterns for different response types
	patterns := map[string]*regexp.Regexp{
		"thought":     regexp.MustCompile(`(?i)(?:thought|thinking):\s*(.+?)(?:\n|$)`),
		"action":      regexp.MustCompile(`(?i)action:\s*(\w+)`),
		"final":       regexp.MustCompile(`(?i)(?:final|answer|result):\s*(.+?)(?:\n|$)`),
		"observation": regexp.MustCompile(`(?i)observation:\s*(.+?)(?:\n|$)`),
	}

	// Check for final answer
	if finalMatch := patterns["final"].FindStringSubmatch(response); finalMatch != nil {
		result.Type = types.ResponseTypeFinal
		result.Content = strings.TrimSpace(finalMatch[1])
		result.Final = true
		return result, nil
	}

	// Check for action
	if actionMatch := patterns["action"].FindStringSubmatch(response); actionMatch != nil {
		result.Type = types.ResponseTypeAction
		result.Action = strings.TrimSpace(actionMatch[1])

		// Try to extract parameters
		params, err := r.extractParameters(response, result.Action)
		if err == nil {
			result.Parameters = params
		}
	}

	// Extract thought/reasoning
	if thoughtMatch := patterns["thought"].FindStringSubmatch(response); thoughtMatch != nil {
		result.Reasoning = strings.TrimSpace(thoughtMatch[1])
	}

	// If no specific patterns match, treat as thought
	if result.Type == types.ResponseTypeThought && result.Reasoning == "" {
		result.Reasoning = response
	}

	return result, nil
}

// extractParameters extracts parameters for an action from response
func (r *LLMReasoner) extractParameters(response, action string) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Look for JSON parameters
	jsonPattern := regexp.MustCompile(`\{[^}]+\}`)
	if match := jsonPattern.FindString(response); match != "" {
		var jsonParams map[string]interface{}
		if err := json.Unmarshal([]byte(match), &jsonParams); err == nil {
			return jsonParams, nil
		}
	}

	// Look for parameter patterns
	paramPattern := regexp.MustCompile(`(\w+):\s*([^\n,]+)`)
	matches := paramPattern.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		if len(match) == 3 {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])

			// Skip common non-parameter words
			if !r.isParameterKey(key) {
				continue
			}

			// Try to convert value to appropriate type
			if convertedValue := r.convertValue(value); convertedValue != nil {
				params[key] = convertedValue
			} else {
				params[key] = value
			}
		}
	}

	return params, nil
}

// isParameterKey checks if a key is likely a parameter
func (r *LLMReasoner) isParameterKey(key string) bool {
	nonParamKeys := map[string]bool{
		"thought": true, "action": true, "final": true, "answer": true,
		"observation": true, "thinking": true, "result": true,
	}
	return !nonParamKeys[strings.ToLower(key)]
}

// convertValue attempts to convert string value to appropriate type
func (r *LLMReasoner) convertValue(value string) interface{} {
	value = strings.Trim(value, `"'`)

	// Try boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try integer
	if matched, _ := regexp.MatchString(`^\d+$`, value); matched {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}

	// Try float
	if matched, _ := regexp.MatchString(`^\d+\.\d+$`, value); matched {
		var floatVal float64
		if _, err := fmt.Sscanf(value, "%f", &floatVal); err == nil {
			return floatVal
		}
	}

	return value
}

// buildReasoningPrompt constructs the reasoning prompt with context
func (r *LLMReasoner) buildReasoningPrompt(input string, context map[string]interface{}) (string, error) {
	var promptBuilder strings.Builder

	// Add reasoning prompt template
	promptBuilder.WriteString(r.config.ReasoningPrompt)
	promptBuilder.WriteString("\n\n")

	// Add user input
	promptBuilder.WriteString(fmt.Sprintf("User Input: %s\n\n", input))

	// Add available tools
	if tools, ok := context["available_tools"].([]string); ok {
		promptBuilder.WriteString("Available Tools:\n")
		for _, tool := range tools {
			promptBuilder.WriteString(fmt.Sprintf("- %s\n", tool))
		}
		promptBuilder.WriteString("\n")
	}

	// Add previous steps
	if steps, ok := context["previous_steps"].([]types.Step); ok && len(steps) > 0 {
		promptBuilder.WriteString("Previous Steps:\n")
		for i, step := range steps {
			promptBuilder.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, step.Type, step.Output))
		}
		promptBuilder.WriteString("\n")
	}

	// Add current step information
	if currentStep, ok := context["current_step"].(int); ok {
		if maxSteps, ok := context["max_steps"].(int); ok {
			promptBuilder.WriteString(fmt.Sprintf("Current Step: %d/%d\n\n", currentStep+1, maxSteps))
		}
	}

	// Add action prompt if not final step
	promptBuilder.WriteString(r.config.ActionPrompt)

	return promptBuilder.String(), nil
}

// Default prompts
const defaultSystemPrompt = `You are a helpful AI agent that uses a Reason-Act-Observe approach to solve problems.

You should think step by step about the problem and decide whether to:
1. Use a tool to get more information or perform an action
2. Provide a final answer if you have enough information

When using tools, always explain your reasoning clearly.

Format your responses as:
Thought: [Your reasoning about what to do next]
Action: [tool_name] (if you want to use a tool)
Parameters: {key: value} (if action requires parameters)
Final: [your final answer] (if you're ready to conclude)

Only use "Final:" when you have a complete answer to the user's question.`

const defaultReasoningPrompt = `Think carefully about the user's request and the current context. Consider what information you have and what you might need to find out.

If you need more information, choose an appropriate tool to help you.
If you have enough information, provide a final answer.`

const defaultActionPrompt = `If you need to use a tool, specify:
- Action: [exact tool name]
- Parameters: [required parameters as JSON]

If you're ready to give a final answer, use:
- Final: [your complete answer]`

const defaultFinalPrompt = `Based on all the information gathered, provide a comprehensive answer to the user's question.`
