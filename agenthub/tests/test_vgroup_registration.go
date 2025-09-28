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

// TestVGroupRegistration 测试独立vGroup的能力注册
func TestVGroupRegistration() {
	fmt.Println("=== 独立vGroup能力注册测试 ===")

	// 1. 创建Mock NamingServer注册中心
	mockRegistry := storage.NewMockNamingserverRegistry()

	// 2. 创建基于技能的NamingServer存储
	skillStorage := storage.NewSkillBasedNamingServerStorage(mockRegistry)

	// 3. 创建测试Agent（多个技能）
	now := time.Now()
	testAgent := &models.RegisteredAgent{
		BaseResource: &common.BaseResource{
			ID:        "multi-skill-agent",
			Kind:      "RegisteredAgent",
			Version:   "v1",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  make(map[string]interface{}),
		},
		AgentCard: models.AgentCard{
			Name:        "MultiSkillAgent",
			Version:     "1.0.0",
			URL:         "http://localhost:9000",
			Description: "具有多个技能的测试Agent",
			Skills: []models.AgentSkill{
				{
					ID:          "web-scraping",
					Name:        "web-scraping",
					Description: "网页数据抓取技能",
					Tags:        []string{"爬虫", "数据"},
				},
				{
					ID:          "image-processing",
					Name:        "image-processing",
					Description: "图像处理技能",
					Tags:        []string{"图像", "AI"},
				},
				{
					ID:          "text-translation",
					Name:        "text-translation",
					Description: "文本翻译技能",
					Tags:        []string{"翻译", "NLP"},
				},
			},
		},
		Host:         "localhost",
		Port:         9000,
		Status:       "active",
		LastSeen:     now,
		RegisteredAt: now,
	}

	ctx := context.Background()

	// 4. 注册Agent（应该创建3个独立的vGroup）
	fmt.Println("\n--- 步骤1：注册多技能Agent ---")
	err := skillStorage.Create(ctx, testAgent)
	if err != nil {
		log.Fatalf("注册Agent失败: %v", err)
	}
	fmt.Println("✅ 多技能Agent注册成功")

	// 5. 验证每个技能都被注册到独立的vGroup
	fmt.Println("\n--- 步骤2：验证独立vGroup注册 ---")

	expectedVGroups := []string{
		"skill-web-scraping",
		"skill-image-processing",
		"skill-text-translation",
	}

	allGroups := mockRegistry.GetAllGroups()
	fmt.Printf("发现的vGroup: %v\n", allGroups)

	// 检查每个预期的vGroup是否存在
	for _, expectedVGroup := range expectedVGroups {
		found := false
		for _, actualVGroup := range allGroups {
			if actualVGroup == expectedVGroup {
				found = true
				break
			}
		}

		if found {
			fmt.Printf("✅ vGroup '%s' 存在\n", expectedVGroup)
		} else {
			fmt.Printf("❌ vGroup '%s' 不存在\n", expectedVGroup)
		}
	}

	// 6. 验证每个vGroup中都有对应的服务实例
	fmt.Println("\n--- 步骤3：验证vGroup中的服务实例 ---")
	for _, vGroup := range expectedVGroups {
		instances, err := mockRegistry.Lookup(vGroup)
		if err != nil {
			fmt.Printf("❌ 查询vGroup '%s' 失败: %v\n", vGroup, err)
			continue
		}

		if len(instances) > 0 {
			instance := instances[0]
			fmt.Printf("✅ vGroup '%s' 包含服务实例: %s:%d\n",
				vGroup, instance.Addr, instance.Port)
		} else {
			fmt.Printf("❌ vGroup '%s' 中没有服务实例\n", vGroup)
		}
	}

	// 7. 测试技能发现是否能找到正确的URL
	fmt.Println("\n--- 步骤4：测试技能发现 ---")
	for _, skill := range testAgent.AgentCard.Skills {
		url, err := skillStorage.DiscoverUrlBySkill(ctx, skill.Name)
		if err != nil {
			fmt.Printf("❌ 发现技能 '%s' 失败: %v\n", skill.Name, err)
		} else {
			fmt.Printf("✅ 技能 '%s' -> %s\n", skill.Name, url)
		}
	}

	// 8. 创建第二个Agent测试vGroup隔离
	fmt.Println("\n--- 步骤5：测试vGroup隔离 ---")

	testAgent2 := &models.RegisteredAgent{
		BaseResource: &common.BaseResource{
			ID:        "another-agent",
			Kind:      "RegisteredAgent",
			Version:   "v1",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  make(map[string]interface{}),
		},
		AgentCard: models.AgentCard{
			Name:        "AnotherAgent",
			Version:     "2.0.0",
			URL:         "http://localhost:8888",
			Description: "另一个具有相同技能名称的Agent",
			Skills: []models.AgentSkill{
				{
					ID:          "web-scraping-v2",
					Name:        "web-scraping", // 同名技能
					Description: "更先进的网页数据抓取技能",
					Tags:        []string{"爬虫", "AI"},
				},
			},
		},
		Host:         "localhost",
		Port:         8888,
		Status:       "active",
		LastSeen:     now,
		RegisteredAt: now,
	}

	err = skillStorage.Create(ctx, testAgent2)
	if err != nil {
		log.Fatalf("注册第二个Agent失败: %v", err)
	}

	// 验证同名技能会覆盖还是共存
	instances, err := mockRegistry.Lookup("skill-web-scraping")
	if err != nil {
		fmt.Printf("❌ 查询skill-web-scraping vGroup失败: %v\n", err)
	} else {
		fmt.Printf("vGroup 'skill-web-scraping' 中的实例数: %d\n", len(instances))
		for i, instance := range instances {
			fmt.Printf("  实例%d: %s:%d\n", i+1, instance.Addr, instance.Port)
		}
	}

	fmt.Println("\n=== 独立vGroup能力注册测试完成 ===")
}

func main() {
	TestVGroupRegistration()
}
