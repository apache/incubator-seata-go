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

# Go Agent CLI

一个用于生成 Go 语言 Agent 项目的命令行脚手架工具。

## 功能特性

- 🚀 快速生成 Go Agent 项目
- 📋 支持两种项目模式：Default 和 Provider
- 🔧 自动生成项目结构和配置文件
- 📊 内置 Agent Card 配置
- 🐳 包含 Docker 支持
- 📝 自动生成项目文档

## 支持的模式

### Default 模式
标准的 Agent 项目，包含：
- 完整的技能系统架构
- HTTP API 接口
- 配置管理
- 日志系统
- 健康检查

### Provider 模式  
外部服务集成 Agent，包含：
- 服务适配器层
- 外部服务客户端
- 数据映射器
- 集成示例和文档

## 快速开始

### 安装

```bash
# 克隆项目
git clone <repository-url>
cd agent-cli

# 安装依赖
go mod tidy

# 构建工具
go build -o agent-cli cmd/cli/main.go
```

### 使用方法

#### 创建 Default 模式项目

```bash
./agent-cli create my-agent --mode default
```

#### 创建 Provider 模式项目

```bash
./agent-cli create my-provider --mode provider
```

#### 指定输出目录和模块名

```bash
./agent-cli create my-agent --mode default --output ./projects --module github.com/myorg/my-agent
```

### 生成的项目结构

#### Default 模式项目结构
```
my-agent/
├── main.go                    # 入口文件
├── go.mod                     # Go模块定义
├── config/
│   ├── config.yaml           # 配置文件
│   └── agent_card.json       # Agent能力卡片
├── internal/
│   ├── agent/
│   │   ├── agent.go          # Agent核心逻辑
│   │   ├── skills/           # 技能实现目录
│   │   │   └── example.go    # 示例技能
│   │   └── handlers/         # RPC处理器
│   │       └── rpc.go        # RPC路由
│   ├── config/
│   │   └── config.go         # 配置加载
│   └── utils/
│       └── logger.go         # 日志工具
├── scripts/
│   ├── build.sh              # 构建脚本
│   └── run.sh                # 运行脚本
├── Dockerfile                # Docker构建文件
├── docker-compose.yml        # 本地开发环境
└── README.md                 # 项目文档
```

#### Provider 模式项目结构
```
my-provider/
├── main.go                    # 入口文件
├── go.mod
├── config/
│   ├── config.yaml
│   └── agent_card.json
├── internal/
│   ├── agent/
│   │   ├── agent.go
│   │   └── provider/         # Provider层
│   │       ├── adapter.go    # 适配器接口
│   │       ├── client.go     # 外部服务客户端
│   │       └── mapper.go     # 数据映射
│   ├── skills/
│   │   └── provider_skills.go # Provider技能实现
│   └── handlers/
│       └── rpc.go
├── examples/
│   └── external_service.go   # 外部服务示例
└── docs/
    └── integration.md        # 集成指南
```

## 生成项目的使用方法

### 运行 Agent

```bash
cd my-agent
go mod tidy
go run main.go
```

### 测试 API

```bash
# 健康检查
curl http://localhost:8080/health

# 获取 Agent 卡片
curl http://localhost:8080/agent-card

# 执行技能 (Default 模式)
curl -X POST http://localhost:8080/skills/example \
  -H "Content-Type: application/json" \
  -d '{"input": "Hello World"}'

# 执行技能 (Provider 模式)
curl -X POST http://localhost:8080/skills/external_service_call \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "example-api",
    "endpoint": "/api/data",
    "method": "GET"
  }'
```

### Docker 部署

```bash
# 使用 Docker Compose
docker-compose up --build

# 或手动构建
docker build -t my-agent .
docker run -p 8080:8080 my-agent
```

## CLI 命令参考

### 全局选项

```bash
agent-cli [command] [flags]
```

### 可用命令

- `create` - 创建新的 Go Agent 项目
- `version` - 显示版本信息
- `help` - 显示帮助信息

### create 命令

```bash
agent-cli create [project-name] [flags]
```

**参数：**
- `project-name` - 项目名称（必需）

**选项：**
- `-m, --mode string` - 项目模式 (default|provider)，默认为 "default"
- `-o, --output string` - 输出目录，默认为当前目录 "."
- `--module string` - Go 模块名，默认为项目名称

**示例：**

```bash
# 基本用法
agent-cli create my-agent

# 指定模式
agent-cli create my-provider --mode provider

# 指定输出目录
agent-cli create my-agent --output ./projects

# 指定模块名
agent-cli create my-agent --module github.com/myorg/my-agent

# 完整示例
agent-cli create my-enterprise-agent \
  --mode provider \
  --output ./enterprise-projects \
  --module github.com/mycompany/agents/enterprise-agent
```

## 项目架构

### 脚手架工具架构

```
go-agent-cli/
├── cmd/cli/main.go           # CLI 入口
├── internal/
│   ├── cmd/                  # 命令实现
│   │   ├── root.go          # 根命令
│   │   ├── create.go        # create 命令
│   │   └── version.go       # version 命令
│   ├── config/              # 配置结构
│   │   └── project.go       # 项目配置定义
│   └── generator/           # 代码生成器
│       ├── generator.go     # 生成器核心
│       └── templates.go     # 项目模板
└── go.mod
```

### 生成项目的技术栈

- **Web 框架**: Gin
- **配置管理**: Viper
- **日志系统**: Zap
- **容器化**: Docker + Docker Compose
- **构建工具**: Go 标准工具链

## 开发指南

### 扩展模板

要添加新的项目模板：

1. 在 `internal/generator/templates.go` 中定义新模板
2. 在 `generator.go` 中添加生成逻辑
3. 更新 `create.go` 中的模式验证

### 添加新命令

要添加新的 CLI 命令：

1. 在 `internal/cmd/` 下创建新文件
2. 实现 Cobra 命令
3. 在 `root.go` 中注册命令

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

[指定许可证]

## 版本历史

- v1.0.0 - 初始版本
  - 支持 Default 和 Provider 两种模式
  - 完整的项目生成功能
  - Docker 支持
  - 详细的文档和示例