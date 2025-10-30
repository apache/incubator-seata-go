package config

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
