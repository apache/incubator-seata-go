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

package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"seata-go-ai-workflow-agent/pkg/agent/types"
	"strings"
)

// ResponseParser implements advanced response parsing capabilities
type ResponseParser struct {
	config   *ParserConfig
	patterns map[string]*regexp.Regexp
}

// ParserConfig holds parser configuration
type ParserConfig struct {
	EnableJSON     bool              `json:"enable_json"`
	EnableMarkdown bool              `json:"enable_markdown"`
	EnableXML      bool              `json:"enable_xml"`
	EnableRegex    bool              `json:"enable_regex"`
	CustomPatterns map[string]string `json:"custom_patterns"`
	StrictMode     bool              `json:"strict_mode"`
	FallbackToText bool              `json:"fallback_to_text"`
}

// DefaultParserConfig returns default parser configuration
func DefaultParserConfig() *ParserConfig {
	return &ParserConfig{
		EnableJSON:     true,
		EnableMarkdown: true,
		EnableXML:      false,
		EnableRegex:    true,
		StrictMode:     false,
		FallbackToText: true,
		CustomPatterns: make(map[string]string),
	}
}

// NewResponseParser creates a new response parser
func NewResponseParser(config *ParserConfig) *ResponseParser {
	if config == nil {
		config = DefaultParserConfig()
	}

	parser := &ResponseParser{
		config:   config,
		patterns: make(map[string]*regexp.Regexp),
	}

	parser.initializePatterns()
	return parser
}

// initializePatterns compiles regex patterns for parsing
func (p *ResponseParser) initializePatterns() {
	// Core ReAct patterns
	patterns := map[string]string{
		"thought":      `(?i)(?:thought|thinking):\s*(.+?)(?:\n|$)`,
		"action":       `(?i)action:\s*(\w+)`,
		"action_input": `(?i)action\s+input:\s*(\{.*?\}|\[.*?\]|[^\n]+)`,
		"observation":  `(?i)observation:\s*(.+?)(?:\n|$)`,
		"final":        `(?i)(?:final|answer|result|conclusion):\s*(.+?)(?:\n|$)`,
		"reasoning":    `(?i)(?:reasoning|rationale):\s*(.+?)(?:\n|$)`,

		// JSON patterns
		"json_object": `\{[^}]*\}`,
		"json_array":  `\[[^\]]*\]`,

		// Parameter patterns
		"key_value":    `(\w+):\s*([^\n,]+)`,
		"quoted_value": `"([^"]+)":\s*"([^"]+)"`,

		// Action patterns
		"tool_call":  `(?i)(?:use|call|invoke)\s+(\w+)(?:\s+with\s+(.+))?`,
		"parameters": `(?i)(?:parameters|params|args):\s*(.+?)(?:\n|$)`,

		// Status patterns
		"completed": `(?i)(?:done|completed|finished|ready)`,
		"error":     `(?i)(?:error|failed|failure):\s*(.+?)(?:\n|$)`,

		// Markdown patterns
		"code_block":  "```(?:(\\w+)\\n)?([\\s\\S]*?)```",
		"inline_code": "`([^`]+)`",
		"heading":     `^#{1,6}\s+(.+)$`,
		"list_item":   `^\s*[-*+]\s+(.+)$`,
		"numbered":    `^\s*\d+\.\s+(.+)$`,
	}

	// Add custom patterns
	for name, pattern := range p.config.CustomPatterns {
		patterns[name] = pattern
	}

	// Compile patterns
	for name, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			p.patterns[name] = compiled
		}
	}
}

// Parse parses a response into structured format
func (p *ResponseParser) Parse(response string) (*types.ParsedResponse, error) {
	response = strings.TrimSpace(response)

	if response == "" {
		return nil, fmt.Errorf("empty response")
	}

	var result *types.ParsedResponse
	var err error

	// Try different parsing strategies in order
	parsers := []func(string) (*types.ParsedResponse, error){
		p.parseJSON,
		p.parseReActFormat,
		p.parseMarkdown,
		p.parseStructured,
		p.parsePlainText,
	}

	for _, parser := range parsers {
		if result, err = parser(response); err == nil && result != nil {
			// Validate result
			if p.validateResult(result) {
				return result, nil
			}
		}
	}

	// Fallback to text if enabled
	if p.config.FallbackToText {
		return &types.ParsedResponse{
			Type:    types.ResponseTypeThought,
			Content: response,
		}, nil
	}

	return nil, fmt.Errorf("failed to parse response using any strategy")
}

// parseJSON attempts to parse response as JSON
func (p *ResponseParser) parseJSON(response string) (*types.ParsedResponse, error) {
	if !p.config.EnableJSON {
		return nil, fmt.Errorf("JSON parsing disabled")
	}

	var jsonResponse struct {
		Type       string                 `json:"type,omitempty"`
		Thought    string                 `json:"thought,omitempty"`
		Action     string                 `json:"action,omitempty"`
		Parameters map[string]interface{} `json:"parameters,omitempty"`
		Final      string                 `json:"final,omitempty"`
		Content    string                 `json:"content,omitempty"`
		Reasoning  string                 `json:"reasoning,omitempty"`
		Completed  bool                   `json:"completed,omitempty"`
	}

	if err := json.Unmarshal([]byte(response), &jsonResponse); err != nil {
		return nil, err
	}

	result := &types.ParsedResponse{
		Parameters: make(map[string]interface{}),
	}

	// Determine type and content
	if jsonResponse.Final != "" || jsonResponse.Completed {
		result.Type = types.ResponseTypeFinal
		result.Content = jsonResponse.Final
		result.Final = true
	} else if jsonResponse.Action != "" {
		result.Type = types.ResponseTypeAction
		result.Action = jsonResponse.Action
		result.Parameters = jsonResponse.Parameters
		result.Content = jsonResponse.Thought
	} else {
		result.Type = types.ResponseTypeThought
		result.Content = jsonResponse.Thought
	}

	if jsonResponse.Content != "" {
		result.Content = jsonResponse.Content
	}

	if jsonResponse.Reasoning != "" {
		result.Reasoning = jsonResponse.Reasoning
	}

	return result, nil
}

// parseReActFormat parses standard ReAct format
func (p *ResponseParser) parseReActFormat(response string) (*types.ParsedResponse, error) {
	result := &types.ParsedResponse{
		Type:       types.ResponseTypeThought,
		Parameters: make(map[string]interface{}),
	}

	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for final answer
		if matches := p.patterns["final"].FindStringSubmatch(line); len(matches) > 1 {
			result.Type = types.ResponseTypeFinal
			result.Content = strings.TrimSpace(matches[1])
			result.Final = true
			return result, nil
		}

		// Check for thought
		if matches := p.patterns["thought"].FindStringSubmatch(line); len(matches) > 1 {
			result.Content = strings.TrimSpace(matches[1])
			result.Reasoning = result.Content
		}

		// Check for action
		if matches := p.patterns["action"].FindStringSubmatch(line); len(matches) > 1 {
			result.Type = types.ResponseTypeAction
			result.Action = strings.TrimSpace(matches[1])
		}

		// Check for action input/parameters
		if matches := p.patterns["action_input"].FindStringSubmatch(line); len(matches) > 1 {
			paramStr := strings.TrimSpace(matches[1])
			if params := p.parseParameters(paramStr); len(params) > 0 {
				result.Parameters = params
			}
		}

		// Check for observation
		if matches := p.patterns["observation"].FindStringSubmatch(line); len(matches) > 1 {
			// Observations typically don't change the response type
			observation := strings.TrimSpace(matches[1])
			if result.Content == "" {
				result.Content = observation
			}
		}
	}

	return result, nil
}

// parseMarkdown parses markdown-formatted responses
func (p *ResponseParser) parseMarkdown(response string) (*types.ParsedResponse, error) {
	if !p.config.EnableMarkdown {
		return nil, fmt.Errorf("markdown parsing disabled")
	}

	result := &types.ParsedResponse{
		Type:       types.ResponseTypeThought,
		Content:    response,
		Parameters: make(map[string]interface{}),
	}

	// Extract code blocks
	if matches := p.patterns["code_block"].FindAllStringSubmatch(response, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 2 {
				language := match[1]
				code := match[2]

				// Try to parse code as JSON parameters
				if language == "json" || language == "" {
					if params := p.parseJSONParameters(code); len(params) > 0 {
						result.Parameters = params
						result.Type = types.ResponseTypeAction
					}
				}
			}
		}
	}

	// Extract headings for structure
	headings := p.patterns["heading"].FindAllStringSubmatch(response, -1)
	for _, match := range headings {
		if len(match) > 1 {
			heading := strings.ToLower(match[1])

			if strings.Contains(heading, "final") || strings.Contains(heading, "conclusion") {
				result.Type = types.ResponseTypeFinal
				result.Final = true
			} else if strings.Contains(heading, "action") || strings.Contains(heading, "tool") {
				result.Type = types.ResponseTypeAction
			}
		}
	}

	return result, nil
}

// parseStructured parses structured but non-standard formats
func (p *ResponseParser) parseStructured(response string) (*types.ParsedResponse, error) {
	result := &types.ParsedResponse{
		Type:       types.ResponseTypeThought,
		Content:    response,
		Parameters: make(map[string]interface{}),
	}

	// Look for tool calls
	if matches := p.patterns["tool_call"].FindStringSubmatch(response); len(matches) > 1 {
		result.Type = types.ResponseTypeAction
		result.Action = matches[1]

		if len(matches) > 2 && matches[2] != "" {
			if params := p.parseParameters(matches[2]); len(params) > 0 {
				result.Parameters = params
			}
		}
	}

	// Extract key-value pairs
	keyValueMatches := p.patterns["key_value"].FindAllStringSubmatch(response, -1)
	for _, match := range keyValueMatches {
		if len(match) > 2 {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])

			// Skip common non-parameter keys
			if p.isParameterKey(key) {
				result.Parameters[key] = p.convertValue(value)
			}
		}
	}

	return result, nil
}

// parsePlainText handles plain text responses
func (p *ResponseParser) parsePlainText(response string) (*types.ParsedResponse, error) {
	result := &types.ParsedResponse{
		Type:    types.ResponseTypeThought,
		Content: response,
	}

	// Check for completion indicators
	if p.patterns["completed"].MatchString(response) {
		result.Type = types.ResponseTypeFinal
		result.Final = true
	}

	return result, nil
}

// parseParameters extracts parameters from a string
func (p *ResponseParser) parseParameters(paramStr string) map[string]interface{} {
	params := make(map[string]interface{})

	// Try JSON first
	if jsonParams := p.parseJSONParameters(paramStr); len(jsonParams) > 0 {
		return jsonParams
	}

	// Parse key-value pairs
	matches := p.patterns["key_value"].FindAllStringSubmatch(paramStr, -1)
	for _, match := range matches {
		if len(match) > 2 {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])

			if p.isParameterKey(key) {
				params[key] = p.convertValue(value)
			}
		}
	}

	return params
}

// parseJSONParameters attempts to parse JSON parameters
func (p *ResponseParser) parseJSONParameters(jsonStr string) map[string]interface{} {
	var params map[string]interface{}

	jsonStr = strings.TrimSpace(jsonStr)

	// Try direct JSON parsing
	if err := json.Unmarshal([]byte(jsonStr), &params); err == nil {
		return params
	}

	// Extract JSON objects
	if matches := p.patterns["json_object"].FindAllString(jsonStr, -1); len(matches) > 0 {
		for _, match := range matches {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(match), &obj); err == nil {
				if params == nil {
					params = make(map[string]interface{})
				}
				for k, v := range obj {
					params[k] = v
				}
			}
		}
	}

	return params
}

// isParameterKey checks if a key is likely a parameter
func (p *ResponseParser) isParameterKey(key string) bool {
	nonParamKeys := map[string]bool{
		"thought": true, "action": true, "final": true, "answer": true,
		"observation": true, "thinking": true, "result": true, "reasoning": true,
		"type": true, "content": true, "completed": true,
	}
	return !nonParamKeys[strings.ToLower(key)]
}

// convertValue converts string value to appropriate type
func (p *ResponseParser) convertValue(value string) interface{} {
	value = strings.Trim(value, `"'`)

	// Boolean
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	}

	// Try JSON parsing for complex types
	var jsonValue interface{}
	if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
		return jsonValue
	}

	return value
}

// validateResult validates the parsed result
func (p *ResponseParser) validateResult(result *types.ParsedResponse) bool {
	if result == nil {
		return false
	}

	// In strict mode, require specific fields
	if p.config.StrictMode {
		switch result.Type {
		case types.ResponseTypeAction:
			return result.Action != ""
		case types.ResponseTypeFinal:
			return result.Final && result.Content != ""
		case types.ResponseTypeThought:
			return result.Content != ""
		}
	}

	return true
}

// ExtractCodeBlocks extracts code blocks from response
func (p *ResponseParser) ExtractCodeBlocks(response string) []CodeBlock {
	var blocks []CodeBlock

	matches := p.patterns["code_block"].FindAllStringSubmatch(response, -1)
	for _, match := range matches {
		if len(match) > 2 {
			block := CodeBlock{
				Language: match[1],
				Code:     strings.TrimSpace(match[2]),
			}
			blocks = append(blocks, block)
		}
	}

	return blocks
}

// CodeBlock represents a code block
type CodeBlock struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}
