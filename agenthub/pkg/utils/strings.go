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
	"strings"
	"unicode"
)

// StringMatcher provides string matching utilities
type StringMatcher struct{}

// NewStringMatcher creates a new string matcher instance
func NewStringMatcher() *StringMatcher {
	return &StringMatcher{}
}

// ContainsIgnoreCase performs case-insensitive substring matching
func (sm *StringMatcher) ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ExactMatchIgnoreCase performs case-insensitive exact matching
func (sm *StringMatcher) ExactMatchIgnoreCase(s1, s2 string) bool {
	return strings.ToLower(s1) == strings.ToLower(s2)
}

// MatchAny checks if any of the patterns match the target string
func (sm *StringMatcher) MatchAny(target string, patterns []string) bool {
	for _, pattern := range patterns {
		if sm.ContainsIgnoreCase(target, pattern) {
			return true
		}
	}
	return false
}

// MatchAll checks if all patterns match the target string
func (sm *StringMatcher) MatchAll(target string, patterns []string) bool {
	for _, pattern := range patterns {
		if !sm.ContainsIgnoreCase(target, pattern) {
			return false
		}
	}
	return true
}

// MatchFields performs field-based search across multiple strings
func (sm *StringMatcher) MatchFields(query string, fields ...string) bool {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if normalizedQuery == "" {
		return true
	}

	for _, field := range fields {
		if sm.ContainsIgnoreCase(field, normalizedQuery) {
			return true
		}
	}
	return false
}

// MatchTags performs tag-based search
func (sm *StringMatcher) MatchTags(query string, tags []string) bool {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if normalizedQuery == "" {
		return true
	}

	for _, tag := range tags {
		if sm.ContainsIgnoreCase(tag, normalizedQuery) {
			return true
		}
	}
	return false
}

// NormalizeString normalizes a string by trimming whitespace and converting to lowercase
func (sm *StringMatcher) NormalizeString(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// IsEmpty checks if a string is empty or contains only whitespace
func (sm *StringMatcher) IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// SplitAndTrim splits a string by delimiter and trims each part
func (sm *StringMatcher) SplitAndTrim(s, delimiter string) []string {
	parts := strings.Split(s, delimiter)
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// ToCamelCase converts snake_case or kebab-case to camelCase
func (sm *StringMatcher) ToCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return c == '_' || c == '-' || c == ' '
	})

	if len(words) == 0 {
		return s
	}

	result := strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			result += strings.ToUpper(string(words[i][0])) + strings.ToLower(words[i][1:])
		}
	}

	return result
}

// ToPascalCase converts snake_case or kebab-case to PascalCase
func (sm *StringMatcher) ToPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return c == '_' || c == '-' || c == ' '
	})

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(string(word[0])))
			result.WriteString(strings.ToLower(word[1:]))
		}
	}

	return result.String()
}

// ToSnakeCase converts camelCase or PascalCase to snake_case
func (sm *StringMatcher) ToSnakeCase(s string) string {
	var result strings.Builder

	for i, char := range s {
		if unicode.IsUpper(char) && i > 0 {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(char))
	}

	return result.String()
}

// Global instance for convenience
var globalMatcher = NewStringMatcher()

// Package-level convenience functions
func ContainsIgnoreCase(s, substr string) bool {
	return globalMatcher.ContainsIgnoreCase(s, substr)
}

func ExactMatchIgnoreCase(s1, s2 string) bool {
	return globalMatcher.ExactMatchIgnoreCase(s1, s2)
}

func MatchAny(target string, patterns []string) bool {
	return globalMatcher.MatchAny(target, patterns)
}

func MatchAll(target string, patterns []string) bool {
	return globalMatcher.MatchAll(target, patterns)
}

func MatchFields(query string, fields ...string) bool {
	return globalMatcher.MatchFields(query, fields...)
}

func MatchTags(query string, tags []string) bool {
	return globalMatcher.MatchTags(query, tags)
}

func NormalizeString(s string) string {
	return globalMatcher.NormalizeString(s)
}

func IsEmpty(s string) bool {
	return globalMatcher.IsEmpty(s)
}

func SplitAndTrim(s, delimiter string) []string {
	return globalMatcher.SplitAndTrim(s, delimiter)
}

func ToCamelCase(s string) string {
	return globalMatcher.ToCamelCase(s)
}

func ToPascalCase(s string) string {
	return globalMatcher.ToPascalCase(s)
}

func ToSnakeCase(s string) string {
	return globalMatcher.ToSnakeCase(s)
}
