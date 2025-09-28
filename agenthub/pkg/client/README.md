<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

# AgentHub Go Client SDK

AgentHub Go Client SDK 是用于与 AgentHub 服务进行交互的官方 Go 语言客户端库。

## 功能特性

- ✅ **完整的API覆盖**: 支持所有AgentHub服务端接口
- 🔐 **认证支持**: 内置JWT认证机制
- 🎯 **动态上下文分析**: 支持AI驱动的智能代理路由
- 📊 **类型安全**: 完整的类型定义和验证
- 🚀 **并发安全**: 支持并发请求
- ⚡ **高性能**: 优化的HTTP客户端实现
- 🛡️ **错误处理**: 详细的错误信息和状态码处理

## 快速开始

### 安装

```bash
go get agenthub/pkg/client
```

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "agenthub/pkg/client"
)

func main() {
    // 创建客户端
    c := client.NewClient(client.ClientConfig{
        BaseURL: "http://localhost:8080",
        Timeout: 30 * time.Second,
    })

    ctx := context.Background()

    // 健康检查
    health, err := c.HealthCheck(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("服务状态: %s\n", health.Status)

    // 注册代理
    req := &client.RegisterRequest{
        AgentCard: client.AgentCard{
            Name:        "my-agent",
            Description: "我的测试代理",
            URL:         "http://localhost:3001",
            Version:     "1.0.0",
            Capabilities: client.AgentCapabilities{
                Streaming: true,
            },
            DefaultInputModes:  []string{"text"},
            DefaultOutputModes: []string{"text"},
            Skills: []client.AgentSkill{
                {
                    ID:          "text-processing",
                    Name:        "文本处理",
                    Description: "处理各种文本内容",
                    Tags:        []string{"nlp", "text"},
                },
            },
        },
        Host: "localhost",
        Port: 3001,
    }

    resp, err := c.RegisterAgent(ctx, req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("代理注册成功: %s\n", resp.AgentID)
}
```

## API 接口

### 代理管理

#### 注册代理

```go
req := &client.RegisterRequest{
    AgentCard: client.AgentCard{
        Name:        "example-agent",
        Description: "示例代理",
        URL:         "http://localhost:3001",
        Version:     "1.0.0",
        // ... 其他字段
    },
    Host: "localhost",
    Port: 3001,
}

resp, err := client.RegisterAgent(ctx, req)
```

#### 发现代理

```go
req := &client.DiscoverRequest{
    Query: "文本处理",
}

resp, err := client.DiscoverAgents(ctx, req)
```

#### 获取代理详情

```go
agent, err := client.GetAgent(ctx, "agent-id")
```

#### 列出所有代理

```go
agents, err := client.ListAgents(ctx)
```

#### 更新代理状态

```go
err := client.UpdateAgentStatus(ctx, "agent-id", "inactive")
```

#### 删除代理

```go
err := client.RemoveAgent(ctx, "agent-id")
```

#### 发送心跳

```go
err := client.SendHeartbeat(ctx, "agent-id")
```

### 动态上下文分析 ⭐

动态上下文分析是 AgentHub 的核心功能，它使用 AI 来分析用户需求并自动路由到最适合的代理。

```go
req := &client.ContextAnalysisRequest{
    NeedDescription: "我需要分析用户评论的情感倾向",
    UserContext:     "电商平台用户反馈分析",
}

resp, err := client.AnalyzeContext(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("分析结果: %s\n", resp.Message)
if len(resp.MatchedSkills) > 0 {
    fmt.Printf("匹配的技能: %s\n", resp.MatchedSkills[0].Name)
}

if resp.RouteResult != nil {
    fmt.Printf("推荐路由: %s\n", resp.RouteResult.AgentURL)
    fmt.Printf("目标技能: %s\n", resp.RouteResult.SkillName)
}
```

### 认证

#### 获取认证令牌

```go
tokenResp, err := client.GetAuthToken(ctx, "username")
if err != nil {
    log.Fatal(err)
}

// 设置认证令牌
client.SetAuthToken(tokenResp.Token)
```

#### 刷新令牌

```go
newTokenResp, err := client.RefreshAuthToken(ctx, refreshToken)
```

### 系统接口

#### 健康检查

```go
health, err := client.HealthCheck(ctx)
```

#### 获取系统指标

```go
metrics, err := client.GetMetrics(ctx)
```

## 配置选项

### 客户端配置

```go
config := client.ClientConfig{
    BaseURL:   "http://localhost:8080",  // AgentHub服务地址
    Timeout:   30 * time.Second,         // 请求超时时间
    AuthToken: "your-jwt-token",         // 认证令牌（可选）
}

c := client.NewClient(config)
```

### 设置认证令牌

```go
c.SetAuthToken("your-jwt-token")
```

## 错误处理

SDK 提供了详细的错误处理机制：

```go
resp, err := client.RegisterAgent(ctx, req)
if err != nil {
    // 检查是否是 API 错误
    if apiErr, ok := err.(*client.APIError); ok {
        fmt.Printf("API错误: %d - %s (代码: %d)\n", 
            apiErr.StatusCode, apiErr.Message, apiErr.Code)
    } else {
        fmt.Printf("网络错误: %v\n", err)
    }
    return
}
```

## 类型定义

### 代理卡片 (AgentCard)

```go
type AgentCard struct {
    Name                 string            `json:"name"`
    Description          string            `json:"description"`
    URL                  string            `json:"url"`
    Version              string            `json:"version"`
    Capabilities         AgentCapabilities `json:"capabilities"`
    DefaultInputModes    []string          `json:"defaultInputModes"`
    DefaultOutputModes   []string          `json:"defaultOutputModes"`
    Skills               []AgentSkill      `json:"skills"`
}
```

### 代理技能 (AgentSkill)

```go
type AgentSkill struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Tags        []string `json:"tags"`
    Examples    []string `json:"examples,omitempty"`
}
```

### 已注册代理 (RegisteredAgent)

```go
type RegisteredAgent struct {
    ID           string    `json:"id"`
    Status       string    `json:"status"`
    AgentCard    AgentCard `json:"agent_card"`
    Host         string    `json:"host"`
    Port         int       `json:"port"`
    LastSeen     time.Time `json:"last_seen"`
    RegisteredAt time.Time `json:"registered_at"`
}
```

## 示例程序

### 基本使用示例

```bash
go run internal/examples/client/basic_usage.go
```

### 动态上下文分析演示

```bash
go run internal/examples/client/context_analysis_demo.go
```

### 高级功能演示

```bash
go run internal/examples/client/advanced_usage.go
```

## 项目结构

```
pkg/client/
├── client.go           # 核心客户端实现
├── types.go            # 类型定义
└── README.md           # 说明文档

internal/examples/client/
├── basic_usage.go           # 基本使用示例
├── context_analysis_demo.go # 上下文分析演示
└── advanced_usage.go        # 高级功能演示
```

## 最佳实践

### 1. 上下文管理

始终使用 `context.Context` 来控制请求的生命周期：

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := client.RegisterAgent(ctx, req)
```

### 2. 错误处理

区分不同类型的错误：

```go
if err != nil {
    if apiErr, ok := err.(*client.APIError); ok {
        // API 错误处理
        switch apiErr.StatusCode {
        case 400:
            fmt.Println("请求参数错误")
        case 401:
            fmt.Println("认证失败")
        case 500:
            fmt.Println("服务器内部错误")
        }
    } else {
        // 网络错误处理
        fmt.Printf("网络错误: %v\n", err)
    }
}
```

### 3. 代理状态检查

使用辅助方法检查代理状态：

```go
agent, err := client.GetAgent(ctx, "agent-id")
if err != nil {
    log.Fatal(err)
}

if agent.IsActive() {
    fmt.Println("代理处于活跃状态")
}

if agent.HasSkill("text-processing") {
    fmt.Println("代理支持文本处理")
}
```

### 4. 并发安全

客户端是并发安全的，可以在多个 goroutine 中使用：

```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        
        req := &client.ContextAnalysisRequest{
            NeedDescription: fmt.Sprintf("需求 %d", id),
        }
        
        _, err := client.AnalyzeContext(ctx, req)
        if err != nil {
            log.Printf("请求 %d 失败: %v", id, err)
        }
    }(i)
}

wg.Wait()
```

## 测试

运行测试用例：

```bash
go test -v ./...
```

运行基准测试：

```bash
go test -bench=. -benchmem
```

## 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 本项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 支持

如有问题或建议：

- 📧 提交 [Issue](https://github.com/your-repo/agenthub/issues)
- 💬 加入 [讨论](https://github.com/your-repo/agenthub/discussions)
- 📚 查看 [API 文档](https://docs.agenthub.com)

---

**AgentHub Go Client SDK** - 让 AI 代理集成变得简单! 🚀