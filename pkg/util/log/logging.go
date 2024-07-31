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

package log

import (
	"bytes"
	"fmt"
	"time"

	getty "github.com/apache/dubbo-getty"
	"github.com/natefinch/lumberjack"
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

func (l *LogLevel) UnmarshalText(text []byte) error {
	if l == nil {
		return fmt.Errorf("can't unmarshal a nil *Level")
	}
	if !l.unmarshalText(text) && !l.unmarshalText(bytes.ToLower(text)) {
		return fmt.Errorf("unrecognized level: %q", text)
	}
	return nil
}

func (l *LogLevel) unmarshalText(text []byte) bool {
	switch string(text) {
	case "debug", "DEBUG":
		*l = DebugLevel
	case "info", "INFO", "": // make the zero value useful
		*l = InfoLevel
	case "warn", "WARN":
		*l = WarnLevel
	case "error", "ERROR":
		*l = ErrorLevel
	case "panic", "PANIC":
		*l = PanicLevel
	case "fatal", "FATAL":
		*l = FatalLevel
	default:
		return false
	}
	return true
}

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

	zapLoggerConfig = zap.Config{
		// todo read level from config
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	zapLoggerEncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     encodeTime,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   encodeCaller,
	}
)

const (
	logTmFmt = "2006-01-02 15:04:05.000 "
)

func encodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(logTmFmt))
}

func encodeCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	// enc.AppendString(fmt.Sprintf("\033[34m%s\033[0m", caller.TrimmedPath()))
	enc.AppendString(fmt.Sprintf("%-45s", caller.TrimmedPath()))
}

func Init() {
	zapLoggerConfig.EncoderConfig = zapLoggerEncoderConfig
	zapLogger, _ = zapLoggerConfig.Build(zap.AddCallerSkip(1))
	log = zapLogger.Sugar()
	getty.SetLogger(log)
}

func InitWithOption(logPath string, level LogLevel) {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}
	syncer := zapcore.AddSync(lumberJackLogger)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = encodeTime
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeCaller = encodeCaller

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, syncer, zap.NewAtomicLevelAt(zapcore.Level(level)))
	zapLogger = zap.New(core, zap.AddCaller())

	log = zapLogger.Sugar()
	getty.SetLogger(log)
}

// SetLogger: customize yourself logger.
func SetLogger(logger Logger) {
	log = logger
}

// GetLogger get logger
func GetLogger() Logger {
	return log
}

// Debug ...
func Debug(v ...interface{}) {
	if log == nil {
		return
	}
	log.Debug(v...)
}

// Debugf ...
func Debugf(format string, v ...interface{}) {
	if log == nil {
		return
	}
	log.Debugf(format, v...)
}

// Info ...
func Info(v ...interface{}) {
	if log == nil {
		return
	}
	log.Info(v...)
}

// Infof ...
func Infof(format string, v ...interface{}) {
	if log == nil {
		return
	}
	log.Infof(format, v...)
}

// Warn ...
func Warn(v ...interface{}) {
	if log == nil {
		return
	}
	log.Warn(v...)
}

// Warnf ...
func Warnf(format string, v ...interface{}) {
	if log == nil {
		return
	}
	log.Warnf(format, v...)
}

// Error ...
func Error(v ...interface{}) {
	if log == nil {
		return
	}
	log.Error(v...)
}

// Errorf ...
func Errorf(format string, v ...interface{}) {
	if log == nil {
		return
	}
	log.Errorf(format, v...)
}

// Panic ...
func Panic(v ...interface{}) {
	if log == nil {
		return
	}
	log.Panic(v...)
}

// Panicf ...
func Panicf(format string, v ...interface{}) {
	if log == nil {
		return
	}
	log.Panicf(format, v...)
}

// Fatal ...
func Fatal(v ...interface{}) {
	if log == nil {
		return
	}
	log.Fatal(v...)
}

// Fatalf ...
func Fatalf(format string, v ...interface{}) {
	if log == nil {
		return
	}
	log.Fatalf(format, v...)
}
