package utils

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
	component string
}

// LoggerConfig represents configuration for the logger
type LoggerConfig struct {
	Level     string `json:"level"`
	Format    string `json:"format"` // json, text
	Output    string `json:"output"` // stdout, stderr, file path
	Component string `json:"component"`
}

// NewLogger creates a new logger instance
func NewLogger(config LoggerConfig) *Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	switch strings.ToLower(config.Format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	}

	// Set output
	switch strings.ToLower(config.Output) {
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "stdout", "":
		logger.SetOutput(os.Stdout)
	default:
		// Assume it's a file path
		if file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			logger.SetOutput(file)
		} else {
			logger.SetOutput(os.Stdout)
			logger.WithError(err).Warn("Failed to open log file, using stdout")
		}
	}

	return &Logger{
		Logger:    logger,
		component: config.Component,
	}
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger(component string) *Logger {
	return NewLogger(LoggerConfig{
		Level:     "info",
		Format:    "text",
		Output:    "stdout",
		Component: component,
	})
}

// WithContext adds context information to log entries
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.Logger.WithContext(ctx)
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	return entry
}

// WithComponent creates a new logger entry with component information
func (l *Logger) WithComponent(component string) *logrus.Entry {
	entry := l.Logger.WithField("component", component)
	if l.component != "" {
		entry = entry.WithField("parent_component", l.component)
	}
	return entry
}

// WithFields creates a new logger entry with multiple fields
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	entry := l.Logger.WithFields(fields)
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	return entry
}

// WithField creates a new logger entry with a single field
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	entry := l.Logger.WithField(key, value)
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	return entry
}

// WithError creates a new logger entry with error information
func (l *Logger) WithError(err error) *logrus.Entry {
	entry := l.Logger.WithError(err)
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	return entry
}

// WithCaller adds caller information to log entries
func (l *Logger) WithCaller() *logrus.Entry {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}

	// Get caller information
	if pc, file, line, ok := runtime.Caller(1); ok {
		entry = entry.WithFields(logrus.Fields{
			"caller_file":     filepath.Base(file),
			"caller_line":     line,
			"caller_function": runtime.FuncForPC(pc).Name(),
		})
	}

	return entry
}

// Debug logs a debug message with component information
func (l *Logger) Debug(args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Debug(args...)
}

// Debugf logs a formatted debug message with component information
func (l *Logger) Debugf(format string, args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Debugf(format, args...)
}

// Info logs an info message with component information
func (l *Logger) Info(args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Info(args...)
}

// Infof logs a formatted info message with component information
func (l *Logger) Infof(format string, args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Infof(format, args...)
}

// Warn logs a warning message with component information
func (l *Logger) Warn(args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Warn(args...)
}

// Warnf logs a formatted warning message with component information
func (l *Logger) Warnf(format string, args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Warnf(format, args...)
}

// Error logs an error message with component information
func (l *Logger) Error(args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Error(args...)
}

// Errorf logs a formatted error message with component information
func (l *Logger) Errorf(format string, args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Errorf(format, args...)
}

// Fatal logs a fatal message with component information and exits
func (l *Logger) Fatal(args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Fatal(args...)
}

// Fatalf logs a formatted fatal message with component information and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	entry := l.Logger.WithFields(logrus.Fields{})
	if l.component != "" {
		entry = entry.WithField("component", l.component)
	}
	entry.Fatalf(format, args...)
}

// SetOutput sets the logger output
func (l *Logger) SetOutput(output io.Writer) {
	l.Logger.SetOutput(output)
}

// SetLevel sets the logger level
func (l *Logger) SetLevel(level logrus.Level) {
	l.Logger.SetLevel(level)
}

// GetLevel returns the current logger level
func (l *Logger) GetLevel() logrus.Level {
	return l.Logger.GetLevel()
}

// Clone creates a new logger instance with the same configuration
func (l *Logger) Clone(component string) *Logger {
	newLogger := &Logger{
		Logger:    l.Logger,
		component: component,
	}
	return newLogger
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(config LoggerConfig) {
	globalLogger = NewLogger(config)
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewDefaultLogger("global")
	}
	return globalLogger
}

// GetLogger returns a logger for the specified component
func GetLogger(component string) *Logger {
	if globalLogger == nil {
		globalLogger = NewDefaultLogger("global")
	}
	return globalLogger.Clone(component)
}
