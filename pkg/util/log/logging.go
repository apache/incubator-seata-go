package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level represents the level of logging.
type LogLevel int8

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel = LogLevel(zapcore.DebugLevel)
	// InfoLevel is the default logging priority.
	InfoLevel = LogLevel(zapcore.InfoLevel)
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel = LogLevel(zapcore.WarnLevel)
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel = LogLevel(zapcore.ErrorLevel)
	// PanicLevel logs a message, then panics.
	PanicLevel = LogLevel(zapcore.PanicLevel)
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel = LogLevel(zapcore.FatalLevel)
)

type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})

	Info(v ...interface{})
	Infof(format string, v ...interface{})

	Warn(v ...interface{})
	Warnf(format string, v ...interface{})

	Error(v ...interface{})
	Errorf(format string, v ...interface{})

	Panic(v ...interface{})
	Panicf(format string, v ...interface{})

	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
}

var (
	log       Logger
	zapLogger *zap.Logger

	zapLoggerConfig        = zap.NewDevelopmentConfig()
	zapLoggerEncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
)

func init() {
	zapLoggerConfig.EncoderConfig = zapLoggerEncoderConfig
	zapLogger, _ = zapLoggerConfig.Build()
	log = zapLogger.Sugar()
}

// SetLogger: customize yourself logger.
func SetLogger(logger Logger) {
	log = logger
}

// GetLogger get logger
func GetLogger() Logger {
	return log
}

// SetLoggerLevel
func SetLoggerLevel(level LogLevel) error {
	var err error
	zapLoggerConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(level))
	zapLogger, err = zapLoggerConfig.Build()
	if err != nil {
		return err
	}
	log = zapLogger.Sugar()
	return nil
}

// SetLoggerCallerDisable: disable caller info in production env for performance improve.
// It is highly recommended that you execute this method in a production environment.
func SetLoggerCallerDisable() error {
	var err error
	zapLoggerConfig.Development = false
	zapLoggerConfig.DisableCaller = true
	zapLogger, err = zapLoggerConfig.Build()
	if err != nil {
		return err
	}
	log = zapLogger.Sugar()
	return nil
}

// Debug ...
func Debug(v ...interface{}) {
	log.Debug(v...)
}

// Debugf ...
func Debugf(format string, v ...interface{}) {
	log.Debugf(format, v...)
}

// Info ...
func Info(v ...interface{}) {
	log.Info(v...)
}

// Infof ...
func Infof(format string, v ...interface{}) {
	log.Infof(format, v...)
}

// Warn ...
func Warn(v ...interface{}) {
	log.Warn(v...)
}

// Warnf ...
func Warnf(format string, v ...interface{}) {
	log.Warnf(format, v...)
}

// Error ...
func Error(v ...interface{}) {
	log.Error(v...)
}

// Errorf ...
func Errorf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}

// Panic ...
func Panic(v ...interface{}) {
	log.Panic(v...)
}

// Panicf ...
func Panicf(format string, v ...interface{}) {
	log.Panicf(format, v...)
}

// Fatal ...
func Fatal(v ...interface{}) {
	log.Fatal(v...)
}

// Fatalf ...
func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}
