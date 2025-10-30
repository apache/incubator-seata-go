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

package types

import "context"

// Logger provides structured logging interface for A2A framework
type Logger interface {
	// Error logs error level messages
	Error(ctx context.Context, msg string, fields ...interface{})

	// Warn logs warning level messages
	Warn(ctx context.Context, msg string, fields ...interface{})

	// Info logs info level messages
	Info(ctx context.Context, msg string, fields ...interface{})

	// Debug logs debug level messages
	Debug(ctx context.Context, msg string, fields ...interface{})
}

// LogField represents a structured log field
type LogField struct {
	Key   string
	Value interface{}
}

// WithFields creates log fields from key-value pairs
func WithFields(keyValues ...interface{}) []interface{} {
	return keyValues
}

// NoOpLogger is a logger that does nothing
type NoOpLogger struct{}

func (l *NoOpLogger) Error(ctx context.Context, msg string, fields ...interface{}) {}
func (l *NoOpLogger) Warn(ctx context.Context, msg string, fields ...interface{})  {}
func (l *NoOpLogger) Info(ctx context.Context, msg string, fields ...interface{})  {}
func (l *NoOpLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {}

// DefaultLogger is a simple console logger implementation
type DefaultLogger struct{}

func (l *DefaultLogger) Error(ctx context.Context, msg string, fields ...interface{}) {
	l.log("ERROR", msg, fields...)
}

func (l *DefaultLogger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	l.log("WARN", msg, fields...)
}

func (l *DefaultLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	l.log("INFO", msg, fields...)
}

func (l *DefaultLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	l.log("DEBUG", msg, fields...)
}

func (l *DefaultLogger) log(level, msg string, fields ...interface{}) {
	// Simple structured logging format
	output := level + ": " + msg

	if len(fields) > 0 {
		output += " ["
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				if i > 0 {
					output += ", "
				}
				output += fields[i].(string) + "=" + formatValue(fields[i+1])
			}
		}
		output += "]"
	}

	println(output)
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return "?"
	}
}
