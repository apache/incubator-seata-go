package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"agenthub/pkg/client"
)

func main() {
	// Create client
	c := client.NewClient(client.ClientConfig{
		BaseURL: "http://localhost:8080",
		Timeout: 30 * time.Second,
	})

	ctx := context.Background()

	// Register multiple agents with different skills
	fmt.Println("=== Setting up Test Environment ===")

	// Register text analysis agent
	textAgent := &client.RegisterRequest{
		AgentCard: client.AgentCard{
			Name:        "text-analyzer",
			Description: "专业文本分析代理",
			URL:         "http://localhost:8081",
			Version:     "1.0.0",
			Capabilities: client.AgentCapabilities{
				Streaming: true,
			},
			DefaultInputModes:  []string{"text"},
			DefaultOutputModes: []string{"json"},
			Skills: []client.AgentSkill{
				{
					ID:          "sentiment_analysis",
					Name:        "情感分析",
					Description: "分析文本的情感倾向",
					Tags:        []string{"nlp", "sentiment", "emotion"},
					Examples:    []string{"分析评论情感", "判断文本正负面"},
				},
				{
					ID:          "keyword_extraction",
					Name:        "关键词提取",
					Description: "从文本中提取关键词",
					Tags:        []string{"nlp", "keywords", "extraction"},
					Examples:    []string{"提取文档关键词", "分析主题"},
				},
			},
		},
		Host: "localhost",
		Port: 8081,
	}

	_, err := c.RegisterAgent(ctx, textAgent)
	if err != nil {
		log.Printf("Failed to register text agent: %v", err)
	} else {
		fmt.Println("✅ Text analyzer agent registered")
	}

	// Register data analysis agent
	dataAgent := &client.RegisterRequest{
		AgentCard: client.AgentCard{
			Name:        "data-analyzer",
			Description: "专业数据分析代理",
			URL:         "http://localhost:8082",
			Version:     "1.0.0",
			Capabilities: client.AgentCapabilities{
				Streaming: false,
			},
			DefaultInputModes:  []string{"json", "csv"},
			DefaultOutputModes: []string{"json", "image"},
			Skills: []client.AgentSkill{
				{
					ID:          "data_visualization",
					Name:        "数据可视化",
					Description: "生成各种类型的数据图表",
					Tags:        []string{"data", "chart", "visualization"},
					Examples:    []string{"生成销售图表", "制作趋势分析"},
				},
				{
					ID:          "statistical_analysis",
					Name:        "统计分析",
					Description: "对数据进行统计分析",
					Tags:        []string{"statistics", "analysis", "data"},
					Examples:    []string{"计算相关性", "趋势分析"},
				},
			},
		},
		Host: "localhost",
		Port: 8082,
	}

	_, err = c.RegisterAgent(ctx, dataAgent)
	if err != nil {
		log.Printf("Failed to register data agent: %v", err)
	} else {
		fmt.Println("✅ Data analyzer agent registered")
	}

	// Register image processing agent
	imageAgent := &client.RegisterRequest{
		AgentCard: client.AgentCard{
			Name:        "image-processor",
			Description: "专业图像处理代理",
			URL:         "http://localhost:8083",
			Version:     "1.0.0",
			Capabilities: client.AgentCapabilities{
				Streaming: false,
			},
			DefaultInputModes:  []string{"image"},
			DefaultOutputModes: []string{"json", "text"},
			Skills: []client.AgentSkill{
				{
					ID:          "object_detection",
					Name:        "物体识别",
					Description: "识别图像中的物体",
					Tags:        []string{"cv", "detection", "vision"},
					Examples:    []string{"识别照片中的物体", "检测产品类型"},
				},
				{
					ID:          "ocr",
					Name:        "文字识别",
					Description: "从图像中提取文字",
					Tags:        []string{"ocr", "text", "extraction"},
					Examples:    []string{"识别文档文字", "提取图片文本"},
				},
			},
		},
		Host: "localhost",
		Port: 8083,
	}

	_, err = c.RegisterAgent(ctx, imageAgent)
	if err != nil {
		log.Printf("Failed to register image agent: %v", err)
	} else {
		fmt.Println("✅ Image processor agent registered")
	}

	fmt.Println()

	// Test various context analysis scenarios
	testScenarios := []struct {
		name        string
		description string
		context     string
		expected    string
	}{
		{
			name:        "文本情感分析",
			description: "我需要分析用户评论的情感倾向，判断是正面还是负面",
			context:     "电商平台用户反馈分析",
			expected:    "sentiment_analysis",
		},
		{
			name:        "关键词提取",
			description: "需要从用户反馈中提取关键词，了解用户关注的重点",
			context:     "产品改进建议收集",
			expected:    "keyword_extraction",
		},
		{
			name:        "数据可视化",
			description: "需要将销售数据生成图表，便于管理层查看",
			context:     "月度销售报告制作",
			expected:    "data_visualization",
		},
		{
			name:        "统计分析",
			description: "需要对用户行为数据进行统计分析，找出规律",
			context:     "用户行为研究",
			expected:    "statistical_analysis",
		},
		{
			name:        "图像物体识别",
			description: "需要识别上传图片中的物体类型",
			context:     "商品自动分类系统",
			expected:    "object_detection",
		},
		{
			name:        "图片文字识别",
			description: "需要从扫描的文档图片中提取文字内容",
			context:     "文档数字化处理",
			expected:    "ocr",
		},
		{
			name:        "复合需求",
			description: "需要先识别图片中的文字，然后分析文字的情感倾向",
			context:     "社交媒体图片内容分析",
			expected:    "ocr,sentiment_analysis",
		},
		{
			name:        "无匹配需求",
			description: "需要训练深度学习模型进行语音识别",
			context:     "AI语音助手开发",
			expected:    "no_match",
		},
	}

	fmt.Println("=== 动态上下文分析测试 ===")
	for i, scenario := range testScenarios {
		fmt.Printf("%d. 测试场景: %s\n", i+1, scenario.name)
		fmt.Printf("   需求描述: %s\n", scenario.description)
		fmt.Printf("   应用场景: %s\n", scenario.context)

		contextReq := &client.ContextAnalysisRequest{
			NeedDescription: scenario.description,
			UserContext:     scenario.context,
		}

		contextResp, err := c.AnalyzeContext(ctx, contextReq)
		if err != nil {
			fmt.Printf("   ❌ 分析失败: %v\n", err)
		} else {
			if contextResp.Success {
				fmt.Printf("   ✅ 分析成功: %s\n", contextResp.Message)
				if len(contextResp.MatchedSkills) > 0 {
					fmt.Printf("   📊 匹配技能: %d个\n", len(contextResp.MatchedSkills))
					for j, skill := range contextResp.MatchedSkills {
						if j < 3 { // 只显示前3个匹配结果
							fmt.Printf("      - %s (%s)\n", skill.Name, skill.ID)
						}
					}
					if len(contextResp.MatchedSkills) > 3 {
						fmt.Printf("      ... 和其他 %d 个技能\n", len(contextResp.MatchedSkills)-3)
					}
				}
				if contextResp.RouteResult != nil {
					fmt.Printf("   🎯 推荐路由: %s\n", contextResp.RouteResult.AgentURL)
					fmt.Printf("   🔧 目标技能: %s\n", contextResp.RouteResult.SkillName)
				}
				if contextResp.AnalysisResult != nil && contextResp.AnalysisResult.SuggestedWorkflow != "" {
					fmt.Printf("   💡 工作流建议: %s\n", contextResp.AnalysisResult.SuggestedWorkflow)
				}
			} else {
				fmt.Printf("   ⚠️  分析结果: %s\n", contextResp.Message)
			}
		}
		fmt.Println()

		// Add a small delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	// Performance test
	fmt.Println("=== 性能测试 ===")
	start := time.Now()
	concurrent := 5
	done := make(chan bool, concurrent)

	contextReq := &client.ContextAnalysisRequest{
		NeedDescription: "需要分析文本内容的情感倾向",
		UserContext:     "性能测试",
	}

	for i := 0; i < concurrent; i++ {
		go func(id int) {
			_, err := c.AnalyzeContext(ctx, contextReq)
			if err != nil {
				fmt.Printf("   并发请求 %d 失败: %v\n", id+1, err)
			} else {
				fmt.Printf("   ✅ 并发请求 %d 完成\n", id+1)
			}
			done <- true
		}(i)
	}

	for i := 0; i < concurrent; i++ {
		<-done
	}
	elapsed := time.Since(start)
	fmt.Printf("   📈 %d 个并发请求完成，总耗时: %v\n", concurrent, elapsed)
	fmt.Printf("   📈 平均响应时间: %v\n", elapsed/time.Duration(concurrent))

	fmt.Println()

	// Cleanup
	fmt.Println("=== 清理测试环境 ===")
	agents := []string{"text-analyzer", "data-analyzer", "image-processor"}
	for _, agentID := range agents {
		err := c.RemoveAgent(ctx, agentID)
		if err != nil {
			fmt.Printf("   删除代理 %s 失败: %v\n", agentID, err)
		} else {
			fmt.Printf("   ✅ 删除代理 %s 成功\n", agentID)
		}
	}

	fmt.Println("\n🎉 动态上下文分析演示完成!")
}
