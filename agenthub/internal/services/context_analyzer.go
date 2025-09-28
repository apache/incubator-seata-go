package services

import (
	"context"
	"fmt"
	"strings"

	"agenthub/pkg/models"
	"agenthub/pkg/storage"
	"agenthub/pkg/utils"
)

// ContextAnalyzer analyzes agent needs and performs skill matching
type ContextAnalyzer struct {
	storage  storage.Storage
	logger   *utils.Logger
	aiClient AIClient // AI服务客户端接口
}

// AIClient interface for AI service integration
type AIClient interface {
	AnalyzeContext(ctx context.Context, needDescription string, availableSkills []models.AgentSkill) (*models.SkillMatchQuery, error)
}

// ContextAnalyzerConfig holds context analyzer configuration
type ContextAnalyzerConfig struct {
	Storage  storage.Storage
	Logger   *utils.Logger
	AIClient AIClient
}

// NewContextAnalyzer creates a new context analyzer
func NewContextAnalyzer(config ContextAnalyzerConfig) *ContextAnalyzer {
	logger := config.Logger
	if logger == nil {
		logger = utils.WithField("component", "context-analyzer")
	}

	return &ContextAnalyzer{
		storage:  config.Storage,
		logger:   logger,
		aiClient: config.AIClient,
	}
}

// AnalyzeAndRoute analyzes the need description and routes to appropriate agent
func (c *ContextAnalyzer) AnalyzeAndRoute(ctx context.Context, req *models.ContextAnalysisRequest) (*models.ContextAnalysisResponse, error) {
	c.logger.Info("Analyzing context for need: %s", req.NeedDescription)

	// 1. Get global agent card with all available skills
	globalSkills, err := c.getAvailableSkills(ctx)
	if err != nil {
		return &models.ContextAnalysisResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get available skills: %v", err),
		}, nil
	}

	c.logger.Debug("Found %d available skills for analysis", len(globalSkills))

	// 2. Use AI to analyze need and generate skill match query
	matchQuery, err := c.aiClient.AnalyzeContext(ctx, req.NeedDescription, globalSkills)
	if err != nil {
		c.logger.Error("AI analysis failed: %v", err)
		return &models.ContextAnalysisResponse{
			Success: false,
			Message: fmt.Sprintf("AI analysis failed: %v", err),
		}, nil
	}

	c.logger.Debug("AI generated match query: %+v", matchQuery)

	// 3. Extract analysis result from AI query
	analysisResult := &models.AnalysisResult{
		RequiredSkills:    matchQuery.Skills,
		ContextTags:       matchQuery.Tags,
		SuggestedWorkflow: c.generateWorkflowSuggestion(req.NeedDescription, matchQuery),
	}

	// 4. Match skills based on AI query
	matchedSkills := c.matchSkillsWithQuery(globalSkills, matchQuery)
	if len(matchedSkills) == 0 {
		return &models.ContextAnalysisResponse{
			Success:        false,
			Message:        "No matching skills found for the given need",
			AnalysisResult: analysisResult,
		}, nil
	}

	c.logger.Info("Found %d matching skills", len(matchedSkills))

	// 5. Route to the best matching agent
	routeResult, err := c.routeToAgent(ctx, matchedSkills[0])
	if err != nil {
		return &models.ContextAnalysisResponse{
			Success:        false,
			Message:        fmt.Sprintf("Failed to route to agent: %v", err),
			AnalysisResult: analysisResult,
		}, nil
	}

	return &models.ContextAnalysisResponse{
		Success:        true,
		Message:        "Successfully analyzed and routed request",
		MatchedSkills:  matchedSkills,
		RouteResult:    routeResult,
		AnalysisResult: analysisResult,
	}, nil
}

// getAvailableSkills retrieves all available skills from global agent card
func (c *ContextAnalyzer) getAvailableSkills(ctx context.Context) ([]models.AgentSkill, error) {
	// Try to get global agent card if storage supports it
	if skillStorage, ok := c.storage.(*storage.SkillBasedNamingServerStorage); ok {
		globalCard := skillStorage.GetGlobalAgentCard()
		return globalCard.Skills, nil
	}

	// Fallback: get skills from all registered agents
	resources, err := c.storage.List(ctx, map[string]interface{}{
		"kind": "RegisteredAgent",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	var allSkills []models.AgentSkill
	skillMap := make(map[string]bool) // deduplicate by skill ID

	for _, resource := range resources {
		if agent, ok := resource.(*models.RegisteredAgent); ok && agent.IsActive() {
			for _, skill := range agent.AgentCard.Skills {
				if !skillMap[skill.ID] {
					allSkills = append(allSkills, skill)
					skillMap[skill.ID] = true
				}
			}
		}
	}

	return allSkills, nil
}

// matchSkillsWithQuery matches skills based on AI-generated query
func (c *ContextAnalyzer) matchSkillsWithQuery(availableSkills []models.AgentSkill, query *models.SkillMatchQuery) []models.AgentSkill {
	var matchedSkills []models.AgentSkill
	normalizedQuery := utils.NormalizeString(query.Query)

	// Score and rank skills
	type skillScore struct {
		skill models.AgentSkill
		score int
	}
	var scoredSkills []skillScore

	for _, skill := range availableSkills {
		score := c.calculateSkillScore(skill, query, normalizedQuery)
		if score > 0 {
			scoredSkills = append(scoredSkills, skillScore{skill: skill, score: score})
		}
	}

	// Sort by score (highest first)
	for i := 0; i < len(scoredSkills)-1; i++ {
		for j := i + 1; j < len(scoredSkills); j++ {
			if scoredSkills[i].score < scoredSkills[j].score {
				scoredSkills[i], scoredSkills[j] = scoredSkills[j], scoredSkills[i]
			}
		}
	}

	// Return top matches
	for _, scored := range scoredSkills {
		matchedSkills = append(matchedSkills, scored.skill)
	}

	return matchedSkills
}

// calculateSkillScore calculates relevance score for a skill
func (c *ContextAnalyzer) calculateSkillScore(skill models.AgentSkill, query *models.SkillMatchQuery, normalizedQuery string) int {
	score := 0

	// Exact skill ID match (highest priority)
	for _, skillID := range query.Skills {
		if skill.ID == skillID {
			score += 100
		}
	}

	// Query keyword matches
	if utils.ContainsIgnoreCase(skill.ID, normalizedQuery) {
		score += 50
	}
	if utils.ContainsIgnoreCase(skill.Name, normalizedQuery) {
		score += 40
	}
	if utils.ContainsIgnoreCase(skill.Description, normalizedQuery) {
		score += 30
	}

	// Tag matches
	for _, queryTag := range query.Tags {
		if utils.MatchTags(queryTag, skill.Tags) {
			score += 20
		}
	}

	return score
}

// routeToAgent routes request to the agent that has the matching skill
// This method returns basic routing info, actual Hub discovery is done by the caller
func (c *ContextAnalyzer) routeToAgent(ctx context.Context, skill models.AgentSkill) (*models.RouteResult, error) {
	// Get agent URL for this skill
	var agentURL string

	if skillStorage, ok := c.storage.(*storage.SkillBasedNamingServerStorage); ok {
		// Use skill-to-URL mapping
		url, err := skillStorage.DiscoverUrlBySkill(ctx, skill.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to discover URL for skill %s: %w", skill.ID, err)
		}
		agentURL = url
	} else {
		return nil, fmt.Errorf("skill-based routing not supported with current storage")
	}

	return &models.RouteResult{
		AgentURL:  agentURL,
		SkillID:   skill.ID,
		SkillName: skill.Name,
		// AgentInfo will be populated by the caller through Hub discovery
	}, nil
}

// generateWorkflowSuggestion generates meaningful workflow suggestions based on user needs and AI analysis
func (c *ContextAnalyzer) generateWorkflowSuggestion(needDescription string, matchQuery *models.SkillMatchQuery) string {
	// Generate suggestions based on need description and tags (prioritize custom logic)
	need := strings.ToLower(needDescription)
	tags := matchQuery.Tags

	// Common learning patterns
	if strings.Contains(need, "学习") || strings.Contains(need, "learn") {
		return c.generateLearningWorkflow(need, tags)
	}

	// Analysis patterns
	if strings.Contains(need, "分析") || strings.Contains(need, "analyze") {
		return c.generateAnalysisWorkflow(need, tags)
	}

	// Processing patterns
	if strings.Contains(need, "处理") || strings.Contains(need, "process") {
		return c.generateProcessingWorkflow(need, tags)
	}

	// Translation patterns
	if strings.Contains(need, "翻译") || strings.Contains(need, "translate") {
		return "需要支持多语言文本翻译的Agent，建议注册具备自然语言处理和翻译技能的Agent"
	}

	// Default workflow based on tags
	return c.generateDefaultWorkflow(need, tags)
}

// generateLearningWorkflow generates workflow for learning requests
func (c *ContextAnalyzer) generateLearningWorkflow(need string, tags []string) string {
	domain := "通用知识"

	// Identify learning domain from tags (support both English and Chinese)
	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		switch tagLower {
		case "computer", "programming", "coding", "计算机", "编程":
			domain = "计算机编程"
		case "math", "mathematics", "数学":
			domain = "数学"
		case "science", "科学":
			domain = "科学"
		case "language", "语言", "外语":
			domain = "语言"
		case "business", "商业", "管理":
			domain = "商业管理"
		}
	}

	return fmt.Sprintf("需要教育辅导类Agent来提供%s学习支持，建议注册具备以下能力的Agent：1) 知识讲解 2) 练习题生成 3) 学习进度跟踪 4) 个性化推荐", domain)
}

// generateAnalysisWorkflow generates workflow for analysis requests
func (c *ContextAnalyzer) generateAnalysisWorkflow(need string, tags []string) string {
	analysisType := "数据分析"

	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		switch tagLower {
		case "text", "nlp", "文本", "自然语言":
			analysisType = "文本分析"
		case "image", "vision", "图像", "图片", "视觉":
			analysisType = "图像分析"
		case "data", "statistics", "数据", "统计":
			analysisType = "数据统计分析"
		case "sentiment", "emotion", "情感", "情绪":
			analysisType = "情感分析"
		}
	}

	return fmt.Sprintf("需要%s能力的Agent，建议注册具备相关算法和模型的Agent来处理此类分析任务", analysisType)
}

// generateProcessingWorkflow generates workflow for processing requests
func (c *ContextAnalyzer) generateProcessingWorkflow(need string, tags []string) string {
	processingType := "数据处理"

	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		switch tagLower {
		case "text", "文本":
			processingType = "文本处理"
		case "image", "图像", "图片":
			processingType = "图像处理"
		case "audio", "音频", "声音":
			processingType = "音频处理"
		case "video", "视频":
			processingType = "视频处理"
		}
	}

	return fmt.Sprintf("需要%s能力的Agent，建议注册支持相关格式和算法的处理Agent", processingType)
}

// generateDefaultWorkflow generates default workflow suggestions
func (c *ContextAnalyzer) generateDefaultWorkflow(need string, tags []string) string {
	if len(tags) == 0 {
		return "建议明确具体需求，以便系统推荐合适的Agent类型"
	}

	tagStr := strings.Join(tags, "、")
	return fmt.Sprintf("根据需求特征，建议注册具备[%s]相关技能的Agent来处理此类请求", tagStr)
}
