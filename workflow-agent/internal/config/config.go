/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"strings"
	"time"
)

import (
	"github.com/spf13/viper"
)

import (
	"seata-go-ai-workflow-agent/pkg/errors"
)

// Config represents the application configuration
type Config struct {
	Agent         AgentConfig         `mapstructure:"agent"`
	Hub           HubConfig           `mapstructure:"hub"`
	Logger        LoggerConfig        `mapstructure:"logger"`
	Server        ServerConfig        `mapstructure:"server"`
	LLM           LLMConfig           `mapstructure:"llm"`
	Orchestration OrchestrationConfig `mapstructure:"orchestration"`
}

// AgentConfig holds agent-specific configuration
type AgentConfig struct {
	Name        string   `mapstructure:"name"`
	Description string   `mapstructure:"description"`
	Version     string   `mapstructure:"version"`
	Skills      []string `mapstructure:"skills"`
}

// HubConfig holds AgentHub client configuration
type HubConfig struct {
	BaseURL          string `mapstructure:"base_url"`
	Timeout          int    `mapstructure:"timeout"`
	HeartbeatEnabled bool   `mapstructure:"heartbeat_enabled"`
	HeartbeatPeriod  int    `mapstructure:"heartbeat_period"`
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// LLMConfig holds LLM client configuration
type LLMConfig struct {
	DefaultModel string                 `mapstructure:"default_model"`
	APIKey       string                 `mapstructure:"api_key"`
	BaseURL      string                 `mapstructure:"base_url"`
	Config       map[string]interface{} `mapstructure:"config"`
}

// OrchestrationConfig holds workflow orchestration configuration
type OrchestrationConfig struct {
	MaxRetryCount int `mapstructure:"max_retry_count"`
}

// Load loads configuration from file
func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	setDefaults(v)

	v.SetEnvPrefix("AGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, errors.Wrap(errors.ErrConfig, "failed to read config", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(errors.ErrConfig, "failed to unmarshal config", err)
	}

	return &cfg, nil
}

// LoadOrDefault loads configuration from file or returns default config
func LoadOrDefault(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		return Default()
	}
	return cfg
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		Agent: AgentConfig{
			Name:        "workflow-agent",
			Description: "Seata workflow agent",
			Version:     "1.0.0",
			Skills:      []string{},
		},
		Hub: HubConfig{
			BaseURL:          "http://localhost:8080",
			Timeout:          30,
			HeartbeatEnabled: true,
			HeartbeatPeriod:  30,
		},
		Logger: LoggerConfig{
			Level:      "info",
			Format:     "text",
			OutputPath: "stdout",
		},
		Server: ServerConfig{
			Host: "localhost",
			Port: 9090,
		},
		LLM: LLMConfig{
			DefaultModel: "qwen-max",
			APIKey:       "",
			BaseURL:      "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Config:       nil,
		},
		Orchestration: OrchestrationConfig{
			MaxRetryCount: 3,
		},
	}
}

// setDefaults sets default values in viper
func setDefaults(v *viper.Viper) {
	v.SetDefault("agent.name", "workflow-agent")
	v.SetDefault("agent.description", "Seata workflow agent")
	v.SetDefault("agent.version", "1.0.0")
	v.SetDefault("agent.skills", []string{})

	v.SetDefault("hub.base_url", "http://localhost:8080")
	v.SetDefault("hub.timeout", 30)
	v.SetDefault("hub.heartbeat_enabled", true)
	v.SetDefault("hub.heartbeat_period", 30)

	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "text")
	v.SetDefault("logger.output_path", "stdout")

	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", 9090)

	v.SetDefault("llm.default_model", "qwen-max")
	v.SetDefault("llm.api_key", "")
	v.SetDefault("llm.base_url", "https://dashscope.aliyuncs.com/compatible-mode/v1")
	v.SetDefault("llm.config", map[string]interface{}{})

	v.SetDefault("orchestration.max_retry_count", 3)
}

// GetHubTimeout returns hub timeout as duration
func (c *Config) GetHubTimeout() time.Duration {
	return time.Duration(c.Hub.Timeout) * time.Second
}

// GetHeartbeatPeriod returns heartbeat period as duration
func (c *Config) GetHeartbeatPeriod() time.Duration {
	return time.Duration(c.Hub.HeartbeatPeriod) * time.Second
}
