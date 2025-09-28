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

package utils

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvUtils provides environment variable utilities
type EnvUtils struct{}

// NewEnvUtils creates a new environment utilities instance
func NewEnvUtils() *EnvUtils {
	return &EnvUtils{}
}

// GetString gets a string environment variable with default value
func (e *EnvUtils) GetString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetBool gets a boolean environment variable with default value
func (e *EnvUtils) GetBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
}

// GetInt gets an integer environment variable with default value
func (e *EnvUtils) GetInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}
	return defaultValue
}

// GetDuration gets a duration environment variable with default value
func (e *EnvUtils) GetDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}
	return defaultValue
}

// GetStringSlice gets a string slice environment variable (comma-separated) with default value
func (e *EnvUtils) GetStringSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}
	return result
}

// IsSet checks if an environment variable is set (even if empty)
func (e *EnvUtils) IsSet(key string) bool {
	_, exists := os.LookupEnv(key)
	return exists
}

// MustGet gets an environment variable or panics if not set
func (e *EnvUtils) MustGet(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("required environment variable not set: " + key)
	}
	return value
}

// Global instance for convenience
var globalEnv = NewEnvUtils()

// Package-level convenience functions
func GetString(key, defaultValue string) string {
	return globalEnv.GetString(key, defaultValue)
}

func GetBool(key string, defaultValue bool) bool {
	return globalEnv.GetBool(key, defaultValue)
}

func GetInt(key string, defaultValue int) int {
	return globalEnv.GetInt(key, defaultValue)
}

func GetDuration(key string, defaultValue time.Duration) time.Duration {
	return globalEnv.GetDuration(key, defaultValue)
}

func GetStringSlice(key string, defaultValue []string) []string {
	return globalEnv.GetStringSlice(key, defaultValue)
}

func IsSet(key string) bool {
	return globalEnv.IsSet(key)
}

func MustGet(key string) string {
	return globalEnv.MustGet(key)
}
