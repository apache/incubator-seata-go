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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agenthub/pkg/models"
	"agenthub/pkg/utils"
)

// QwenClient implements AIClient for Alibaba Cloud Qwen-3
type QwenClient struct {
	apiKey     string
	baseURL    string
	model      string
	maxTokens  int
	timeout    time.Duration
	httpClient *http.Client
	logger     *utils.Logger
}

// QwenConfig holds configuration for Qwen client
type QwenConfig struct {
	APIKey    string
	BaseURL   string // https://dashscope.aliyuncs.com
	Model     string // qwen-plus, qwen-turbo, qwen-max
	MaxTokens int
	Timeout   time.Duration
}

// QwenRequest represents the request structure for Qwen API
type QwenRequest struct {
	Model      string         `json:"model"`
	Input      QwenInput      `json:"input"`
	Parameters QwenParameters `json:"parameters"`
}

type QwenInput struct {
	Messages []QwenMessage `json:"messages"`
}

type QwenMessage struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

type QwenParameters struct {
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
}

// QwenResponse represents the response structure from Qwen API
type QwenResponse struct {
	Output    QwenOutput `json:"output"`
	Usage     QwenUsage  `json:"usage"`
	RequestID string     `json:"request_id"`
}

type QwenOutput struct {
	Choices []QwenChoice `json:"choices"`
	Text    string       `json:"text"` // 兼容不同的响应格式
}

type QwenChoice struct {
	Message      QwenMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type QwenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// NewQwenClient creates a new Qwen AI client
func NewQwenClient(config QwenConfig) AIClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	}
	if config.Model == "" {
		config.Model = "qwen-plus" // 默认使用qwen-plus，性价比较好
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 1000
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &QwenClient{
		apiKey:    config.APIKey,
		baseURL:   config.BaseURL,
		model:     config.Model,
		maxTokens: config.MaxTokens,
		timeout:   config.Timeout,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: utils.WithField("component", "qwen-ai-client"),
	}
}

// AnalyzeContext analyzes context using Qwen-3 and generates skill match query
func (c *QwenClient) AnalyzeContext(ctx context.Context, needDescription string, availableSkills []models.AgentSkill) (*models.SkillMatchQuery, error) {
	c.logger.Info("Qwen AI analyzing context: %s", needDescription)

	// 构建提示词
	prompt := c.buildAnalysisPrompt(needDescription, availableSkills)

	// 调用Qwen API
	response, err := c.callQwenAPI(ctx, prompt)
	if err != nil {
		c.logger.Error("Qwen API call failed: %v", err)
		return nil, fmt.Errorf("qwen API call failed: %w", err)
	}

	// 解析AI回复为SkillMatchQuery
	query, err := c.parseAIResponse(response)
	if err != nil {
		c.logger.Error("Failed to parse Qwen response: %v", err)
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	c.logger.Debug("Qwen generated query: %+v", query)
	return query, nil
}

// buildAnalysisPrompt constructs the prompt for context analysis
func (c *QwenClient) buildAnalysisPrompt(needDescription string, availableSkills []models.AgentSkill) string {
	// 构建技能列表JSON
	skillsJSON, _ := json.MarshalIndent(availableSkills, "", "  ")

	// 设计专门的提示词模板
	prompt := fmt.Sprintf(`你是一个智能代理路由系统的分析器。你的任务是根据用户需求描述，从可用技能列表中找到最匹配的技能。

用户需求描述：
%s

当前系统中可用的代理技能列表：
%s

请仔细分析用户需求，并返回一个JSON格式的技能匹配查询。要求：

1. 理解用户的真实意图和具体需求
2. 从可用技能中找到最相关的技能ID
3. 提取关键词和相关标签
4. 设置合适的优先级策略

**匹配严格标准：**
- 只有当技能与用户需求有直接、明确的关联时才返回该技能
- 技能必须能够直接解决用户的核心需求，而不是间接相关
- 置信度低于0.8的匹配应该被忽略
- 宁可返回空数组也不要勉强匹配不相关的技能

请严格按照以下JSON格式返回，不要包含任何其他内容：

{
  "query": "主要搜索关键词",
  "skills": ["精确匹配的技能ID列表"],
  "tags": ["相关标签列表"],
  "priority": "accuracy",
  "metadata": {
    "confidence": "0.95",
    "reasoning": "匹配理由说明"
  }
}

注意：
- skills字段只包含在可用技能列表中真实存在的技能ID，且必须高度相关
- tags要与用户需求和匹配技能高度相关  
- 确保返回的是合法的JSON格式
- 如果没有高度匹配的技能（置信度<0.8），skills必须为空数组
- 例如：数学学习需求不应该匹配到文本分析技能`, needDescription, string(skillsJSON))

	return prompt
}

// callQwenAPI makes HTTP request to Qwen API
func (c *QwenClient) callQwenAPI(ctx context.Context, prompt string) (string, error) {
	// 构建请求体
	reqBody := QwenRequest{
		Model: c.model,
		Input: QwenInput{
			Messages: []QwenMessage{
				{
					Role:    "user",
					Content: prompt,
				},
			},
		},
		Parameters: QwenParameters{
			MaxTokens:   c.maxTokens,
			Temperature: 0.1, // 低温度确保稳定输出
			TopP:        0.8,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-DashScope-SSE", "disable") // 禁用流式传输

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// 先记录原始响应用于调试
	c.logger.Debug("Raw Qwen API response: %s", string(respBody))

	// 解析响应
	var qwenResp QwenResponse
	if err := json.Unmarshal(respBody, &qwenResp); err != nil {
		c.logger.Error("Failed to parse JSON response: %v", err)
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	// 提取AI回复文本 - 兼容多种响应格式
	var content string
	if len(qwenResp.Output.Choices) > 0 {
		// 标准格式：使用choices中的message.content
		content = qwenResp.Output.Choices[0].Message.Content
	} else if qwenResp.Output.Text != "" {
		// 备用格式：直接从output.text获取
		content = qwenResp.Output.Text
		c.logger.Debug("Using text field from response instead of choices")
	} else {
		c.logger.Error("No content found in response. Full response: %+v", qwenResp)
		return "", fmt.Errorf("no content in AI response")
	}

	c.logger.Debug("Qwen API response: %s", content)

	return content, nil
}

// parseAIResponse parses AI response text into SkillMatchQuery
func (c *QwenClient) parseAIResponse(response string) (*models.SkillMatchQuery, error) {
	// 清理响应文本，提取JSON部分
	response = strings.TrimSpace(response)

	// 查找JSON开始和结束位置
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[start : end+1]

	// 解析JSON
	var query models.SkillMatchQuery
	if err := json.Unmarshal([]byte(jsonStr), &query); err != nil {
		c.logger.Warn("Failed to parse JSON, trying to fix: %v", err)
		// 尝试修复常见的JSON问题
		fixedJSON := c.fixJSONFormat(jsonStr)
		if err := json.Unmarshal([]byte(fixedJSON), &query); err != nil {
			return nil, fmt.Errorf("failed to parse JSON after fix: %w", err)
		}
	}

	// 验证和设置默认值
	if query.Query == "" {
		query.Query = "general"
	}
	if query.Priority == "" {
		query.Priority = "accuracy"
	}
	if query.Metadata == nil {
		query.Metadata = make(map[string]string)
	}
	query.Metadata["ai_provider"] = "qwen-3"
	query.Metadata["model"] = c.model

	// 置信度检查 - 如果置信度低于0.8，清空skills数组
	if confidenceStr, exists := query.Metadata["confidence"]; exists {
		if confidence, err := strconv.ParseFloat(confidenceStr, 64); err == nil {
			if confidence < 0.8 {
				c.logger.Info("Low confidence match (%.2f < 0.8), clearing skills", confidence)
				query.Skills = []string{} // 清空技能列表
				query.Metadata["filtered_reason"] = "low_confidence"
			} else {
				c.logger.Debug("High confidence match: %.2f", confidence)
			}
		}
	}

	return &query, nil
}

// fixJSONFormat attempts to fix common JSON formatting issues
func (c *QwenClient) fixJSONFormat(jsonStr string) string {
	// 移除可能的markdown代码块标记
	jsonStr = strings.ReplaceAll(jsonStr, "```json", "")
	jsonStr = strings.ReplaceAll(jsonStr, "```", "")

	// 移除多余的空白字符
	jsonStr = strings.TrimSpace(jsonStr)

	return jsonStr
}

// Health checks if the Qwen AI service is available
func (c *QwenClient) Health(ctx context.Context) error {
	// 发送一个简单的测试请求
	testSkills := []models.AgentSkill{
		{ID: "test", Name: "测试技能", Description: "测试用技能"},
	}

	_, err := c.AnalyzeContext(ctx, "测试连接", testSkills)
	return err
}
