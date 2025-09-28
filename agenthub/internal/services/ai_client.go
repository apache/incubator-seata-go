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

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agenthub/pkg/models"
	"agenthub/pkg/utils"
)

// MockAIClient is a mock implementation of AIClient for testing
type MockAIClient struct {
	logger *utils.Logger
}

// NewMockAIClient creates a new mock AI client
func NewMockAIClient() AIClient {
	return &MockAIClient{
		logger: utils.WithField("component", "mock-ai-client"),
	}
}

// AnalyzeContext analyzes context and generates skill match query using simple rule-based logic
func (c *MockAIClient) AnalyzeContext(ctx context.Context, needDescription string, availableSkills []models.AgentSkill) (*models.SkillMatchQuery, error) {
	c.logger.Info("Mock AI analyzing context: %s", needDescription)

	// Simple rule-based analysis (replace with real AI later)
	query := c.generateQueryFromDescription(needDescription, availableSkills)

	c.logger.Debug("Generated query: %+v", query)
	return query, nil
}

// generateQueryFromDescription generates a skill match query based on keywords
func (c *MockAIClient) generateQueryFromDescription(description string, availableSkills []models.AgentSkill) *models.SkillMatchQuery {
	normalizedDesc := strings.ToLower(description)

	// Define keyword mappings
	keywordMappings := map[string][]string{
		"text":      {"text-processing", "nlp", "language"},
		"文本":        {"text-processing", "nlp", "language"},
		"image":     {"image-processing", "image-analysis", "cv", "vision"},
		"图片":        {"image-processing", "image-analysis", "cv", "vision"},
		"图像":        {"image-processing", "image-analysis", "cv", "vision"},
		"data":      {"data-processing", "data-analysis", "analytics"},
		"数据":        {"data-processing", "data-analysis", "analytics"},
		"translate": {"translation", "language"},
		"翻译":        {"translation", "language"},
		"speech":    {"speech-recognition", "audio", "voice"},
		"语音":        {"speech-recognition", "audio", "voice"},
		"video":     {"video-processing", "multimedia"},
		"视频":        {"video-processing", "multimedia"},
	}

	// Extract main keywords
	var matchedTags []string
	var skillIds []string
	var mainQuery string

	// Find matching keywords and tags
	for keyword, tags := range keywordMappings {
		if strings.Contains(normalizedDesc, keyword) {
			matchedTags = append(matchedTags, tags...)
			if mainQuery == "" {
				mainQuery = keyword
			}
		}
	}

	// If no keywords matched, use the first few words
	if mainQuery == "" {
		words := strings.Fields(normalizedDesc)
		if len(words) > 0 {
			mainQuery = words[0]
			if len(words) > 1 {
				mainQuery += " " + words[1]
			}
		}
	}

	// Find specific skill IDs that match
	for _, skill := range availableSkills {
		skillLower := strings.ToLower(skill.ID + " " + skill.Name + " " + skill.Description)
		if strings.Contains(skillLower, normalizedDesc) || strings.Contains(normalizedDesc, strings.ToLower(skill.ID)) {
			skillIds = append(skillIds, skill.ID)
		}
	}

	// Remove duplicates from tags
	uniqueTags := removeDuplicates(matchedTags)

	return &models.SkillMatchQuery{
		Query:    mainQuery,
		Skills:   skillIds,
		Tags:     uniqueTags,
		Priority: "accuracy",
		Metadata: map[string]string{
			"source":      "mock-ai",
			"description": description,
		},
	}
}

// removeDuplicates removes duplicate strings from slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// AIClientConfig holds configuration for creating AI clients
type AIClientConfig struct {
	Provider  string // "mock", "qwen", "openai"
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
	Timeout   time.Duration
}

// NewAIClient creates an AI client based on configuration
func NewAIClient(config AIClientConfig) AIClient {
	switch config.Provider {
	case "qwen":
		return NewQwenClient(QwenConfig{
			APIKey:    config.APIKey,
			BaseURL:   config.BaseURL,
			Model:     config.Model,
			MaxTokens: config.MaxTokens,
			Timeout:   config.Timeout,
		})
	case "mock":
		fallthrough
	default:
		return NewMockAIClient()
	}
}

// RealAIClient is a placeholder for real AI integration (OpenAI, Anthropic, etc.)
type RealAIClient struct {
	apiKey  string
	baseURL string
	logger  *utils.Logger
}

// NewRealAIClient creates a new real AI client (placeholder)
func NewRealAIClient(apiKey, baseURL string) AIClient {
	return &RealAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		logger:  utils.WithField("component", "real-ai-client"),
	}
}

// AnalyzeContext analyzes context using real AI service (placeholder implementation)
func (c *RealAIClient) AnalyzeContext(ctx context.Context, needDescription string, availableSkills []models.AgentSkill) (*models.SkillMatchQuery, error) {
	c.logger.Info("Real AI analyzing context: %s", needDescription)

	// TODO: Implement real AI integration
	// 1. Construct prompt with needDescription and availableSkills
	// 2. Call AI service (OpenAI/Anthropic/etc.)
	// 3. Parse AI response into SkillMatchQuery

	prompt := c.buildAnalysisPrompt(needDescription, availableSkills)
	c.logger.Debug("Generated prompt: %s", prompt)

	// For now, fallback to mock implementation
	mockClient := NewMockAIClient()
	return mockClient.AnalyzeContext(ctx, needDescription, availableSkills)
}

// buildAnalysisPrompt builds the prompt for AI analysis
func (c *RealAIClient) buildAnalysisPrompt(needDescription string, availableSkills []models.AgentSkill) string {
	skillsJSON, _ := json.MarshalIndent(availableSkills, "", "  ")

	prompt := fmt.Sprintf(`
Based on the following need description and available skills, generate a JSON query to match the most appropriate skills:

Need Description: %s

Available Skills: %s

Please respond with a JSON object in this format:
{
  "query": "main search keyword",
  "skills": ["specific_skill_ids"],  
  "tags": ["relevant_tags"],
  "priority": "accuracy" or "speed",
  "metadata": {"key": "value"}
}

Focus on finding the best matching skills for the given need.
`, needDescription, string(skillsJSON))

	return prompt
}
