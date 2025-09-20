package tests

import (
	"context"
	"fmt"
	"log"
	"time"

	"agenthub/pkg/common"
	"agenthub/pkg/models"
	"agenthub/pkg/services"
	"agenthub/pkg/storage"
)

// TestCapabilityDiscoveryDirect 直接测试能力发现功能（不通过HTTP）
func TestCapabilityDiscoveryDirect() {
	fmt.Println("=== 直接能力发现功能测试 ===")

	// 1. 创建NamingServer存储和Agent服务
	fmt.Println("\n--- 步骤1：初始化存储和服务 ---")
	
	mockRegistry := storage.NewMockNamingserverRegistry()
	skillStorage := storage.NewSkillBasedNamingServerStorage(mockRegistry)
	
	agentService := services.NewAgentService(services.AgentServiceConfig{
		Storage: skillStorage,
	})
	
	fmt.Println("✅ 存储和服务初始化完成")

	// 2. 注册多个Agent，每个具有不同的技能组合
	fmt.Println("\n--- 步骤2：注册多个Agent ---")
	
	agents := []*models.RegisteredAgent{
		createTestAgent("agent-nlp", "http://localhost:9001", []models.AgentSkill{
			{ID: "nlp-1", Name: "text-analysis", Description: "文本分析", Tags: []string{"NLP"}},
			{ID: "nlp-2", Name: "sentiment-analysis", Description: "情感分析", Tags: []string{"NLP", "AI"}},
		}),
		createTestAgent("agent-cv", "http://localhost:9002", []models.AgentSkill{
			{ID: "cv-1", Name: "image-classification", Description: "图像分类", Tags: []string{"CV", "AI"}},
			{ID: "cv-2", Name: "object-detection", Description: "物体检测", Tags: []string{"CV", "AI"}},
		}),
		createTestAgent("agent-multimodal", "http://localhost:9003", []models.AgentSkill{
			{ID: "mm-1", Name: "text-analysis", Description: "高级文本分析", Tags: []string{"NLP", "advanced"}}, // 与第一个Agent重复技能
			{ID: "mm-2", Name: "speech-to-text", Description: "语音转文本", Tags: []string{"ASR", "AI"}},
			{ID: "mm-3", Name: "text-to-speech", Description: "文本转语音", Tags: []string{"TTS", "AI"}},
		}),
	}

	ctx := context.Background()
	
	// 注册所有Agent
	for i, agent := range agents {
		response, err := agentService.RegisterAgent(ctx, &models.RegisterRequest{
			AgentCard: agent.AgentCard,
			Host:      agent.Host,
			Port:      agent.Port,
		})
		if err != nil {
			log.Fatalf("注册Agent %d失败: %v", i+1, err)
		}
		if !response.Success {
			log.Fatalf("注册Agent %d失败: %s", i+1, response.Message)
		}
		fmt.Printf("✅ Agent %s 注册成功\n", agent.AgentCard.Name)
	}

	// 3. 测试单个技能发现
	fmt.Println("\n--- 步骤3：测试单个技能发现 ---")
	
	singleSkillTests := []string{
		"sentiment-analysis",    // 只有agent-nlp有
		"object-detection",      // 只有agent-cv有
		"speech-to-text",        // 只有agent-multimodal有
	}

	for _, skillName := range singleSkillTests {
		fmt.Printf("\n🔍 发现技能: %s\n", skillName)
		
		response, err := agentService.DiscoverAgents(ctx, &models.DiscoverRequest{
			Query: skillName,
		})
		if err != nil {
			fmt.Printf("❌ 发现失败: %v\n", err)
			continue
		}

		if len(response.Agents) == 0 {
			fmt.Printf("❌ 没有找到提供技能 '%s' 的Agent\n", skillName)
		} else {
			fmt.Printf("✅ 找到 %d 个Agent提供技能 '%s':\n", len(response.Agents), skillName)
			for i, agent := range response.Agents {
				fmt.Printf("   %d. %s (%s)\n", i+1, agent.Name, agent.URL)
			}
		}
	}

	// 4. 测试重复技能发现（多个Agent提供同一技能）
	fmt.Println("\n--- 步骤4：测试重复技能发现 ---")
	
	fmt.Printf("🔍 发现重复技能: text-analysis\n")
	response, err := agentService.DiscoverAgents(ctx, &models.DiscoverRequest{
		Query: "text-analysis",
	})
	if err != nil {
		fmt.Printf("❌ 发现失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个Agent提供 'text-analysis' 技能:\n", len(response.Agents))
		for i, agent := range response.Agents {
			fmt.Printf("   %d. %s (%s) - %s\n", i+1, agent.Name, agent.URL, agent.Description)
		}
		
		// 检查NamingServer中的实际实例数
		instances, err := mockRegistry.Lookup("skill-text-analysis")
		if err != nil {
			fmt.Printf("   NamingServer查询失败: %v\n", err)
		} else {
			fmt.Printf("   NamingServer中 'skill-text-analysis' 的实例数: %d\n", len(instances))
			for i, instance := range instances {
				fmt.Printf("     实例%d: %s:%d\n", i+1, instance.Addr, instance.Port)
			}
		}
	}

	// 5. 测试不存在技能的发现
	fmt.Println("\n--- 步骤5：测试不存在技能的发现 ---")
	
	nonExistentSkills := []string{
		"quantum-computing",
		"telepathy",
		"time-manipulation",
	}

	for _, skillName := range nonExistentSkills {
		fmt.Printf("🔍 测试不存在技能: %s\n", skillName)
		
		response, err := agentService.DiscoverAgents(ctx, &models.DiscoverRequest{
			Query: skillName,
		})
		if err != nil {
			fmt.Printf("❌ API调用失败: %v\n", err)
			continue
		}

		if len(response.Agents) == 0 {
			fmt.Printf("✅ 正确处理: 技能 '%s' 没有找到Agent\n", skillName)
		} else {
			fmt.Printf("❌ 意外结果: 技能 '%s' 找到了 %d 个Agent\n", skillName, len(response.Agents))
		}
	}

	// 6. 验证技能到Agent的完整映射
	fmt.Println("\n--- 步骤6：验证技能映射完整性 ---")
	
	fmt.Println("所有已注册的vGroup:")
	allGroups := mockRegistry.GetAllGroups()
	for i, group := range allGroups {
		fmt.Printf("%d. %s\n", i+1, group)
	}
	
	fmt.Printf("总计: %d 个独立的技能vGroup\n", len(allGroups))

	// 7. 测试AgentService的其他功能
	fmt.Println("\n--- 步骤7：测试Agent管理功能 ---")
	
	// 列出所有Agent
	allAgents, err := agentService.ListAllAgents(ctx)
	if err != nil {
		fmt.Printf("❌ 列出所有Agent失败: %v\n", err)
	} else {
		fmt.Printf("✅ 总共注册了 %d 个Agent\n", len(allAgents))
		for i, agent := range allAgents {
			fmt.Printf("   %d. %s - %d个技能\n", i+1, agent.AgentCard.Name, len(agent.AgentCard.Skills))
		}
	}

	fmt.Println("\n=== 直接能力发现功能测试完成 ===")
}

// createTestAgent 创建测试Agent
func createTestAgent(name, url string, skills []models.AgentSkill) *models.RegisteredAgent {
	now := time.Now()
	return &models.RegisteredAgent{
		BaseResource: &common.BaseResource{
			ID:        name,
			Kind:      "RegisteredAgent",
			Version:   "v1",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  make(map[string]interface{}),
		},
		AgentCard: models.AgentCard{
			Name:        name,
			Version:     "1.0.0",
			URL:         url,
			Description: fmt.Sprintf("测试Agent: %s", name),
			Skills:      skills,
		},
		Host:         "localhost",
		Port:         9001, // 简化，使用固定端口
		Status:       "active",
		LastSeen:     now,
		RegisteredAt: now,
	}
}

func main() {
	TestCapabilityDiscoveryDirect()
}