package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel represents logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String returns string representation of log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging capabilities
type Logger struct {
	level      LogLevel
	prefix     string
	logger     *log.Logger
	mutex      sync.RWMutex
	fields     map[string]interface{}
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level  string
	Prefix string
	Fields map[string]interface{}
}

// NewLogger creates a new logger instance
func NewLogger(config LoggerConfig) *Logger {
	level := parseLogLevel(config.Level)
	logger := log.New(os.Stdout, config.Prefix, log.LstdFlags|log.Lshortfile)
	
	fields := make(map[string]interface{})
	if config.Fields != nil {
		for k, v := range config.Fields {
			fields[k] = v
		}
	}
	
	return &Logger{
		level:  level,
		prefix: config.Prefix,
		logger: logger,
		fields: fields,
	}
}

// NewDefaultLogger creates a logger with default settings
func NewDefaultLogger() *Logger {
	return NewLogger(LoggerConfig{
		Level:  "info",
		Prefix: "[AgentHub] ",
		Fields: make(map[string]interface{}),
	})
}

// WithField returns a new logger with an additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	
	return &Logger{
		level:  l.level,
		prefix: l.prefix,
		logger: l.logger,
		fields: newFields,
	}
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	
	return &Logger{
		level:  l.level,
		prefix: l.prefix,
		logger: l.logger,
		fields: newFields,
	}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = parseLogLevel(level)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(DEBUG, msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(INFO, msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(WARN, msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(ERROR, msg, args...)
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, msg string, args ...interface{}) {
	l.mutex.RLock()
	currentLevel := l.level
	fields := l.fields
	l.mutex.RUnlock()
	
	if level < currentLevel {
		return
	}
	
	// Format message
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	
	// Add fields to message
	if len(fields) > 0 {
		var fieldParts []string
		for k, v := range fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		msg = fmt.Sprintf("%s [%s]", msg, strings.Join(fieldParts, " "))
	}
	
	// Add timestamp and level
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] [%s] %s", timestamp, level.String(), msg)
	
	l.logger.Print(logMessage)
}

// parseLogLevel converts string to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// Global logger instance
var globalLogger = NewDefaultLogger()

// SetGlobalLevel sets the global logger level
func SetGlobalLevel(level string) {
	globalLogger.SetLevel(level)
}

// Package-level convenience functions
func Debug(msg string, args ...interface{}) {
	globalLogger.Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	globalLogger.Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	globalLogger.Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	globalLogger.Error(msg, args...)
}

// WithField returns a logger with an additional field
func WithField(key string, value interface{}) *Logger {
	return globalLogger.WithField(key, value)
}

// WithFields returns a logger with additional fields
func WithFields(fields map[string]interface{}) *Logger {
	return globalLogger.WithFields(fields)
}