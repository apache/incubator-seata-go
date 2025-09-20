package config

import (
	"fmt"
	"os"
	"time"

	"agenthub/pkg/utils"
)

// Config represents the application configuration following K8s patterns
type Config struct {
	Hub             HubConfig             `yaml:"hub" json:"hub"`
	Seata           SeataConfig           `yaml:"seata" json:"seata"`
	Logging         LoggingConfig         `yaml:"logging" json:"logging"`
	Metrics         MetricsConfig         `yaml:"metrics" json:"metrics"`
	Auth            AuthConfig            `yaml:"auth" json:"auth"`
	Storage         StorageConfig         `yaml:"storage" json:"storage"`
	NamingServer    NamingServerConfig    `yaml:"naming_server" json:"naming_server"`
	AI              AIConfig              `yaml:"ai" json:"ai"`
	ContextAnalysis ContextAnalysisConfig `yaml:"context_analysis" json:"context_analysis"`
}

// HubConfig holds hub-specific configuration
type HubConfig struct {
	ID            string `yaml:"id" json:"id"`
	Name          string `yaml:"name" json:"name"`
	Version       string `yaml:"version" json:"version"`
	ListenAddress string `yaml:"listen_address" json:"listen_address"`
}

// SeataConfig holds Seata configuration
type SeataConfig struct {
	ServerAddr      string `yaml:"server_addr" json:"server_addr"`
	Namespace       string `yaml:"namespace" json:"namespace"`
	Cluster         string `yaml:"cluster" json:"cluster"`
	HeartbeatPeriod int    `yaml:"heartbeat_period" json:"heartbeat_period"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
	Output string `yaml:"output" json:"output"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	ListenAddress string `yaml:"listen_address" json:"listen_address"`
	Enabled       bool   `yaml:"enabled" json:"enabled"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	JWTSecret string `yaml:"jwt_secret" json:"jwt_secret"`
	JWTExpiry string `yaml:"jwt_expiry" json:"jwt_expiry"`
	Optional  bool   `yaml:"optional" json:"optional"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type       string                 `yaml:"type" json:"type"`
	Connection string                 `yaml:"connection" json:"connection"`
	Options    map[string]interface{} `yaml:"options" json:"options"`
}

// NamingServerConfig holds NamingServer configuration
type NamingServerConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Address  string `yaml:"address" json:"address"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// AIConfig holds AI service configuration
type AIConfig struct {
	Provider  string `yaml:"provider" json:"provider"`   // mock, qwen, openai
	APIKey    string `yaml:"api_key" json:"api_key"`
	BaseURL   string `yaml:"base_url" json:"base_url"`
	Model     string `yaml:"model" json:"model"`
	MaxTokens int    `yaml:"max_tokens" json:"max_tokens"`
	Timeout   string `yaml:"timeout" json:"timeout"`
}

// ContextAnalysisConfig holds context analysis configuration
type ContextAnalysisConfig struct {
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	CacheTTL      string `yaml:"cache_ttl" json:"cache_ttl"`
	MaxConcurrent int    `yaml:"max_concurrent" json:"max_concurrent"`
}

// ConfigLoader loads configuration from various sources
type ConfigLoader struct {
	envUtils  *utils.EnvUtils
	yamlUtils *utils.YAMLUtils
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		envUtils:  utils.NewEnvUtils(),
		yamlUtils: utils.NewYAMLUtils(),
	}
}

// Load loads configuration from file and environment variables
func (c *ConfigLoader) Load() (*Config, error) {
	return c.LoadFromFile("config.yaml")
}

// LoadFromFile loads configuration from a specific file
func (c *ConfigLoader) LoadFromFile(configPath string) (*Config, error) {
	// Set default configuration
	config := c.getDefaultConfig()

	// Try to load from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		if err := c.yamlUtils.UnmarshalFromFile(configPath, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}
	}

	// Override with environment variables
	c.overrideWithEnv(config)

	return config, nil
}

// getDefaultConfig returns the default configuration
func (c *ConfigLoader) getDefaultConfig() *Config {
	return &Config{
		Hub: HubConfig{
			ID:            "agent-hub-01",
			Name:          "AgentHub",
			Version:       "1.0.0",
			ListenAddress: ":8080",
		},
		Seata: SeataConfig{
			ServerAddr:      "127.0.0.1:8091",
			Namespace:       "public",
			Cluster:         "default",
			HeartbeatPeriod: 5000,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Metrics: MetricsConfig{
			ListenAddress: ":9090",
			Enabled:       true,
		},
		Auth: AuthConfig{
			Enabled:   false,
			JWTSecret: "",
			JWTExpiry: "24h",
			Optional:  false,
		},
		Storage: StorageConfig{
			Type:    "memory",
			Options: make(map[string]interface{}),
		},
		NamingServer: NamingServerConfig{
			Enabled:  false,
			Address:  "127.0.0.1:8091",
			Username: "",
			Password: "",
		},
		AI: AIConfig{
			Provider:  "mock",
			APIKey:    "",
			BaseURL:   "",
			Model:     "qwen-plus",
			MaxTokens: 1000,
			Timeout:   "30s",
		},
		ContextAnalysis: ContextAnalysisConfig{
			Enabled:       true,
			CacheTTL:      "1h",
			MaxConcurrent: 10,
		},
	}
}

// overrideWithEnv overrides configuration with environment variables
func (c *ConfigLoader) overrideWithEnv(config *Config) {
	// Hub configuration
	config.Hub.ID = c.envUtils.GetString("HUB_ID", config.Hub.ID)
	config.Hub.ListenAddress = c.envUtils.GetString("LISTEN_ADDRESS", config.Hub.ListenAddress)
	
	// Seata configuration
	config.Seata.ServerAddr = c.envUtils.GetString("SEATA_SERVER_ADDR", config.Seata.ServerAddr)
	config.Seata.Namespace = c.envUtils.GetString("SEATA_NAMESPACE", config.Seata.Namespace)
	config.Seata.Cluster = c.envUtils.GetString("SEATA_CLUSTER", config.Seata.Cluster)
	
	// Logging configuration
	config.Logging.Level = c.envUtils.GetString("LOG_LEVEL", config.Logging.Level)
	config.Logging.Format = c.envUtils.GetString("LOG_FORMAT", config.Logging.Format)
	config.Logging.Output = c.envUtils.GetString("LOG_OUTPUT", config.Logging.Output)
	
	// Metrics configuration
	config.Metrics.ListenAddress = c.envUtils.GetString("METRICS_ADDRESS", config.Metrics.ListenAddress)
	config.Metrics.Enabled = c.envUtils.GetBool("METRICS_ENABLED", config.Metrics.Enabled)
	
	// Auth configuration
	config.Auth.Enabled = c.envUtils.GetBool("AUTH_ENABLED", config.Auth.Enabled)
	config.Auth.JWTSecret = c.envUtils.GetString("JWT_SECRET", config.Auth.JWTSecret)
	config.Auth.JWTExpiry = c.envUtils.GetString("JWT_EXPIRY", config.Auth.JWTExpiry)
	config.Auth.Optional = c.envUtils.GetBool("AUTH_OPTIONAL", config.Auth.Optional)
	
	// Storage configuration
	config.Storage.Type = c.envUtils.GetString("STORAGE_TYPE", config.Storage.Type)
	config.Storage.Connection = c.envUtils.GetString("STORAGE_CONNECTION", config.Storage.Connection)
	
	// NamingServer configuration
	config.NamingServer.Enabled = c.envUtils.GetBool("NAMING_SERVER_ENABLED", config.NamingServer.Enabled)
	config.NamingServer.Address = c.envUtils.GetString("NAMING_SERVER_ADDRESS", config.NamingServer.Address)
	config.NamingServer.Username = c.envUtils.GetString("NAMING_SERVER_USERNAME", config.NamingServer.Username)
	config.NamingServer.Password = c.envUtils.GetString("NAMING_SERVER_PASSWORD", config.NamingServer.Password)
	
	// AI configuration
	config.AI.Provider = c.envUtils.GetString("AI_PROVIDER", config.AI.Provider)
	config.AI.APIKey = c.envUtils.GetString("AI_API_KEY", config.AI.APIKey)
	config.AI.BaseURL = c.envUtils.GetString("AI_BASE_URL", config.AI.BaseURL)
	config.AI.Model = c.envUtils.GetString("AI_MODEL", config.AI.Model)
	config.AI.MaxTokens = c.envUtils.GetInt("AI_MAX_TOKENS", config.AI.MaxTokens)
	config.AI.Timeout = c.envUtils.GetString("AI_TIMEOUT", config.AI.Timeout)
	
	// Context Analysis configuration
	config.ContextAnalysis.Enabled = c.envUtils.GetBool("CONTEXT_ANALYSIS_ENABLED", config.ContextAnalysis.Enabled)
	config.ContextAnalysis.CacheTTL = c.envUtils.GetString("CONTEXT_ANALYSIS_CACHE_TTL", config.ContextAnalysis.CacheTTL)
	config.ContextAnalysis.MaxConcurrent = c.envUtils.GetInt("CONTEXT_ANALYSIS_MAX_CONCURRENT", config.ContextAnalysis.MaxConcurrent)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if utils.IsEmpty(c.Hub.ID) {
		return fmt.Errorf("hub.id is required")
	}
	if utils.IsEmpty(c.Hub.ListenAddress) {
		return fmt.Errorf("hub.listen_address is required")
	}
	if utils.IsEmpty(c.Seata.ServerAddr) {
		return fmt.Errorf("seata.server_addr is required")
	}
	if utils.IsEmpty(c.Seata.Namespace) {
		return fmt.Errorf("seata.namespace is required")
	}
	if utils.IsEmpty(c.Seata.Cluster) {
		return fmt.Errorf("seata.cluster is required")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging.level: %s (must be debug/info/warn/error)", c.Logging.Level)
	}

	// Validate JWT expiry format
	if c.Auth.Enabled && c.Auth.JWTExpiry != "" {
		if _, err := time.ParseDuration(c.Auth.JWTExpiry); err != nil {
			return fmt.Errorf("invalid JWT expiry format: %s", c.Auth.JWTExpiry)
		}
	}

	return nil
}

// GetPort extracts port from listen address
func (c *Config) GetPort() string {
	if len(c.Hub.ListenAddress) > 0 && c.Hub.ListenAddress[0] == ':' {
		return c.Hub.ListenAddress[1:]
	}
	return "8080"
}

// GetJWTExpiry returns JWT expiry as duration
func (c *Config) GetJWTExpiry() time.Duration {
	if c.Auth.JWTExpiry == "" {
		return 24 * time.Hour
	}
	
	duration, err := time.ParseDuration(c.Auth.JWTExpiry)
	if err != nil {
		return 24 * time.Hour
	}
	
	return duration
}

// Address returns the listen address (backward compatibility)
func (c *Config) Address() string {
	return c.Hub.ListenAddress
}

// Global configuration loader instance
var globalLoader = NewConfigLoader()

// Load loads configuration using the global loader
func Load() (*Config, error) {
	return globalLoader.Load()
}

// LoadFromFile loads configuration from a file using the global loader
func LoadFromFile(configPath string) (*Config, error) {
	return globalLoader.LoadFromFile(configPath)
}