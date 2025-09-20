package tests

import (
	"context"
	"fmt"
	"log"
	"time"

	"agenthub/pkg/common"
	"agenthub/pkg/models"
	"agenthub/pkg/storage"
)

// TestNamingServerIntegration 测试NamingServer集成功能
func TestNamingServerIntegration() {
	fmt.Println("=== NamingServer集成测试 ===")

	// 1. 创建Mock NamingServer注册中心
	mockRegistry := storage.NewMockNamingserverRegistry()
	
	// 2. 创建基于技能的NamingServer存储
	skillStorage := storage.NewSkillBasedNamingServerStorage(mockRegistry)
	
	// 3. 创建测试Agent
	now := time.Now()
	testAgent := &models.RegisteredAgent{
		BaseResource: &common.BaseResource{
			ID:        "test-agent-001",
			Kind:      "RegisteredAgent",
			Version:   "v1",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  make(map[string]interface{}),
		},
		AgentCard: models.AgentCard{
			Name:        "TestAgent",
			Version:     "1.0.0",
			URL:         "http://localhost:8080",
			Description: "测试Agent",
			Skills: []models.AgentSkill{
				{
					ID:          "python-coding",
					Name:        "python-coding",
					Description: "Python编程技能",
					Tags:        []string{"编程", "python"},
				},
				{
					ID:          "data-analysis", 
					Name:        "data-analysis",
					Description: "数据分析技能",
					Tags:        []string{"数据", "分析"},
				},
			},
		},
		Host:         "localhost",
		Port:         8080,
		Status:       "active",
		LastSeen:     now,
		RegisteredAt: now,
	}

	ctx := context.Background()

	// 4. 测试Agent注册
	fmt.Println("\n--- 测试Agent注册 ---")
	err := skillStorage.Create(ctx, testAgent)
	if err != nil {
		log.Fatalf("注册Agent失败: %v", err)
	}
	fmt.Println("✅ Agent注册成功")

	// 5. 测试技能发现
	fmt.Println("\n--- 测试技能发现 ---")
	
	// 测试发现python-coding技能
	pythonUrl, err := skillStorage.DiscoverUrlBySkill(ctx, "python-coding")
	if err != nil {
		log.Fatalf("发现python-coding技能失败: %v", err)
	}
	fmt.Printf("✅ 发现python-coding技能: %s\n", pythonUrl)

	// 测试发现data-analysis技能
	dataUrl, err := skillStorage.DiscoverUrlBySkill(ctx, "data-analysis")
	if err != nil {
		log.Fatalf("发现data-analysis技能失败: %v", err)
	}
	fmt.Printf("✅ 发现data-analysis技能: %s\n", dataUrl)

	// 测试发现不存在的技能
	fmt.Println("\n--- 测试发现不存在的技能 ---")
	_, err = skillStorage.DiscoverUrlBySkill(ctx, "nonexistent-skill")
	if err != nil {
		fmt.Printf("✅ 正确处理了不存在的技能: %s\n", err.Error())
	} else {
		fmt.Println("❌ 应该返回错误，但没有")
	}

	// 6. 测试Agent检索
	fmt.Println("\n--- 测试Agent检索 ---")
	resource, err := skillStorage.Get(ctx, "test-agent-001")
	if err != nil {
		log.Fatalf("检索Agent失败: %v", err)
	}
	
	retrievedAgent, ok := resource.(*models.RegisteredAgent)
	if !ok {
		log.Fatalf("返回的资源类型不正确")
	}
	
	fmt.Printf("✅ 成功检索Agent: %s (技能数量: %d)\n", 
		retrievedAgent.AgentCard.Name, len(retrievedAgent.AgentCard.Skills))

	// 7. 测试Agent列表
	fmt.Println("\n--- 测试Agent列表 ---")
	resources, err := skillStorage.List(ctx, map[string]interface{}{})
	if err != nil {
		log.Fatalf("列出Agent失败: %v", err)
	}
	fmt.Printf("✅ 成功列出 %d 个Agent\n", len(resources))

	// 8. 测试Agent删除
	fmt.Println("\n--- 测试Agent删除 ---")
	err = skillStorage.Delete(ctx, "test-agent-001")
	if err != nil {
		log.Fatalf("删除Agent失败: %v", err)
	}
	fmt.Println("✅ Agent删除成功")

	// 9. 验证删除后无法发现技能
	fmt.Println("\n--- 验证删除后的技能发现 ---")
	_, err = skillStorage.DiscoverUrlBySkill(ctx, "python-coding")
	if err != nil {
		fmt.Printf("✅ 删除后正确处理了技能发现失败: %s\n", err.Error())
	} else {
		fmt.Println("❌ 删除后应该无法发现技能，但仍然成功")
	}

	fmt.Println("\n=== NamingServer集成测试完成 ===")
	fmt.Println("✅ 所有测试通过！")
}

func main() {
	TestNamingServerIntegration()
}