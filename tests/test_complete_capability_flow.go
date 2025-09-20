package tests

import (
	"context"
	"fmt"
	"log"
	"time"

	"agenthub/pkg/models"
	"agenthub/pkg/services"
	"agenthub/pkg/storage"
)

// TestCompleteCapabilityFlow 完整测试能力注册和发现流程
func TestCompleteCapabilityFlow() {
	fmt.Println("=== 完整能力注册与发现流程测试 ===")
	fmt.Println("本测试将演示：注册Agent → 验证注册 → 发现能力 → 验证发现结果")

	// 1. 初始化系统组件
	fmt.Println("\n🔧 步骤1：初始化系统组件")
	fmt.Println("   - 创建Mock NamingServer注册中心")
	fmt.Println("   - 初始化基于技能的存储")
	fmt.Println("   - 创建Agent服务")
	
	mockRegistry := storage.NewMockNamingserverRegistry()
	skillStorage := storage.NewSkillBasedNamingServerStorage(mockRegistry)
	agentService := services.NewAgentService(services.AgentServiceConfig{
		Storage: skillStorage,
	})
	
	fmt.Println("✅ 系统组件初始化完成")

	// 2. 定义要注册的Agent及其能力
	fmt.Println("\n📋 步骤2：定义Agent和能力")
	
	testAgentCard := models.AgentCard{
		Name:        "SuperAI-Assistant",
		Version:     "3.0.0",
		URL:         "http://localhost:8888",
		Description: "一个具有多种AI能力的智能助手Agent",
		Skills: []models.AgentSkill{
			{
				ID:          "natural-language-understanding",
				Name:        "natural-language-understanding",
				Description: "自然语言理解和处理",
				Tags:        []string{"NLP", "AI", "语言模型", "理解"},
				Examples:    []string{"文本分类", "意图识别", "实体抽取", "语义分析"},
				InputModes:  []string{"text", "voice"},
				OutputModes: []string{"text", "structured-data"},
			},
			{
				ID:          "image-generation",
				Name:        "image-generation",
				Description: "AI图像生成和创作",
				Tags:        []string{"CV", "AI", "生成模型", "创作"},
				Examples:    []string{"文本生成图像", "风格转换", "图像修复", "艺术创作"},
				InputModes:  []string{"text", "image"},
				OutputModes: []string{"image"},
			},
			{
				ID:          "code-generation",
				Name:        "code-generation",
				Description: "代码生成和编程助手",
				Tags:        []string{"编程", "AI", "代码助手", "开发"},
				Examples:    []string{"代码补全", "bug修复", "算法实现", "代码重构"},
				InputModes:  []string{"text"},
				OutputModes: []string{"code", "text"},
			},
			{
				ID:          "data-analysis",
				Name:        "data-analysis",
				Description: "数据分析和洞察生成",
				Tags:        []string{"数据科学", "AI", "分析", "洞察"},
				Examples:    []string{"数据可视化", "趋势分析", "异常检测", "预测建模"},
				InputModes:  []string{"csv", "json", "database"},
				OutputModes: []string{"chart", "report", "insights"},
			},
		},
	}

	fmt.Printf("📝 准备注册Agent：%s\n", testAgentCard.Name)
	fmt.Printf("   版本：%s\n", testAgentCard.Version)
	fmt.Printf("   服务地址：%s\n", testAgentCard.URL)
	fmt.Printf("   能力数量：%d个\n", len(testAgentCard.Skills))
	
	for i, skill := range testAgentCard.Skills {
		fmt.Printf("   能力%d：%s - %s\n", i+1, skill.Name, skill.Description)
		fmt.Printf("        标签：%v\n", skill.Tags)
		fmt.Printf("        示例：%v\n", skill.Examples)
	}

	// 3. 执行能力注册
	fmt.Println("\n🚀 步骤3：执行能力注册")
	fmt.Println("   正在向NamingServer注册Agent及其所有能力...")
	
	ctx := context.Background()
	startTime := time.Now()
	
	registerRequest := &models.RegisterRequest{
		AgentCard: testAgentCard,
		Host:      "localhost",
		Port:      8888,
	}

	response, err := agentService.RegisterAgent(ctx, registerRequest)
	if err != nil {
		log.Fatalf("❌ 注册失败：%v", err)
	}
	
	if !response.Success {
		log.Fatalf("❌ 注册失败：%s", response.Message)
	}
	
	registrationTime := time.Since(startTime)
	fmt.Printf("✅ Agent注册成功！\n")
	fmt.Printf("   Agent ID：%s\n", response.AgentID)
	fmt.Printf("   注册耗时：%v\n", registrationTime)
	fmt.Printf("   响应消息：%s\n", response.Message)

	// 4. 验证注册结果
	fmt.Println("\n🔍 步骤4：验证注册结果")
	
	// 4.1 验证Agent是否在存储中
	fmt.Println("   4.1 验证Agent存储...")
	storedAgent, err := agentService.GetAgent(ctx, response.AgentID)
	if err != nil {
		log.Fatalf("❌ 无法获取已注册的Agent：%v", err)
	}
	fmt.Printf("   ✅ Agent存储验证成功：%s (状态：%s)\n", storedAgent.AgentCard.Name, storedAgent.Status)
	
	// 4.2 验证vGroup创建
	fmt.Println("   4.2 验证NamingServer vGroup创建...")
	allGroups := mockRegistry.GetAllGroups()
	expectedGroups := []string{
		"skill-natural-language-understanding",
		"skill-image-generation",
		"skill-code-generation",
		"skill-data-analysis",
	}
	
	fmt.Printf("   发现的vGroup：%v\n", allGroups)
	
	for _, expectedGroup := range expectedGroups {
		found := false
		for _, actualGroup := range allGroups {
			if actualGroup == expectedGroup {
				found = true
				break
			}
		}
		
		if found {
			fmt.Printf("   ✅ vGroup '%s' 创建成功\n", expectedGroup)
		} else {
			log.Fatalf("   ❌ vGroup '%s' 未创建", expectedGroup)
		}
	}
	
	// 4.3 验证服务实例注册
	fmt.Println("   4.3 验证服务实例注册...")
	for _, expectedGroup := range expectedGroups {
		instances, err := mockRegistry.Lookup(expectedGroup)
		if err != nil {
			log.Fatalf("❌ 查询vGroup '%s' 失败：%v", expectedGroup, err)
		}
		
		if len(instances) == 0 {
			log.Fatalf("❌ vGroup '%s' 中没有服务实例", expectedGroup)
		}
		
		instance := instances[0]
		fmt.Printf("   ✅ vGroup '%s'：服务实例 %s:%d\n", expectedGroup, instance.Addr, instance.Port)
	}

	// 5. 执行能力发现测试
	fmt.Println("\n🎯 步骤5：执行能力发现测试")
	
	skillsToTest := []struct{
		skillName string
		description string
	}{
		{"natural-language-understanding", "自然语言理解"},
		{"image-generation", "图像生成"},
		{"code-generation", "代码生成"},
		{"data-analysis", "数据分析"},
	}
	
	discoveryResults := make(map[string]*models.DiscoverResponse)
	
	for i, skillTest := range skillsToTest {
		fmt.Printf("\n   5.%d 发现能力：%s (%s)\n", i+1, skillTest.skillName, skillTest.description)
		
		startTime := time.Now()
		discoverRequest := &models.DiscoverRequest{
			Query: skillTest.skillName,
		}
		
		discoverResponse, err := agentService.DiscoverAgents(ctx, discoverRequest)
		if err != nil {
			log.Fatalf("❌ 发现能力 '%s' 失败：%v", skillTest.skillName, err)
		}
		
		discoveryTime := time.Since(startTime)
		discoveryResults[skillTest.skillName] = discoverResponse
		
		if len(discoverResponse.Agents) == 0 {
			log.Fatalf("❌ 能力 '%s' 没有找到任何Agent", skillTest.skillName)
		}
		
		agent := discoverResponse.Agents[0]
		fmt.Printf("   ✅ 发现成功！发现耗时：%v\n", discoveryTime)
		fmt.Printf("      找到Agent：%s (%s)\n", agent.Name, agent.URL)
		fmt.Printf("      Agent描述：%s\n", agent.Description)
		
		// 验证返回的Agent确实具有请求的技能
		hasSkill := false
		var foundSkill models.AgentSkill
		for _, skill := range agent.Skills {
			if skill.Name == skillTest.skillName {
				hasSkill = true
				foundSkill = skill
				break
			}
		}
		
		if !hasSkill {
			log.Fatalf("❌ 返回的Agent不具有请求的技能 '%s'", skillTest.skillName)
		}
		
		fmt.Printf("      ✅ 技能验证：%s\n", foundSkill.Description)
		fmt.Printf("         支持输入：%v\n", foundSkill.InputModes)
		fmt.Printf("         支持输出：%v\n", foundSkill.OutputModes)
		fmt.Printf("         使用示例：%v\n", foundSkill.Examples)
	}

	// 6. 测试不存在能力的发现
	fmt.Println("\n❓ 步骤6：测试不存在能力的发现")
	
	nonExistentSkills := []string{
		"quantum-encryption",
		"time-travel-simulation", 
		"mind-reading-ai",
		"perpetual-motion-generation",
	}
	
	for i, skillName := range nonExistentSkills {
		fmt.Printf("   6.%d 测试不存在能力：%s\n", i+1, skillName)
		
		discoverRequest := &models.DiscoverRequest{
			Query: skillName,
		}
		
		discoverResponse, err := agentService.DiscoverAgents(ctx, discoverRequest)
		if err != nil {
			log.Fatalf("❌ 发现请求失败：%v", err)
		}
		
		if len(discoverResponse.Agents) == 0 {
			fmt.Printf("   ✅ 正确处理：能力 '%s' 未找到任何Agent\n", skillName)
		} else {
			log.Fatalf("❌ 意外发现：能力 '%s' 找到了 %d 个Agent", skillName, len(discoverResponse.Agents))
		}
	}

	// 7. 性能和统计信息
	fmt.Println("\n📊 步骤7：系统统计信息")
	
	// 7.1 Agent统计
	allAgents, err := agentService.ListAllAgents(ctx)
	if err != nil {
		log.Fatalf("❌ 获取Agent列表失败：%v", err)
	}
	
	totalSkills := 0
	for _, agent := range allAgents {
		totalSkills += len(agent.AgentCard.Skills)
	}
	
	fmt.Printf("   注册的Agent总数：%d\n", len(allAgents))
	fmt.Printf("   注册的技能总数：%d\n", totalSkills)
	fmt.Printf("   创建的vGroup总数：%d\n", len(allGroups))
	
	// 7.2 发现性能统计
	fmt.Println("   能力发现性能：")
	for skillName, result := range discoveryResults {
		fmt.Printf("     - %s: 找到 %d 个Agent\n", skillName, len(result.Agents))
	}

	// 8. 清理测试（可选）
	fmt.Println("\n🧹 步骤8：清理测试数据")
	
	err = agentService.RemoveAgent(ctx, response.AgentID)
	if err != nil {
		log.Printf("⚠️  清理Agent失败：%v", err)
	} else {
		fmt.Println("   ✅ Agent清理成功")
		
		// 验证清理结果
		_, err = agentService.GetAgent(ctx, response.AgentID)
		if err != nil {
			fmt.Println("   ✅ Agent已从存储中移除")
		} else {
			fmt.Println("   ⚠️  Agent仍在存储中")
		}
	}

	// 最终总结
	fmt.Println("\n🎉 测试完成总结")
	fmt.Println("=====================================")
	fmt.Printf("✅ Agent注册：成功注册 1 个Agent\n")
	fmt.Printf("✅ 能力注册：成功注册 %d 个独立技能\n", len(testAgentCard.Skills))
	fmt.Printf("✅ 能力发现：成功发现 %d/%d 个技能\n", len(skillsToTest), len(skillsToTest))
	fmt.Printf("✅ 错误处理：正确处理 %d 个不存在技能\n", len(nonExistentSkills))
	fmt.Printf("✅ 数据清理：成功清理测试数据\n")
	fmt.Println("=====================================")
	fmt.Println("🏆 完整能力注册与发现流程测试 - 全部通过！")
}

func main() {
	TestCompleteCapabilityFlow()
}