package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"agenthub/pkg/client"
)

// AgentManager 演示高级代理管理功能
type AgentManager struct {
	client *client.Client
	agents map[string]*client.RegisteredAgent
	mutex  sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAgentManager 创建代理管理器
func NewAgentManager(baseURL string) *AgentManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &AgentManager{
		client: client.NewClient(client.ClientConfig{
			BaseURL: baseURL,
			Timeout: 30 * time.Second,
		}),
		agents: make(map[string]*client.RegisteredAgent),
		ctx:    ctx,
		cancel: cancel,
	}
}

// RegisterMultipleAgents 批量注册代理
func (am *AgentManager) RegisterMultipleAgents() error {
	fmt.Println("=== 批量注册代理 ===")

	agentConfigs := []struct {
		name   string
		port   int
		skills []client.AgentSkill
	}{
		{
			name: "nlp-specialist",
			port: 8081,
			skills: []client.AgentSkill{
				{
					ID:          "sentiment_analysis",
					Name:        "情感分析",
					Description: "分析文本情感倾向",
					Tags:        []string{"nlp", "sentiment", "analysis"},
					Examples:    []string{"分析评论情感", "判断文本正负面"},
				},
				{
					ID:          "named_entity_recognition",
					Name:        "命名实体识别",
					Description: "识别文本中的人名、地名、组织等",
					Tags:        []string{"nlp", "ner", "extraction"},
					Examples:    []string{"提取人名地名", "识别公司名称"},
				},
			},
		},
		{
			name: "data-scientist",
			port: 8082,
			skills: []client.AgentSkill{
				{
					ID:          "regression_analysis",
					Name:        "回归分析",
					Description: "进行线性和非线性回归分析",
					Tags:        []string{"statistics", "regression", "modeling"},
					Examples:    []string{"销售预测", "趋势分析"},
				},
				{
					ID:          "clustering",
					Name:        "聚类分析",
					Description: "对数据进行聚类分组",
					Tags:        []string{"ml", "clustering", "unsupervised"},
					Examples:    []string{"用户分群", "市场细分"},
				},
			},
		},
		{
			name: "cv-expert",
			port: 8083,
			skills: []client.AgentSkill{
				{
					ID:          "face_recognition",
					Name:        "人脸识别",
					Description: "识别和验证人脸",
					Tags:        []string{"cv", "face", "recognition"},
					Examples:    []string{"身份验证", "人脸检测"},
				},
				{
					ID:          "image_classification",
					Name:        "图像分类",
					Description: "对图像进行分类识别",
					Tags:        []string{"cv", "classification", "deep learning"},
					Examples:    []string{"商品分类", "医学影像分析"},
				},
			},
		},
	}

	var wg sync.WaitGroup
	errorChan := make(chan error, len(agentConfigs))

	for _, config := range agentConfigs {
		wg.Add(1)
		go func(cfg struct {
			name   string
			port   int
			skills []client.AgentSkill
		}) {
			defer wg.Done()

			req := &client.RegisterRequest{
				AgentCard: client.AgentCard{
					Name:        cfg.name,
					Description: fmt.Sprintf("专业%s代理", cfg.name),
					URL:         fmt.Sprintf("http://localhost:%d", cfg.port),
					Version:     "1.0.0",
					Capabilities: client.AgentCapabilities{
						Streaming: true,
					},
					DefaultInputModes:  []string{"text", "json"},
					DefaultOutputModes: []string{"json"},
					Skills:             cfg.skills,
				},
				Host: "localhost",
				Port: cfg.port,
			}

			_, err := am.client.RegisterAgent(am.ctx, req)
			if err != nil {
				errorChan <- fmt.Errorf("注册代理 %s 失败: %w", cfg.name, err)
			} else {
				fmt.Printf("✅ 成功注册代理: %s (端口: %d)\n", cfg.name, cfg.port)
			}
		}(config)
	}

	wg.Wait()
	close(errorChan)

	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Printf("❌ %v\n", err)
		}
		return fmt.Errorf("部分代理注册失败")
	}

	return nil
}

// MonitorAgents 监控代理状态
func (am *AgentManager) MonitorAgents() {
	fmt.Println("\n=== 启动代理监控 ===")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-am.ctx.Done():
				return
			case <-ticker.C:
				am.checkAgentHealth()
			}
		}
	}()

	fmt.Println("✅ 代理监控已启动（每5秒检查一次）")
}

// checkAgentHealth 检查代理健康状态
func (am *AgentManager) checkAgentHealth() {
	agents, err := am.client.ListAgents(am.ctx)
	if err != nil {
		log.Printf("获取代理列表失败: %v", err)
		return
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	for _, agent := range agents {
		am.agents[agent.ID] = agent

		// 检查代理是否长时间未活跃
		if time.Since(agent.LastSeen) > 10*time.Second && agent.IsActive() {
			fmt.Printf("⚠️  代理 %s 超过10秒未活跃，更新状态为非活跃\n", agent.ID)
			err := am.client.UpdateAgentStatus(am.ctx, agent.ID, "inactive")
			if err != nil {
				log.Printf("更新代理状态失败: %v", err)
			}
		}
	}
}

// AutoSendHeartbeats 自动发送心跳
func (am *AgentManager) AutoSendHeartbeats() {
	fmt.Println("\n=== 启动自动心跳 ===")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-am.ctx.Done():
				return
			case <-ticker.C:
				am.sendHeartbeats()
			}
		}
	}()

	fmt.Println("✅ 自动心跳已启动（每3秒发送一次）")
}

// sendHeartbeats 发送心跳到所有活跃代理
func (am *AgentManager) sendHeartbeats() {
	am.mutex.RLock()
	agentIDs := make([]string, 0, len(am.agents))
	for id, agent := range am.agents {
		if agent.IsActive() {
			agentIDs = append(agentIDs, id)
		}
	}
	am.mutex.RUnlock()

	for _, agentID := range agentIDs {
		go func(id string) {
			err := am.client.SendHeartbeat(am.ctx, id)
			if err != nil {
				log.Printf("发送心跳失败 (代理: %s): %v", id, err)
			} else {
				fmt.Printf("💓 心跳已发送: %s\n", id)
			}
		}(agentID)
	}
}

// IntelligentTaskRouting 智能任务路由演示
func (am *AgentManager) IntelligentTaskRouting() error {
	fmt.Println("\n=== 智能任务路由演示 ===")

	tasks := []struct {
		description  string
		context      string
		expectedType string
	}{
		{
			description:  "需要分析客户评论中表达的情感，判断满意度",
			context:      "客户满意度调研",
			expectedType: "NLP",
		},
		{
			description:  "需要对销售数据进行回归分析，预测下季度业绩",
			context:      "业绩预测分析",
			expectedType: "数据科学",
		},
		{
			description:  "需要识别监控视频中的人脸，进行身份验证",
			context:      "安防系统",
			expectedType: "计算机视觉",
		},
		{
			description:  "需要从客户反馈中提取公司名称和人名信息",
			context:      "客户关系管理",
			expectedType: "NLP",
		},
		{
			description:  "需要对用户行为数据进行聚类，识别不同的用户群体",
			context:      "用户画像分析",
			expectedType: "数据科学",
		},
	}

	for i, task := range tasks {
		fmt.Printf("\n%d. 任务: %s\n", i+1, task.description)
		fmt.Printf("   场景: %s\n", task.context)

		req := &client.ContextAnalysisRequest{
			NeedDescription: task.description,
			UserContext:     task.context,
		}

		resp, err := am.client.AnalyzeContext(am.ctx, req)
		if err != nil {
			fmt.Printf("   ❌ 路由失败: %v\n", err)
			continue
		}

		if resp.Success && len(resp.MatchedSkills) > 0 {
			fmt.Printf("   ✅ 找到匹配技能: %s\n", resp.MatchedSkills[0].Name)
			fmt.Printf("   🎯 技能标签: %v\n", resp.MatchedSkills[0].Tags)

			if resp.RouteResult != nil {
				fmt.Printf("   🔗 路由目标: %s\n", resp.RouteResult.AgentURL)
				fmt.Printf("   🔧 推荐技能: %s\n", resp.RouteResult.SkillName)
			}

			if resp.AnalysisResult != nil && resp.AnalysisResult.SuggestedWorkflow != "" {
				fmt.Printf("   💡 工作流建议: %s\n", resp.AnalysisResult.SuggestedWorkflow)
			}
		} else {
			fmt.Printf("   ⚠️  未找到匹配的技能: %s\n", resp.Message)
		}

		// 模拟处理延迟
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

// PerformanceTest 性能测试
func (am *AgentManager) PerformanceTest() error {
	fmt.Println("\n=== 性能测试 ===")

	tests := []struct {
		name       string
		concurrent int
		requests   int
	}{
		{"低并发测试", 5, 20},
		{"中等并发测试", 15, 50},
		{"高并发测试", 30, 100},
	}

	for _, test := range tests {
		fmt.Printf("\n🚀 开始 %s (并发: %d, 请求: %d)\n", test.name, test.concurrent, test.requests)

		start := time.Now()
		err := am.runConcurrentTest(test.concurrent, test.requests)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("   ❌ 测试失败: %v\n", err)
		} else {
			fmt.Printf("   ✅ 测试完成\n")
			fmt.Printf("   📊 总耗时: %v\n", elapsed)
			fmt.Printf("   📊 平均响应时间: %v\n", elapsed/time.Duration(test.requests))
			fmt.Printf("   📊 QPS: %.2f\n", float64(test.requests)/elapsed.Seconds())
		}
	}

	return nil
}

// runConcurrentTest 执行并发测试
func (am *AgentManager) runConcurrentTest(concurrent, totalRequests int) error {
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrent)
	errorCount := int64(0)
	successCount := int64(0)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(reqID int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			req := &client.ContextAnalysisRequest{
				NeedDescription: fmt.Sprintf("性能测试请求 %d", reqID),
				UserContext:     "性能测试场景",
			}

			_, err := am.client.AnalyzeContext(am.ctx, req)
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		return fmt.Errorf("%d 个请求失败，%d 个请求成功", errorCount, successCount)
	}

	return nil
}

// GetAgentStatistics 获取代理统计信息
func (am *AgentManager) GetAgentStatistics() error {
	fmt.Println("\n=== 代理统计信息 ===")

	agents, err := am.client.ListAgents(am.ctx)
	if err != nil {
		return fmt.Errorf("获取代理列表失败: %w", err)
	}

	stats := make(map[string]int)
	skillStats := make(map[string]int)
	tagStats := make(map[string]int)

	for _, agent := range agents {
		stats[agent.Status]++

		for _, skill := range agent.AgentCard.Skills {
			skillStats[skill.Name]++
			for _, tag := range skill.Tags {
				tagStats[tag]++
			}
		}
	}

	fmt.Printf("📊 总代理数: %d\n", len(agents))
	fmt.Println("\n📈 代理状态分布:")
	for status, count := range stats {
		fmt.Printf("   %s: %d\n", status, count)
	}

	fmt.Println("\n🎯 技能分布 (Top 5):")
	skillList := make([]struct {
		name  string
		count int
	}, 0, len(skillStats))
	for name, count := range skillStats {
		skillList = append(skillList, struct {
			name  string
			count int
		}{name, count})
	}

	// 简单排序
	for i := 0; i < len(skillList)-1; i++ {
		for j := i + 1; j < len(skillList); j++ {
			if skillList[i].count < skillList[j].count {
				skillList[i], skillList[j] = skillList[j], skillList[i]
			}
		}
	}

	for i, skill := range skillList {
		if i >= 5 {
			break
		}
		fmt.Printf("   %s: %d\n", skill.name, skill.count)
	}

	fmt.Println("\n🏷️  标签分布 (Top 5):")
	tagList := make([]struct {
		name  string
		count int
	}, 0, len(tagStats))
	for name, count := range tagStats {
		tagList = append(tagList, struct {
			name  string
			count int
		}{name, count})
	}

	// 简单排序
	for i := 0; i < len(tagList)-1; i++ {
		for j := i + 1; j < len(tagList); j++ {
			if tagList[i].count < tagList[j].count {
				tagList[i], tagList[j] = tagList[j], tagList[i]
			}
		}
	}

	for i, tag := range tagList {
		if i >= 5 {
			break
		}
		fmt.Printf("   %s: %d\n", tag.name, tag.count)
	}

	return nil
}

// Cleanup 清理资源
func (am *AgentManager) Cleanup() error {
	fmt.Println("\n=== 清理测试环境 ===")

	am.cancel() // 停止所有后台任务

	agents, err := am.client.ListAgents(am.ctx)
	if err != nil {
		return fmt.Errorf("获取代理列表失败: %w", err)
	}

	var wg sync.WaitGroup
	for _, agent := range agents {
		wg.Add(1)
		go func(agentID string) {
			defer wg.Done()

			err := am.client.RemoveAgent(am.ctx, agentID)
			if err != nil {
				fmt.Printf("   删除代理 %s 失败: %v\n", agentID, err)
			} else {
				fmt.Printf("   ✅ 删除代理: %s\n", agentID)
			}
		}(agent.ID)
	}

	wg.Wait()
	fmt.Println("✅ 清理完成")
	return nil
}

func main() {
	fmt.Println("🚀 AgentHub 高级功能演示")
	fmt.Println("============================")

	// 创建代理管理器
	manager := NewAgentManager("http://localhost:8080")
	defer manager.Cleanup()

	// 检查服务健康状态
	health, err := manager.client.HealthCheck(context.Background())
	if err != nil {
		log.Fatalf("❌ 服务不可用: %v", err)
	}
	fmt.Printf("✅ 服务状态: %s\n\n", health.Status)

	// 批量注册代理
	if err := manager.RegisterMultipleAgents(); err != nil {
		log.Printf("❌ 批量注册失败: %v", err)
	}

	// 等待代理注册完成
	time.Sleep(2 * time.Second)

	// 启动代理监控
	manager.MonitorAgents()

	// 启动自动心跳
	manager.AutoSendHeartbeats()

	// 等待监控和心跳启动
	time.Sleep(1 * time.Second)

	// 智能任务路由演示
	if err := manager.IntelligentTaskRouting(); err != nil {
		log.Printf("❌ 智能路由失败: %v", err)
	}

	// 性能测试
	if err := manager.PerformanceTest(); err != nil {
		log.Printf("❌ 性能测试失败: %v", err)
	}

	// 获取统计信息
	if err := manager.GetAgentStatistics(); err != nil {
		log.Printf("❌ 获取统计信息失败: %v", err)
	}

	// 让监控和心跳运行一段时间
	fmt.Println("\n⏰ 让系统运行10秒钟以观察监控和心跳...")
	time.Sleep(10 * time.Second)

	fmt.Println("\n🎉 高级功能演示完成!")
}
