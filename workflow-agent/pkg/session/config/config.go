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
	"time"
)

// StoreType represents the type of session store
type StoreType string

const (
	StoreTypeMemory StoreType = "memory"
	StoreTypeRedis  StoreType = "redis"
	StoreTypeMySQL  StoreType = "mysql"
)

// Config holds session manager configuration
type Config struct {
	Store   StoreConfig   `yaml:"store" json:"store"`
	Session SessionConfig `yaml:"session" json:"session"`
	Cleanup CleanupConfig `yaml:"cleanup" json:"cleanup"`
}

// StoreConfig holds store-specific configuration
type StoreConfig struct {
	Type   StoreType    `yaml:"type" json:"type"`
	Memory MemoryConfig `yaml:"memory" json:"memory"`
	Redis  RedisConfig  `yaml:"redis" json:"redis"`
	MySQL  MySQLConfig  `yaml:"mysql" json:"mysql"`
}

// SessionConfig holds session-specific configuration
type SessionConfig struct {
	DefaultTTL        time.Duration `yaml:"default_ttl" json:"default_ttl"`
	MaxMessageCount   int           `yaml:"max_message_count" json:"max_message_count"`
	MaxContextSize    int           `yaml:"max_context_size" json:"max_context_size"`
	EnableCompression bool          `yaml:"enable_compression" json:"enable_compression"`
	EnableEncryption  bool          `yaml:"enable_encryption" json:"enable_encryption"`
	EncryptionKey     string        `yaml:"encryption_key" json:"-"`
}

// CleanupConfig holds cleanup configuration
type CleanupConfig struct {
	Enabled   bool          `yaml:"enabled" json:"enabled"`
	Interval  time.Duration `yaml:"interval" json:"interval"`
	BatchSize int           `yaml:"batch_size" json:"batch_size"`
}

// MemoryConfig holds memory store configuration
type MemoryConfig struct {
	MaxSessions     int           `yaml:"max_sessions" json:"max_sessions"`
	EvictionPolicy  string        `yaml:"eviction_policy" json:"eviction_policy"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}

// RedisConfig holds Redis store configuration
type RedisConfig struct {
	Addr         string        `yaml:"addr" json:"addr"`
	Password     string        `yaml:"password" json:"-"`
	DB           int           `yaml:"db" json:"db"`
	PoolSize     int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns" json:"min_idle_conns"`
	MaxRetries   int           `yaml:"max_retries" json:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	KeyPrefix    string        `yaml:"key_prefix" json:"key_prefix"`
	TLS          TLSConfig     `yaml:"tls" json:"tls"`
}

// MySQLConfig holds MySQL store configuration
type MySQLConfig struct {
	DSN             string        `yaml:"dsn" json:"-"`
	Host            string        `yaml:"host" json:"host"`
	Port            int           `yaml:"port" json:"port"`
	Username        string        `yaml:"username" json:"username"`
	Password        string        `yaml:"password" json:"-"`
	Database        string        `yaml:"database" json:"database"`
	TableName       string        `yaml:"table_name" json:"table_name"`
	MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`
	TLS             TLSConfig     `yaml:"tls" json:"tls"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled" json:"enabled"`
	CertFile           string `yaml:"cert_file" json:"cert_file"`
	KeyFile            string `yaml:"key_file" json:"key_file"`
	CAFile             string `yaml:"ca_file" json:"ca_file"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Store: StoreConfig{
			Type: StoreTypeMemory,
			Memory: MemoryConfig{
				MaxSessions:     10000,
				EvictionPolicy:  "lru",
				CleanupInterval: 5 * time.Minute,
			},
			Redis: RedisConfig{
				Addr:         "localhost:6379",
				DB:           0,
				PoolSize:     10,
				MinIdleConns: 5,
				MaxRetries:   3,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
				IdleTimeout:  5 * time.Minute,
				KeyPrefix:    "session:",
			},
			MySQL: MySQLConfig{
				Host:            "localhost",
				Port:            3306,
				Database:        "sessions",
				TableName:       "sessions",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 10 * time.Minute,
			},
		},
		Session: SessionConfig{
			DefaultTTL:        24 * time.Hour,
			MaxMessageCount:   1000,
			MaxContextSize:    1024 * 1024, // 1MB
			EnableCompression: false,
			EnableEncryption:  false,
		},
		Cleanup: CleanupConfig{
			Enabled:   true,
			Interval:  time.Hour,
			BatchSize: 100,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Store.Type == "" {
		c.Store.Type = StoreTypeMemory
	}

	if c.Session.DefaultTTL <= 0 {
		c.Session.DefaultTTL = 24 * time.Hour
	}

	if c.Session.MaxMessageCount <= 0 {
		c.Session.MaxMessageCount = 1000
	}

	if c.Cleanup.Interval <= 0 {
		c.Cleanup.Interval = time.Hour
	}

	if c.Cleanup.BatchSize <= 0 {
		c.Cleanup.BatchSize = 100
	}

	return nil
}
