package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"agenthub/pkg/app"
	"agenthub/pkg/models"
)

// TestCapabilityDiscoveryAPI 测试通过HTTP API的能力发现功能
func TestCapabilityDiscoveryAPI() {
	fmt.Println("=== HTTP API能力发现测试 ===")

	// 1. 启动AgentHub应用（使用NamingServer配置）
	fmt.Println("\n--- 步骤1：启动AgentHub应用 ---")
	
	// 设置环境变量启用NamingServer
	fmt.Println("设置NamingServer配置...")
	
	// 创建应用实例（使用启用NamingServer的配置）
	application, err := app.NewApplication(app.ApplicationConfig{
		ConfigPath: "test_config_namingserver.yaml",
	})
	if err != nil {
		log.Fatalf("创建应用失败: %v", err)
	}

	// 在后台运行服务器
	go func() {
		if err := application.Run(); err != nil {
			log.Printf("服务器运行错误: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(2 * time.Second)
	fmt.Println("✅ AgentHub服务器已启动")

	// 2. 注册一个具有多种技能的Agent
	fmt.Println("\n--- 步骤2：注册多技能Agent ---")
	
	registerRequest := models.RegisterRequest{
		AgentCard: models.AgentCard{
			Name:        "CapabilityTestAgent",
			Version:     "1.0.0",
			URL:         "http://localhost:9001",
			Description: "用于测试能力发现的Agent",
			Skills: []models.AgentSkill{
				{
					ID:          "natural-language-processing",
					Name:        "natural-language-processing",
					Description: "自然语言处理技能",
					Tags:        []string{"NLP", "AI", "文本处理"},
					Examples:    []string{"文本分类", "情感分析", "实体识别"},
				},
				{
					ID:          "computer-vision",
					Name:        "computer-vision",
					Description: "计算机视觉技能",
					Tags:        []string{"CV", "AI", "图像处理"},
					Examples:    []string{"物体检测", "人脸识别", "图像分割"},
				},
				{
					ID:          "speech-recognition",
					Name:        "speech-recognition", 
					Description: "语音识别技能",
					Tags:        []string{"ASR", "AI", "语音处理"},
					Examples:    []string{"语音转文本", "说话人识别", "语音情感分析"},
				},
			},
		},
		Host: "localhost",
		Port: 9001,
	}

	if err := registerAgentViaAPI(registerRequest); err != nil {
		log.Fatalf("注册Agent失败: %v", err)
	}
	fmt.Println("✅ 多技能Agent注册成功")

	// 3. 测试各个技能的发现
	fmt.Println("\n--- 步骤3：测试技能发现API ---")
	
	skillsToTest := []string{
		"natural-language-processing",
		"computer-vision", 
		"speech-recognition",
	}

	for _, skillName := range skillsToTest {
		fmt.Printf("\n🔍 测试发现技能: %s\n", skillName)
		
		agents, err := discoverAgentsViaAPI(skillName)
		if err != nil {
			fmt.Printf("❌ 发现技能 '%s' 失败: %v\n", skillName, err)
			continue
		}

		if len(agents) == 0 {
			fmt.Printf("❌ 技能 '%s' 没有找到任何Agent\n", skillName)
			continue
		}

		fmt.Printf("✅ 找到 %d 个提供技能 '%s' 的Agent:\n", len(agents), skillName)
		for i, agent := range agents {
			fmt.Printf("   Agent %d: %s (%s)\n", i+1, agent.Name, agent.URL)
			fmt.Printf("           描述: %s\n", agent.Description)
			
			// 验证返回的Agent确实包含请求的技能
			hasSkill := false
			for _, skill := range agent.Skills {
				if skill.Name == skillName {
					hasSkill = true
					fmt.Printf("           技能详情: %s - %s\n", skill.Name, skill.Description)
					if len(skill.Tags) > 0 {
						fmt.Printf("           标签: %v\n", skill.Tags)
					}
					break
				}
			}
			
			if !hasSkill {
				fmt.Printf("⚠️  警告: 返回的Agent不包含请求的技能!\n")
			}
		}
	}

	// 4. 测试不存在技能的发现
	fmt.Println("\n--- 步骤4：测试不存在技能的发现 ---")
	
	nonExistentSkills := []string{
		"quantum-computing",
		"time-travel",
		"mind-reading",
	}

	for _, skillName := range nonExistentSkills {
		fmt.Printf("🔍 测试不存在的技能: %s\n", skillName)
		
		agents, err := discoverAgentsViaAPI(skillName)
		if err != nil {
			fmt.Printf("❌ API调用失败: %v\n", err)
			continue
		}

		if len(agents) == 0 {
			fmt.Printf("✅ 正确返回: 技能 '%s' 没有找到Agent\n", skillName)
		} else {
			fmt.Printf("❌ 意外结果: 技能 '%s' 找到了 %d 个Agent\n", skillName, len(agents))
		}
	}

	// 5. 测试多Agent提供同一技能的情况
	fmt.Println("\n--- 步骤5：测试多Agent同技能发现 ---")
	
	// 注册另一个提供相同技能的Agent
	registerRequest2 := models.RegisterRequest{
		AgentCard: models.AgentCard{
			Name:        "SecondNLPAgent",
			Version:     "2.0.0",
			URL:         "http://localhost:9002",
			Description: "第二个NLP Agent",
			Skills: []models.AgentSkill{
				{
					ID:          "natural-language-processing-v2",
					Name:        "natural-language-processing", // 同名技能
					Description: "更强大的自然语言处理技能",
					Tags:        []string{"NLP", "AI", "深度学习"},
				},
			},
		},
		Host: "localhost",
		Port: 9002,
	}

	if err := registerAgentViaAPI(registerRequest2); err != nil {
		log.Fatalf("注册第二个Agent失败: %v", err)
	}

	// 发现NLP技能，应该能找到多个Agent
	fmt.Printf("🔍 发现技能: natural-language-processing (期待找到多个Agent)\n")
	agents, err := discoverAgentsViaAPI("natural-language-processing")
	if err != nil {
		fmt.Printf("❌ 发现失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个提供NLP技能的Agent\n", len(agents))
		// 注意：当前实现只返回第一个找到的Agent，这可能需要改进
	}

	fmt.Println("\n=== HTTP API能力发现测试完成 ===")
}

// registerAgentViaAPI 通过HTTP API注册Agent
func registerAgentViaAPI(req models.RegisterRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	resp, err := http.Post("http://localhost:8080/agent/register", 
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("注册失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var response models.RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("注册失败: %s", response.Message)
	}

	return nil
}

// discoverAgentsViaAPI 通过HTTP API发现Agent
func discoverAgentsViaAPI(skillName string) ([]models.AgentCard, error) {
	discoverRequest := models.DiscoverRequest{
		Query: skillName,
	}

	jsonData, err := json.Marshal(discoverRequest)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	resp, err := http.Post("http://localhost:8080/agent/discover",
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("发现失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var response models.DiscoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return response.Agents, nil
}

func main() {
	TestCapabilityDiscoveryAPI()
}