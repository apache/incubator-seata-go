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

package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Logger provides structured logging for the agent
type Logger struct {
	level     Level
	output    io.Writer
	formatter Formatter
	fields    map[string]interface{}
	mu        sync.RWMutex
	component string
}

// Level represents log levels
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "unknown"
	}
}

// ParseLevel parses a string level
func ParseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// Formatter interface for log formatting
type Formatter interface {
	Format(entry *Entry) ([]byte, error)
}

// Entry represents a log entry
type Entry struct {
	Level     Level                  `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Component string                 `json:"component,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// JSONFormatter formats logs as JSON
type JSONFormatter struct{}

// Format implements the Formatter interface
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// TextFormatter formats logs as plain text
type TextFormatter struct{}

// Format implements the Formatter interface
func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	var builder strings.Builder

	// Timestamp
	builder.WriteString(entry.Timestamp.Format("2006-01-02 15:04:05"))

	// Level
	builder.WriteString(" [")
	builder.WriteString(strings.ToUpper(entry.Level.String()))
	builder.WriteString("]")

	// Component
	if entry.Component != "" {
		builder.WriteString(" [")
		builder.WriteString(entry.Component)
		builder.WriteString("]")
	}

	// Session ID
	if entry.SessionID != "" {
		builder.WriteString(" [")
		builder.WriteString(entry.SessionID)
		builder.WriteString("]")
	}

	// Message
	builder.WriteString(" ")
	builder.WriteString(entry.Message)

	// Fields
	if len(entry.Fields) > 0 {
		builder.WriteString(" |")
		for key, value := range entry.Fields {
			builder.WriteString(fmt.Sprintf(" %s=%v", key, value))
		}
	}

	builder.WriteString("\n")
	return []byte(builder.String()), nil
}

// Config holds logger configuration
type Config struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	EnableFile bool   `json:"enable_file"`
	FilePath   string `json:"file_path"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`
	Component  string `json:"component"`
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		EnableFile: false,
		FilePath:   "/var/log/seata-agent.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Component:  "agent",
	}
}

// NewLogger creates a new logger
func NewLogger(config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Determine output writer
	var output io.Writer
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "file":
		if config.FilePath == "" {
			return nil, fmt.Errorf("file path required for file output")
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	default:
		output = os.Stdout
	}

	// Determine formatter
	var formatter Formatter
	switch config.Format {
	case "json":
		formatter = &JSONFormatter{}
	case "text":
		formatter = &TextFormatter{}
	default:
		formatter = &JSONFormatter{}
	}

	logger := &Logger{
		level:     ParseLevel(config.Level),
		output:    output,
		formatter: formatter,
		fields:    make(map[string]interface{}),
		component: config.Component,
	}

	return logger, nil
}

// WithComponent creates a logger with a specific component
func (l *Logger) WithComponent(component string) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &Logger{
		level:     l.level,
		output:    l.output,
		formatter: l.formatter,
		fields:    copyFields(l.fields),
		component: component,
	}
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	fields := copyFields(l.fields)
	fields[key] = value

	return &Logger{
		level:     l.level,
		output:    l.output,
		formatter: l.formatter,
		fields:    fields,
		component: l.component,
	}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := copyFields(l.fields)
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		level:     l.level,
		output:    l.output,
		formatter: l.formatter,
		fields:    newFields,
		component: l.component,
	}
}

// WithSessionID adds session ID to the logger
func (l *Logger) WithSessionID(sessionID string) *Logger {
	return l.WithField("session_id", sessionID)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, args ...interface{}) {
	if l.level <= DebugLevel {
		l.log(DebugLevel, fmt.Sprintf(message, args...))
	}
}

// Info logs an info message
func (l *Logger) Info(message string, args ...interface{}) {
	if l.level <= InfoLevel {
		l.log(InfoLevel, fmt.Sprintf(message, args...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(message string, args ...interface{}) {
	if l.level <= WarnLevel {
		l.log(WarnLevel, fmt.Sprintf(message, args...))
	}
}

// Error logs an error message
func (l *Logger) Error(message string, args ...interface{}) {
	if l.level <= ErrorLevel {
		l.log(ErrorLevel, fmt.Sprintf(message, args...))
	}
}

// ErrorWithError logs an error with an error object
func (l *Logger) ErrorWithError(err error, message string, args ...interface{}) {
	if l.level <= ErrorLevel {
		logger := l.WithField("error", err.Error())
		logger.log(ErrorLevel, fmt.Sprintf(message, args...))
	}
}

// log performs the actual logging
func (l *Logger) log(level Level, message string) {
	l.mu.RLock()
	fields := copyFields(l.fields)
	component := l.component
	l.mu.RUnlock()

	entry := &Entry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Component: component,
		Fields:    fields,
	}

	// Extract session ID from fields if present
	if sessionID, ok := fields["session_id"].(string); ok {
		entry.SessionID = sessionID
		delete(entry.Fields, "session_id") // Remove from fields to avoid duplication
	}

	data, err := l.formatter.Format(entry)
	if err != nil {
		// Fallback to standard logger
		log.Printf("Failed to format log entry: %v", err)
		return
	}

	l.output.Write(data)
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// copyFields creates a deep copy of fields map
func copyFields(fields map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range fields {
		copy[k] = v
	}
	return copy
}

// Global logger instance
var defaultLogger *Logger

// Initialize initializes the global logger
func Initialize(config *Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// GetLogger returns the global logger or creates a default one
func GetLogger() *Logger {
	if defaultLogger == nil {
		defaultLogger, _ = NewLogger(DefaultConfig())
	}
	return defaultLogger
}

// Global logging functions
func Debug(message string, args ...interface{}) {
	GetLogger().Debug(message, args...)
}

func Info(message string, args ...interface{}) {
	GetLogger().Info(message, args...)
}

func Warn(message string, args ...interface{}) {
	GetLogger().Warn(message, args...)
}

func Error(message string, args ...interface{}) {
	GetLogger().Error(message, args...)
}

func ErrorWithError(err error, message string, args ...interface{}) {
	GetLogger().ErrorWithError(err, message, args...)
}

// Component-specific loggers
func WithComponent(component string) *Logger {
	return GetLogger().WithComponent(component)
}

func WithField(key string, value interface{}) *Logger {
	return GetLogger().WithField(key, value)
}

func WithFields(fields map[string]interface{}) *Logger {
	return GetLogger().WithFields(fields)
}

func WithSessionID(sessionID string) *Logger {
	return GetLogger().WithSessionID(sessionID)
}

// LoggingMiddleware can be used to add logging to functions
type LoggingMiddleware struct {
	logger *Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger *Logger) *LoggingMiddleware {
	return &LoggingMiddleware{logger: logger}
}

// WrapFunction wraps a function with logging
func (m *LoggingMiddleware) WrapFunction(name string, fn func() error) error {
	start := time.Now()
	m.logger.Debug("Function started: %s", name)

	err := fn()
	duration := time.Since(start)

	if err != nil {
		m.logger.ErrorWithError(err, "Function failed: %s (duration: %v)", name, duration)
	} else {
		m.logger.Debug("Function completed: %s (duration: %v)", name, duration)
	}

	return err
}
