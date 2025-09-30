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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadFromFile loads configuration from a file (JSON or YAML)
func LoadFromFile(filename string) (*Config, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", filename)
	}

	// Read file content
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine file format by extension
	ext := strings.ToLower(filepath.Ext(filename))

	var config Config
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &config)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(&config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	config := DefaultConfig()
	applyEnvOverrides(config)
	return config
}

// SaveToFile saves configuration to a file
func SaveToFile(config *Config, filename string) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Determine file format by extension
	ext := strings.ToLower(filepath.Ext(filename))

	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write file
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides to config
func applyEnvOverrides(config *Config) {
	// Store type
	if storeType := os.Getenv("SESSION_STORE_TYPE"); storeType != "" {
		config.Store.Type = StoreType(storeType)
	}

	// Memory store
	if maxSessions := os.Getenv("SESSION_MEMORY_MAX_SESSIONS"); maxSessions != "" {
		if val := parseInt(maxSessions); val > 0 {
			config.Store.Memory.MaxSessions = val
		}
	}
	if evictionPolicy := os.Getenv("SESSION_MEMORY_EVICTION_POLICY"); evictionPolicy != "" {
		config.Store.Memory.EvictionPolicy = evictionPolicy
	}
	if cleanupInterval := os.Getenv("SESSION_MEMORY_CLEANUP_INTERVAL"); cleanupInterval != "" {
		if duration := parseDuration(cleanupInterval); duration > 0 {
			config.Store.Memory.CleanupInterval = duration
		}
	}

	// Redis store
	if addr := os.Getenv("SESSION_REDIS_ADDR"); addr != "" {
		config.Store.Redis.Addr = addr
	}
	if password := os.Getenv("SESSION_REDIS_PASSWORD"); password != "" {
		config.Store.Redis.Password = password
	}
	if db := os.Getenv("SESSION_REDIS_DB"); db != "" {
		if val := parseInt(db); val >= 0 {
			config.Store.Redis.DB = val
		}
	}
	if poolSize := os.Getenv("SESSION_REDIS_POOL_SIZE"); poolSize != "" {
		if val := parseInt(poolSize); val > 0 {
			config.Store.Redis.PoolSize = val
		}
	}
	if keyPrefix := os.Getenv("SESSION_REDIS_KEY_PREFIX"); keyPrefix != "" {
		config.Store.Redis.KeyPrefix = keyPrefix
	}

	// MySQL store
	if dsn := os.Getenv("SESSION_MYSQL_DSN"); dsn != "" {
		config.Store.MySQL.DSN = dsn
	}
	if host := os.Getenv("SESSION_MYSQL_HOST"); host != "" {
		config.Store.MySQL.Host = host
	}
	if port := os.Getenv("SESSION_MYSQL_PORT"); port != "" {
		if val := parseInt(port); val > 0 {
			config.Store.MySQL.Port = val
		}
	}
	if username := os.Getenv("SESSION_MYSQL_USERNAME"); username != "" {
		config.Store.MySQL.Username = username
	}
	if password := os.Getenv("SESSION_MYSQL_PASSWORD"); password != "" {
		config.Store.MySQL.Password = password
	}
	if database := os.Getenv("SESSION_MYSQL_DATABASE"); database != "" {
		config.Store.MySQL.Database = database
	}
	if tableName := os.Getenv("SESSION_MYSQL_TABLE_NAME"); tableName != "" {
		config.Store.MySQL.TableName = tableName
	}

	// Session configuration
	if defaultTTL := os.Getenv("SESSION_DEFAULT_TTL"); defaultTTL != "" {
		if duration := parseDuration(defaultTTL); duration > 0 {
			config.Session.DefaultTTL = duration
		}
	}
	if maxMessageCount := os.Getenv("SESSION_MAX_MESSAGE_COUNT"); maxMessageCount != "" {
		if val := parseInt(maxMessageCount); val > 0 {
			config.Session.MaxMessageCount = val
		}
	}
	if enableCompression := os.Getenv("SESSION_ENABLE_COMPRESSION"); enableCompression != "" {
		config.Session.EnableCompression = parseBool(enableCompression)
	}
	if enableEncryption := os.Getenv("SESSION_ENABLE_ENCRYPTION"); enableEncryption != "" {
		config.Session.EnableEncryption = parseBool(enableEncryption)
	}
	if encryptionKey := os.Getenv("SESSION_ENCRYPTION_KEY"); encryptionKey != "" {
		config.Session.EncryptionKey = encryptionKey
	}

	// Cleanup configuration
	if cleanupEnabled := os.Getenv("SESSION_CLEANUP_ENABLED"); cleanupEnabled != "" {
		config.Cleanup.Enabled = parseBool(cleanupEnabled)
	}
	if cleanupInterval := os.Getenv("SESSION_CLEANUP_INTERVAL"); cleanupInterval != "" {
		if duration := parseDuration(cleanupInterval); duration > 0 {
			config.Cleanup.Interval = duration
		}
	}
	if batchSize := os.Getenv("SESSION_CLEANUP_BATCH_SIZE"); batchSize != "" {
		if val := parseInt(batchSize); val > 0 {
			config.Cleanup.BatchSize = val
		}
	}

	// TLS configuration for Redis
	if tlsEnabled := os.Getenv("SESSION_REDIS_TLS_ENABLED"); tlsEnabled != "" {
		config.Store.Redis.TLS.Enabled = parseBool(tlsEnabled)
	}
	if certFile := os.Getenv("SESSION_REDIS_TLS_CERT_FILE"); certFile != "" {
		config.Store.Redis.TLS.CertFile = certFile
	}
	if keyFile := os.Getenv("SESSION_REDIS_TLS_KEY_FILE"); keyFile != "" {
		config.Store.Redis.TLS.KeyFile = keyFile
	}
	if caFile := os.Getenv("SESSION_REDIS_TLS_CA_FILE"); caFile != "" {
		config.Store.Redis.TLS.CAFile = caFile
	}
	if skipVerify := os.Getenv("SESSION_REDIS_TLS_SKIP_VERIFY"); skipVerify != "" {
		config.Store.Redis.TLS.InsecureSkipVerify = parseBool(skipVerify)
	}

	// TLS configuration for MySQL
	if tlsEnabled := os.Getenv("SESSION_MYSQL_TLS_ENABLED"); tlsEnabled != "" {
		config.Store.MySQL.TLS.Enabled = parseBool(tlsEnabled)
	}
	if certFile := os.Getenv("SESSION_MYSQL_TLS_CERT_FILE"); certFile != "" {
		config.Store.MySQL.TLS.CertFile = certFile
	}
	if keyFile := os.Getenv("SESSION_MYSQL_TLS_KEY_FILE"); keyFile != "" {
		config.Store.MySQL.TLS.KeyFile = keyFile
	}
	if caFile := os.Getenv("SESSION_MYSQL_TLS_CA_FILE"); caFile != "" {
		config.Store.MySQL.TLS.CAFile = caFile
	}
	if skipVerify := os.Getenv("SESSION_MYSQL_TLS_SKIP_VERIFY"); skipVerify != "" {
		config.Store.MySQL.TLS.InsecureSkipVerify = parseBool(skipVerify)
	}
}

// Helper functions for parsing environment variables
func parseInt(s string) int {
	var val int
	fmt.Sscanf(s, "%d", &val)
	return val
}

func parseDuration(s string) time.Duration {
	duration, _ := time.ParseDuration(s)
	return duration
}

func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1" || s == "yes" || s == "on"
}
