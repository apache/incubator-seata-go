# AgentHub API 测试文档

## 基础信息

- **服务地址**: `http://localhost:8080`
- **内容类型**: `application/json`
- **认证方式**: Bearer Token (部分接口需要)

---

## 1. 认证接口

### 1.1 生成Token

```http
POST /auth/token
Content-Type: application/json

{
  "username": "testuser"
}
```

**响应示例**:

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 3600
  }
}
```

### 1.2 刷新Token

```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

---

## 2. Agent管理接口

### 2.1 注册Agent

```http
POST /agent/register
Content-Type: application/json

{
  "agent_card": {
    "name": "test-agent",
    "description": "测试代理",
    "url": "http://example.com/agent",
    "version": "1.0.0",
    "capabilities": {
      "streaming": true,
      "pushNotifications": false
    },
    "defaultInputModes": ["text"],
    "defaultOutputModes": ["text"],
    "skills": [
      {
        "id": "skill1",
        "name": "文本处理",
        "description": "处理文本数据",
        "tags": ["nlp", "text"],
        "examples": ["处理用户输入"],
        "inputModes": ["text"],
        "outputModes": ["text"]
      }
    ]
  },
  "host": "192.168.1.100",
  "port": 8081
}
```

**响应示例**:

```json
{
  "success": true,
  "message": "Agent registered successfully",
  "agent_id": "test-agent"
}
```

### 2.2 发现Agent

```http
POST /agent/discover
Content-Type: application/json

{
  "query": "文本处理"
}
```

**响应示例**:

```json
{
  "agents": [
    {
      "name": "test-agent",
      "description": "测试代理",
      "url": "http://example.com/agent",
      "version": "1.0.0",
      "capabilities": {
        "streaming": true
      },
      "skills": [...]
    }
  ]
}
```

### 2.3 获取Agent详情

```http
GET /agent/get?id=test-agent
```

**响应示例**:

```json
{
  "success": true,
  "data": {
    "id": "test-agent",
    "kind": "RegisteredAgent",
    "version": "v1",
    "agent_card": {...},
    "host": "192.168.1.100",
    "port": 8081,
    "status": "active",
    "last_seen": "2025-01-20T10:00:00Z",
    "registered_at": "2025-01-20T09:00:00Z"
  }
}
```

### 2.4 列出所有Agent

```http
GET /agents
```

**响应示例**:

```json
{
  "success": true,
  "data": [
    {
      "id": "test-agent",
      "status": "active",
      "agent_card": {...}
    }
  ]
}
```

### 2.5 更新Agent状态

```http
PUT /agent/status
Content-Type: application/json

{
  "agent_id": "test-agent",
  "status": "inactive"
}
```

### 2.6 删除Agent

```http
DELETE /agent/remove?id=test-agent
```

### 2.7 Agent心跳

```http
POST /agent/heartbeat
Content-Type: application/json

{
  "agent_id": "test-agent",
  "timestamp": "2025-01-20T10:00:00Z"
}
```

### 2.8 动态上下文分析 ⭐

```http
POST /agent/analyze
Content-Type: application/json

{
  "need_description": "我需要对用户输入的文本进行情感分析和关键词提取",
  "user_context": "电商平台用户评论分析"
}
```

**响应示例**:

```json
{
  "success": true,
  "message": "Successfully analyzed and routed request",
  "matched_skills": [
    {
      "id": "sentiment_analysis",
      "name": "情感分析",
      "description": "分析文本的情感倾向",
      "tags": ["nlp", "sentiment", "emotion"],
      "examples": ["分析评论情感", "判断文本正负面"]
    }
  ],
  "route_result": {
    "agent_url": "http://localhost:8081/text-processor",
    "skill_id": "sentiment_analysis",
    "skill_name": "情感分析",
    "agent_info": {
      "name": "text-processor-agent",
      "description": "专业文本处理代理"
    }
  }
}
```

---

## 3. 系统接口

### 3.1 健康检查

```http
GET /health
```

**响应示例**:

```json
{
  "status": "healthy",
  "timestamp": "2025-01-20T10:00:00Z"
}
```

### 3.2 根路径

```http
GET /
```

**响应示例**:

```
AgentHub is running
```

### 3.3 监控指标

```http
GET /metrics
```

**响应示例**:

```json
{
  "status": "healthy"
}
```

---

## 4. 动态上下文分析测试用例 ⭐

### 4.1 测试场景示例

#### 场景1: 文本处理需求

```json
{
  "need_description": "我需要对用户输入的文本进行情感分析和关键词提取",
  "user_context": "电商平台用户评论分析"
}
```

#### 场景2: 数据分析需求

```json
{
  "need_description": "需要分析销售数据，生成可视化图表",
  "user_context": "月度销售报告生成"
}
```

#### 场景3: 图像处理需求

```json
{
  "need_description": "需要识别图片中的物体和文字",
  "user_context": "产品图片自动标注"
}
```

#### 场景4: 复杂组合需求

```json
{
  "need_description": "我需要先处理用户上传的文档，然后根据内容生成摘要，最后转换为语音播报",
  "user_context": "智能文档助手场景"
}
```

### 4.2 动态上下文完整测试流程

```bash
# 步骤1: 注册一个文本分析Agent
curl -X POST http://localhost:8080/agent/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_card": {
      "name": "text-analyzer",
      "description": "文本分析代理",
      "url": "http://localhost:8081",
      "version": "1.0.0",
      "capabilities": {"streaming": true},
      "defaultInputModes": ["text"],
      "defaultOutputModes": ["json"],
      "skills": [
        {
          "id": "sentiment_analysis",
          "name": "情感分析",
          "description": "分析文本的情感倾向",
          "tags": ["nlp", "sentiment", "emotion"],
          "examples": ["分析评论情感", "判断文本正负面"]
        },
        {
          "id": "keyword_extraction",
          "name": "关键词提取",
          "description": "从文本中提取关键词",
          "tags": ["nlp", "keywords", "extraction"],
          "examples": ["提取文档关键词", "分析主题"]
        }
      ]
    },
    "host": "localhost",
    "port": 8081
  }'

# 步骤2: 注册一个数据分析Agent
curl -X POST http://localhost:8080/agent/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_card": {
      "name": "data-analyzer",
      "description": "数据分析代理",
      "url": "http://localhost:8082",
      "version": "1.0.0",
      "capabilities": {"streaming": false},
      "defaultInputModes": ["json", "csv"],
      "defaultOutputModes": ["json", "image"],
      "skills": [
        {
          "id": "data_visualization",
          "name": "数据可视化",
          "description": "生成各种类型的数据图表",
          "tags": ["data", "chart", "visualization"],
          "examples": ["生成销售图表", "制作趋势分析"]
        },
        {
          "id": "statistical_analysis",
          "name": "统计分析",
          "description": "对数据进行统计分析",
          "tags": ["statistics", "analysis", "data"],
          "examples": ["计算相关性", "趋势分析"]
        }
      ]
    },
    "host": "localhost",
    "port": 8082
  }'

# 步骤3: 测试文本分析需求匹配
curl -X POST http://localhost:8080/agent/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "need_description": "我需要分析用户评论的情感倾向，找出正面和负面的关键词",
    "user_context": "电商平台用户反馈分析"
  }'

# 步骤4: 测试数据分析需求匹配
curl -X POST http://localhost:8080/agent/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "need_description": "需要对销售数据进行统计分析，并生成可视化图表",
    "user_context": "月度销售报告生成"
  }'

# 步骤5: 测试无匹配技能的需求
curl -X POST http://localhost:8080/agent/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "need_description": "我需要训练深度学习模型进行图像识别",
    "user_context": "AI视觉项目"
  }'

# 步骤6: 验证所有注册的Agent
curl -X GET http://localhost:8080/agents
```

---

## 5. 完整API测试用例

### 5.1 基本流程测试

```bash
# 1. 健康检查
curl -X GET http://localhost:8080/health

# 2. 获取认证token
curl -X POST http://localhost:8080/auth/token \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser"}'

# 3. 注册基础Agent
curl -X POST http://localhost:8080/agent/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_card": {
      "name": "basic-agent",
      "description": "基础测试代理",
      "url": "http://example.com/agent",
      "version": "1.0.0",
      "capabilities": {"streaming": true},
      "defaultInputModes": ["text"],
      "defaultOutputModes": ["text"],
      "skills": []
    },
    "host": "localhost",
    "port": 8081
  }'

# 4. 查看所有Agent
curl -X GET http://localhost:8080/agents

# 5. 发现Agent
curl -X POST http://localhost:8080/agent/discover \
  -H "Content-Type: application/json" \
  -d '{"query": "basic"}'

# 6. Agent心跳
curl -X POST http://localhost:8080/agent/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "basic-agent"}'
```

### 5.2 PowerShell测试脚本

```powershell
# PowerShell完整测试脚本
Write-Host "开始AgentHub API测试..." -ForegroundColor Green

# 1. 健康检查
Write-Host "1. 健康检查..." -ForegroundColor Yellow
try {
    $healthResponse = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get
    Write-Host "健康检查成功: $($healthResponse | ConvertTo-Json)" -ForegroundColor Green
} catch {
    Write-Host "健康检查失败: $_" -ForegroundColor Red
    exit
}

# 2. 注册文本分析Agent
Write-Host "2. 注册文本分析Agent..." -ForegroundColor Yellow
$textAnalyzerAgent = @{
    agent_card = @{
        name = "text-analyzer"
        description = "文本分析代理"
        url = "http://localhost:8081"
        version = "1.0.0"
        capabilities = @{ streaming = $true }
        defaultInputModes = @("text")
        defaultOutputModes = @("json")
        skills = @(
            @{
                id = "sentiment_analysis"
                name = "情感分析"
                description = "分析文本的情感倾向"
                tags = @("nlp", "sentiment")
                examples = @("分析评论情感")
            },
            @{
                id = "keyword_extraction"
                name = "关键词提取"
                description = "从文本中提取关键词"
                tags = @("nlp", "keywords")
                examples = @("提取文档关键词")
            }
        )
    }
    host = "localhost"
    port = 8081
} | ConvertTo-Json -Depth 5

try {
    $registerResponse = Invoke-RestMethod -Uri "http://localhost:8080/agent/register" -Method Post -Body $textAnalyzerAgent -ContentType "application/json"
    Write-Host "文本分析Agent注册成功: $($registerResponse.message)" -ForegroundColor Green
} catch {
    Write-Host "Agent注册失败: $_" -ForegroundColor Red
}

# 3. 注册数据分析Agent
Write-Host "3. 注册数据分析Agent..." -ForegroundColor Yellow
$dataAnalyzerAgent = @{
    agent_card = @{
        name = "data-analyzer"
        description = "数据分析代理"
        url = "http://localhost:8082"
        version = "1.0.0"
        capabilities = @{ streaming = $false }
        defaultInputModes = @("json", "csv")
        defaultOutputModes = @("json", "image")
        skills = @(
            @{
                id = "data_visualization"
                name = "数据可视化"
                description = "生成各种类型的数据图表"
                tags = @("data", "chart", "visualization")
                examples = @("生成销售图表")
            }
        )
    }
    host = "localhost"
    port = 8082
} | ConvertTo-Json -Depth 5

try {
    $registerResponse2 = Invoke-RestMethod -Uri "http://localhost:8080/agent/register" -Method Post -Body $dataAnalyzerAgent -ContentType "application/json"
    Write-Host "数据分析Agent注册成功: $($registerResponse2.message)" -ForegroundColor Green
} catch {
    Write-Host "数据分析Agent注册失败: $_" -ForegroundColor Red
}

# 4. 查看所有Agent
Write-Host "4. 查看所有注册的Agent..." -ForegroundColor Yellow
try {
    $allAgents = Invoke-RestMethod -Uri "http://localhost:8080/agents" -Method Get
    Write-Host "当前注册的Agent数量: $($allAgents.data.Count)" -ForegroundColor Green
    $allAgents.data | ForEach-Object { Write-Host "- $($_.id): $($_.status)" -ForegroundColor Cyan }
} catch {
    Write-Host "获取Agent列表失败: $_" -ForegroundColor Red
}

# 5. 测试动态上下文分析 - 文本处理需求
Write-Host "5. 测试动态上下文分析 - 文本处理需求..." -ForegroundColor Yellow
$contextTest1 = @{
    need_description = "我需要分析用户评论的情感倾向，提取其中的关键词"
    user_context = "电商平台用户反馈分析"
} | ConvertTo-Json

try {
    $contextResult1 = Invoke-RestMethod -Uri "http://localhost:8080/agent/analyze" -Method Post -Body $contextTest1 -ContentType "application/json"
    Write-Host "上下文分析成功!" -ForegroundColor Green
    Write-Host "匹配到的技能数量: $($contextResult1.matched_skills.Count)" -ForegroundColor Cyan
    if ($contextResult1.matched_skills.Count -gt 0) {
        Write-Host "最佳匹配技能: $($contextResult1.matched_skills[0].name)" -ForegroundColor Cyan
    }
    if ($contextResult1.route_result) {
        Write-Host "路由目标: $($contextResult1.route_result.agent_url)" -ForegroundColor Cyan
    }
} catch {
    Write-Host "上下文分析失败: $_" -ForegroundColor Red
}

# 6. 测试动态上下文分析 - 数据分析需求
Write-Host "6. 测试动态上下文分析 - 数据分析需求..." -ForegroundColor Yellow
$contextTest2 = @{
    need_description = "需要对销售数据进行统计分析并生成可视化图表"
    user_context = "月度销售报告生成"
} | ConvertTo-Json

try {
    $contextResult2 = Invoke-RestMethod -Uri "http://localhost:8080/agent/analyze" -Method Post -Body $contextTest2 -ContentType "application/json"
    Write-Host "数据分析需求匹配成功!" -ForegroundColor Green
    Write-Host "匹配到的技能数量: $($contextResult2.matched_skills.Count)" -ForegroundColor Cyan
    if ($contextResult2.matched_skills.Count -gt 0) {
        Write-Host "最佳匹配技能: $($contextResult2.matched_skills[0].name)" -ForegroundColor Cyan
    }
} catch {
    Write-Host "数据分析需求匹配失败: $_" -ForegroundColor Red
}

# 7. 测试无匹配技能的需求
Write-Host "7. 测试无匹配技能的需求..." -ForegroundColor Yellow
$contextTest3 = @{
    need_description = "我需要训练深度学习模型进行图像识别"
    user_context = "AI视觉项目开发"
} | ConvertTo-Json

try {
    $contextResult3 = Invoke-RestMethod -Uri "http://localhost:8080/agent/analyze" -Method Post -Body $contextTest3 -ContentType "application/json"
    if (-not $contextResult3.success) {
        Write-Host "预期结果: 无匹配技能 - $($contextResult3.message)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "无匹配技能测试正常: $_" -ForegroundColor Yellow
}

# 8. Agent心跳测试
Write-Host "8. 测试Agent心跳..." -ForegroundColor Yellow
$heartbeatTest = @{
    agent_id = "text-analyzer"
    timestamp = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ssZ")
} | ConvertTo-Json

try {
    $heartbeatResult = Invoke-RestMethod -Uri "http://localhost:8080/agent/heartbeat" -Method Post -Body $heartbeatTest -ContentType "application/json"
    Write-Host "心跳测试成功!" -ForegroundColor Green
} catch {
    Write-Host "心跳测试失败: $_" -ForegroundColor Red
}

Write-Host "AgentHub API测试完成!" -ForegroundColor Green
```

### 5.3 错误响应格式

```json
{
  "success": false,
  "error": {
    "error": "Error message",
    "code": 400
  }
}
```

---

## 6. 状态码说明

- `200` - 成功
- `400` - 请求参数错误
- `401` - 未授权
- `403` - 权限不足
- `404` - 资源未找到
- `405` - 方法不允许
- `500` - 服务器内部错误

---

## 7. 动态上下文测试要点

### 7.1 测试原理

1. **需求描述**: `need_description` 描述具体功能需求
2. **上下文信息**: `user_context` 提供使用场景背景
3. **技能匹配**: 系统根据注册Agent的技能进行智能匹配
4. **AI分析**: 使用AI服务分析需求并生成技能匹配查询
5. **路由结果**: 返回最佳匹配的Agent和技能信息

### 7.2 匹配算法

- **精确匹配**: 技能ID完全匹配 (权重: 100)
- **关键词匹配**: 技能名称/描述包含查询词 (权重: 30-50)
- **标签匹配**: 技能标签匹配 (权重: 20)
- **按分数排序**: 返回得分最高的技能

### 7.3 测试建议

1. **多样化测试**: 测试不同类型的需求描述
2. **边界测试**: 测试无匹配技能、模糊需求等场景
3. **性能测试**: 测试大量技能时的匹配性能
4. **准确性验证**: 验证返回的技能是否真正匹配需求

---

## 8. 注意事项

1. **启动服务**: 在测试前确保AgentHub服务已启动
2. **端口检查**: 确认服务运行在8080端口
3. **JSON格式**: 请求体必须是有效的JSON格式
4. **认证**: 某些接口可能需要Bearer Token认证
5. **错误处理**: 注意检查响应状态码和错误信息
6. **AI服务**: 动态上下文分析需要AI服务支持

---

## 9. 快速测试命令

### 启动服务并测试

```bash
# 启动服务 (如果尚未启动)
./agentHub.exe

# 等待几秒钟让服务启动完成
sleep 3

# 快速健康检查
curl -X GET http://localhost:8080/health

# 如果返回健康状态，则可以开始其他接口测试
```

### 使用Postman测试

1. 导入以上API endpoints到Postman
2. 设置Environment变量: `base_url = http://localhost:8080`
3. 按顺序执行测试用例
4. 重点测试动态上下文分析功能