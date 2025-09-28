package generator

// Go Module Template
const goModTemplate = `module {{.ModuleName}}

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/spf13/viper v1.18.2
	gopkg.in/yaml.v3 v3.0.1
	go.uber.org/zap v1.26.0
)
`

// Default Mode Templates
const defaultMainTemplate = `package main

import (
	"log"

	"{{.ModuleName}}/internal/agent"
	"{{.ModuleName}}/internal/config"
	"{{.ModuleName}}/internal/utils"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := utils.NewLogger(cfg.Log.Level)
	defer logger.Sync()

	agent := agent.New(cfg, logger)
	if err := agent.Start(); err != nil {
		logger.Fatal("Failed to start agent", "error", err)
	}
}
`

const defaultConfigTemplate = `# {{.ProjectName}} Agent Configuration
# Generated on {{.Timestamp}}

server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

agent:
  name: "{{.ProjectName}}"
  version: "1.0.0"
  description: "A default mode Go agent"

log:
  level: "info"
  format: "json"

skills:
  enabled:
    - "example"
`

const defaultAgentTemplate = `package agent

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"{{.ModuleName}}/internal/agent/handlers"
	"{{.ModuleName}}/internal/agent/skills"
	"{{.ModuleName}}/internal/config"
)

type Agent struct {
	cfg    *config.Config
	logger *zap.Logger
	router *gin.Engine
	skills map[string]skills.Skill
}

func New(cfg *config.Config, logger *zap.Logger) *Agent {
	a := &Agent{
		cfg:    cfg,
		logger: logger,
		router: gin.New(),
		skills: make(map[string]skills.Skill),
	}

	a.initializeSkills()
	a.setupRoutes()
	return a
}

func (a *Agent) initializeSkills() {
	a.skills["example"] = skills.NewExampleSkill(a.logger)
}

func (a *Agent) setupRoutes() {
	a.router.Use(gin.Logger())
	a.router.Use(gin.Recovery())

	a.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	a.router.GET("/agent-card", handlers.NewAgentCardHandler(a.cfg).Handle)

	skillsGroup := a.router.Group("/skills")
	for name, skill := range a.skills {
		skillsGroup.POST("/"+name, a.createSkillHandler(skill))
	}
}

func (a *Agent) createSkillHandler(skill skills.Skill) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request map[string]interface{}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := skill.Execute(context.Background(), request)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func (a *Agent) Start() error {
	addr := fmt.Sprintf("%s:%d", a.cfg.Server.Host, a.cfg.Server.Port)
	a.logger.Info("Starting agent server", zap.String("address", addr))
	return a.router.Run(addr)
}
`

const defaultSkillTemplate = `package skills

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type Skill interface {
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	Name() string
	Description() string
}

type ExampleSkill struct {
	logger *zap.Logger
}

func NewExampleSkill(logger *zap.Logger) *ExampleSkill {
	return &ExampleSkill{logger: logger}
}

func (s *ExampleSkill) Name() string {
	return "example"
}

func (s *ExampleSkill) Description() string {
	return "An example skill that processes input text"
}

func (s *ExampleSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	input, ok := params["input"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'input' parameter")
	}

	s.logger.Info("Executing example skill", zap.String("input", input))

	result := fmt.Sprintf("Processed: %s (length: %d)", input, len(input))

	return map[string]interface{}{
		"result":   result,
		"original": input,
		"length":   len(input),
	}, nil
}
`

const defaultRPCTemplate = `package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"{{.ModuleName}}/internal/config"
)

type AgentCardHandler struct {
	cfg *config.Config
}

func NewAgentCardHandler(cfg *config.Config) *AgentCardHandler {
	return &AgentCardHandler{cfg: cfg}
}

func (h *AgentCardHandler) Handle(c *gin.Context) {
	agentCard, err := h.cfg.LoadAgentCard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load agent card"})
		return
	}

	c.JSON(http.StatusOK, agentCard)
}
`

const configLoaderTemplate = `package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig
	Agent  AgentConfig
	Log    LogConfig
	Skills SkillsConfig
}

type ServerConfig struct {
	Host string
	Port int
	Mode string
}

type AgentConfig struct {
	Name        string
	Version     string
	Description string
}

type LogConfig struct {
	Level  string
	Format string
}

type SkillsConfig struct {
	Enabled []string
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) LoadAgentCard() (interface{}, error) {
	cardPath := filepath.Join("config", "agent_card.json")
	data, err := os.ReadFile(cardPath)
	if err != nil {
		return nil, err
	}

	var card interface{}
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, err
	}

	return card, nil
}
`

const loggerTemplate = `package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level.SetLevel(zapLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, _ := config.Build()
	return logger
}
`

const buildScriptTemplate = `#!/bin/bash
set -e

echo "Building {{.ProjectName}}..."

rm -rf ./bin
mkdir -p ./bin

go build -o ./bin/{{.ProjectName}} ./main.go

echo "Build completed successfully!"
echo "Binary location: ./bin/{{.ProjectName}}"
`

const runScriptTemplate = `#!/bin/bash
set -e

echo "Starting {{.ProjectName}}..."

if [ ! -f "./config/config.yaml" ]; then
    echo "Error: config/config.yaml not found"
    exit 1
fi

go run main.go
`

const dockerfileTemplate = `FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/config ./config/

EXPOSE 8080

CMD ["./main"]
`

const dockerComposeTemplate = `version: '3.8'

services:
  {{.ProjectName}}:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config:/root/config
    environment:
      - GIN_MODE=release
    restart: unless-stopped
`

const defaultReadmeTemplate = `# {{.ProjectName}}

A {{.Mode}} mode Go agent generated on {{.Timestamp}}.

## Description

This is a Go-based agent that provides skill-based functionality through HTTP APIs.

## Features

- RESTful API endpoints
- Configurable skills system
- Health check endpoint
- Agent card endpoint
- Docker support
- Structured logging

## Quick Start

1. Install dependencies:
   go mod tidy

2. Run the agent:
   go run main.go

3. Test the agent:
   curl http://localhost:8080/health
   curl http://localhost:8080/agent-card
   curl -X POST http://localhost:8080/skills/example -H "Content-Type: application/json" -d '{"input": "Hello World"}'

## Configuration

Edit config/config.yaml to customize the agent settings.

## Docker

docker-compose up --build

## API Endpoints

- GET /health - Health check
- GET /agent-card - Agent capabilities
- POST /skills/{skill-name} - Execute a skill

## Skills

- example: Processes input text and returns analysis

## Development

- scripts/build.sh - Build the project
- scripts/run.sh - Run in development mode
`

// Provider Mode Templates (simplified for now)
const providerMainTemplate = defaultMainTemplate
const providerConfigTemplate = defaultConfigTemplate
const providerAgentTemplate = defaultAgentTemplate
const providerAdapterTemplate = defaultAgentTemplate
const providerClientTemplate = defaultAgentTemplate
const providerMapperTemplate = defaultAgentTemplate
const providerSkillsTemplate = defaultSkillTemplate
const providerRPCTemplate = defaultRPCTemplate
const externalServiceTemplate = defaultReadmeTemplate
const integrationDocTemplate = defaultReadmeTemplate
