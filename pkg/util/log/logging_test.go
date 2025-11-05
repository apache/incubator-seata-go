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
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

var testLoggerMutex sync.Mutex

func setupTest(t *testing.T) Logger {
	t.Helper()

	testLoggerMutex.Lock()

	if log == nil {
		Init()
	}

	originalLogger := log

	t.Cleanup(func() {
		log = originalLogger
		testLoggerMutex.Unlock()
	})

	return originalLogger
}

// mockLogger implements Logger interface for testing
type mockLogger struct {
	debugCalls  []string
	infoCalls   []string
	warnCalls   []string
	errorCalls  []string
	panicCalls  []string
	fatalCalls  []string
	shouldPanic bool
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		debugCalls: make([]string, 0),
		infoCalls:  make([]string, 0),
		warnCalls:  make([]string, 0),
		errorCalls: make([]string, 0),
		panicCalls: make([]string, 0),
		fatalCalls: make([]string, 0),
	}
}

func (m *mockLogger) Debug(v ...interface{}) {
	m.debugCalls = append(m.debugCalls, fmt.Sprint(v...))
}

func (m *mockLogger) Debugf(format string, v ...interface{}) {
	m.debugCalls = append(m.debugCalls, fmt.Sprintf(format, v...))
}

func (m *mockLogger) Info(v ...interface{}) {
	m.infoCalls = append(m.infoCalls, fmt.Sprint(v...))
}

func (m *mockLogger) Infof(format string, v ...interface{}) {
	m.infoCalls = append(m.infoCalls, fmt.Sprintf(format, v...))
}

func (m *mockLogger) Warn(v ...interface{}) {
	m.warnCalls = append(m.warnCalls, fmt.Sprint(v...))
}

func (m *mockLogger) Warnf(format string, v ...interface{}) {
	m.warnCalls = append(m.warnCalls, fmt.Sprintf(format, v...))
}

func (m *mockLogger) Error(v ...interface{}) {
	m.errorCalls = append(m.errorCalls, fmt.Sprint(v...))
}

func (m *mockLogger) Errorf(format string, v ...interface{}) {
	m.errorCalls = append(m.errorCalls, fmt.Sprintf(format, v...))
}

func (m *mockLogger) Panic(v ...interface{}) {
	m.panicCalls = append(m.panicCalls, fmt.Sprint(v...))
	if m.shouldPanic {
		panic(fmt.Sprint(v...))
	}
}

func (m *mockLogger) Panicf(format string, v ...interface{}) {
	m.panicCalls = append(m.panicCalls, fmt.Sprintf(format, v...))
	if m.shouldPanic {
		panic(fmt.Sprintf(format, v...))
	}
}

func (m *mockLogger) Fatal(v ...interface{}) {
	m.fatalCalls = append(m.fatalCalls, fmt.Sprint(v...))
}

func (m *mockLogger) Fatalf(format string, v ...interface{}) {
	m.fatalCalls = append(m.fatalCalls, fmt.Sprintf(format, v...))
}

// TestLogLevel tests the LogLevel type and its methods
func TestLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected LogLevel
		wantErr  bool
	}{
		{"debug lowercase", "debug", DebugLevel, false},
		{"debug uppercase", "DEBUG", DebugLevel, false},
		{"info lowercase", "info", InfoLevel, false},
		{"info uppercase", "INFO", InfoLevel, false},
		{"info empty", "", InfoLevel, false},
		{"warn lowercase", "warn", WarnLevel, false},
		{"warn uppercase", "WARN", WarnLevel, false},
		{"error lowercase", "error", ErrorLevel, false},
		{"error uppercase", "ERROR", ErrorLevel, false},
		{"panic lowercase", "panic", PanicLevel, false},
		{"panic uppercase", "PANIC", PanicLevel, false},
		{"fatal lowercase", "fatal", FatalLevel, false},
		{"fatal uppercase", "FATAL", FatalLevel, false},
		{"invalid level", "invalid", 0, true},
		{"mixed case", "DeBuG", DebugLevel, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var level LogLevel
			err := level.UnmarshalText([]byte(tt.input))

			if tt.wantErr {
				assert.Error(t, err, "UnmarshalText() should return error")
				assert.Contains(t, err.Error(), "unrecognized level")
			} else {
				assert.NoError(t, err, "UnmarshalText() should not return error")
				assert.Equal(t, tt.expected, level, "LogLevel should match expected value")
			}
		})
	}
}

func TestLogLevelUnmarshalTextNil(t *testing.T) {
	var level *LogLevel
	err := level.UnmarshalText([]byte("info"))

	assert.Error(t, err, "UnmarshalText() on nil pointer should return error")
	assert.Contains(t, err.Error(), "can't unmarshal a nil *Level")
}

func TestLogLevelValues(t *testing.T) {
	// Test that LogLevel values match zapcore values
	assert.Equal(t, LogLevel(zapcore.DebugLevel), DebugLevel, "DebugLevel value should match")
	assert.Equal(t, LogLevel(zapcore.InfoLevel), InfoLevel, "InfoLevel value should match")
	assert.Equal(t, LogLevel(zapcore.WarnLevel), WarnLevel, "WarnLevel value should match")
	assert.Equal(t, LogLevel(zapcore.ErrorLevel), ErrorLevel, "ErrorLevel value should match")
	assert.Equal(t, LogLevel(zapcore.PanicLevel), PanicLevel, "PanicLevel value should match")
	assert.Equal(t, LogLevel(zapcore.FatalLevel), FatalLevel, "FatalLevel value should match")
}

func TestInit(t *testing.T) {
	setupTest(t)

	// Test Init
	Init()

	// Verify logger is initialized
	assert.NotNil(t, log, "Init() should initialize logger")
	assert.NotNil(t, zapLogger, "Init() should initialize zapLogger")
}

func TestSetLogger(t *testing.T) {
	setupTest(t)

	mock := newMockLogger()
	SetLogger(mock)

	assert.Equal(t, mock, log, "SetLogger() should set the logger")

	// Test that the mock logger is used
	Info("test")
	assert.Len(t, mock.infoCalls, 1, "Mock logger should be called")
}

func TestGetLogger(t *testing.T) {
	setupTest(t)

	mock := newMockLogger()
	SetLogger(mock)

	retrieved := GetLogger()
	assert.Equal(t, mock, retrieved, "GetLogger() should return the current logger")
}

func TestDebugFunctions(t *testing.T) {
	setupTest(t)

	mock := newMockLogger()
	SetLogger(mock)

	Debug("test", "message")
	require.Len(t, mock.debugCalls, 1)
	assert.Equal(t, "testmessage", mock.debugCalls[0])

	Debugf("formatted %s %d", "test", 123)
	require.Len(t, mock.debugCalls, 2)
	assert.Equal(t, "formatted test 123", mock.debugCalls[1])
}

func TestInfoFunctions(t *testing.T) {
	setupTest(t)

	mock := newMockLogger()
	SetLogger(mock)

	Info("test", "message")
	require.Len(t, mock.infoCalls, 1)
	assert.Equal(t, "testmessage", mock.infoCalls[0])

	Infof("formatted %s %d", "test", 123)
	require.Len(t, mock.infoCalls, 2)
	assert.Equal(t, "formatted test 123", mock.infoCalls[1])
}

func TestWarnFunctions(t *testing.T) {
	setupTest(t)

	mock := newMockLogger()
	SetLogger(mock)

	Warn("test", "message")
	require.Len(t, mock.warnCalls, 1)
	assert.Equal(t, "testmessage", mock.warnCalls[0])

	Warnf("formatted %s %d", "test", 123)
	require.Len(t, mock.warnCalls, 2)
	assert.Equal(t, "formatted test 123", mock.warnCalls[1])
}

func TestErrorFunctions(t *testing.T) {
	setupTest(t)

	mock := newMockLogger()
	SetLogger(mock)

	Error("test", "message")
	require.Len(t, mock.errorCalls, 1)
	assert.Equal(t, "testmessage", mock.errorCalls[0])

	Errorf("formatted %s %d", "test", 123)
	require.Len(t, mock.errorCalls, 2)
	assert.Equal(t, "formatted test 123", mock.errorCalls[1])
}

func TestErrorFunctionsWithNilLogger(t *testing.T) {
	setupTest(t)

	// Save stderr
	oldStderr := os.Stderr
	t.Cleanup(func() {
		os.Stderr = oldStderr
	})

	// Redirect stderr to capture output
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Set logger to nil
	log = nil

	// Test Error
	Error("test error")
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "ERROR", "Error output should contain 'ERROR'")
	assert.Contains(t, output, "test error", "Error output should contain the message")
}

func TestErrorfWithNilLogger(t *testing.T) {
	setupTest(t)

	// Save stderr
	oldStderr := os.Stderr
	t.Cleanup(func() {
		os.Stderr = oldStderr
	})

	// Redirect stderr to capture output
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Set logger to nil
	log = nil

	// Test Errorf
	Errorf("formatted %s", "error")
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "ERROR", "Errorf output should contain 'ERROR'")
	assert.Contains(t, output, "formatted error", "Errorf output should contain the formatted message")
}

func TestPanicFunctions(t *testing.T) {
	setupTest(t)

	mock := newMockLogger()
	SetLogger(mock)

	Panic("test", "panic")
	require.Len(t, mock.panicCalls, 1)
	assert.Equal(t, "testpanic", mock.panicCalls[0])

	Panicf("formatted %s", "panic")
	require.Len(t, mock.panicCalls, 2)
	assert.Equal(t, "formatted panic", mock.panicCalls[1])
}

func TestPanicWithNilLogger(t *testing.T) {
	setupTest(t)

	// Set logger to nil
	log = nil

	// Test Panic with nil logger should panic
	assert.Panics(t, func() {
		Panic("test panic")
	}, "Panic() with nil logger should panic")
}

func TestPanicfWithNilLogger(t *testing.T) {
	setupTest(t)

	// Set logger to nil
	log = nil

	// Test Panicf with nil logger should panic
	assert.Panics(t, func() {
		Panicf("test %s", "panic")
	}, "Panicf() with nil logger should panic")
}

func TestFatalFunctions(t *testing.T) {
	setupTest(t)

	mock := newMockLogger()
	SetLogger(mock)

	Fatal("test", "fatal")
	require.Len(t, mock.fatalCalls, 1)
	assert.Equal(t, "testfatal", mock.fatalCalls[0])

	Fatalf("formatted %s", "fatal")
	require.Len(t, mock.fatalCalls, 2)
	assert.Equal(t, "formatted fatal", mock.fatalCalls[1])
}

func TestLoggingWithNilLogger(t *testing.T) {
	setupTest(t)

	// Set logger to nil
	log = nil

	// Test that functions don't panic when logger is nil (except Panic functions)
	assert.NotPanics(t, func() {
		Debug("test")
		Debugf("test %s", "debug")
		Info("test")
		Infof("test %s", "info")
		Warn("test")
		Warnf("test %s", "warn")
	}, "Logging functions should not panic when logger is nil")
}

// mockArrayEncoder implements zapcore.PrimitiveArrayEncoder for testing
type mockArrayEncoder struct {
	elements []interface{}
}

func newMockArrayEncoder() *mockArrayEncoder {
	return &mockArrayEncoder{
		elements: make([]interface{}, 0),
	}
}

func (m *mockArrayEncoder) AppendBool(v bool)              { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendByteString(v []byte)      { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendComplex128(v complex128)  { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendComplex64(v complex64)    { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendFloat64(v float64)        { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendFloat32(v float32)        { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendInt(v int)                { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendInt64(v int64)            { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendInt32(v int32)            { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendInt16(v int16)            { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendInt8(v int8)              { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendString(v string)          { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendUint(v uint)              { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendUint64(v uint64)          { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendUint32(v uint32)          { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendUint16(v uint16)          { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendUint8(v uint8)            { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendUintptr(v uintptr)        { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendDuration(v time.Duration) { m.elements = append(m.elements, v) }
func (m *mockArrayEncoder) AppendTime(v time.Time)         { m.elements = append(m.elements, v) }

func TestEncodeTime(t *testing.T) {
	enc := newMockArrayEncoder()
	testTime := time.Date(2024, 1, 1, 12, 30, 45, 123456789, time.UTC)

	encodeTime(testTime, enc)

	require.Len(t, enc.elements, 1, "Should append one element")
	timeStr, ok := enc.elements[0].(string)
	require.True(t, ok, "Encoded element should be a string")
	assert.Equal(t, "2024-01-01 12:30:45.123 ", timeStr, "Time format should match expected format")
}

func TestEncodeCaller(t *testing.T) {
	enc := newMockArrayEncoder()
	caller := zapcore.EntryCaller{
		Defined: true,
		PC:      0,
		File:    "/path/to/file.go",
		Line:    42,
	}

	encodeCaller(caller, enc)

	require.Len(t, enc.elements, 1, "Should append one element")
	callerStr, ok := enc.elements[0].(string)
	require.True(t, ok, "Encoded element should be a string")
	assert.Len(t, callerStr, 45, "Caller should be formatted to 45 characters")
	assert.Contains(t, callerStr, caller.TrimmedPath(), "Should contain trimmed path")
}

func TestLoggerInterface(t *testing.T) {
	// Verify that mockLogger implements Logger interface
	var _ Logger = (*mockLogger)(nil)

	// Test all interface methods
	mock := newMockLogger()

	mock.Debug("test")
	mock.Debugf("test %s", "format")
	mock.Info("test")
	mock.Infof("test %s", "format")
	mock.Warn("test")
	mock.Warnf("test %s", "format")
	mock.Error("test")
	mock.Errorf("test %s", "format")
	mock.Panic("test")
	mock.Panicf("test %s", "format")
	mock.Fatal("test")
	mock.Fatalf("test %s", "format")

	// Verify calls were recorded
	assert.Len(t, mock.debugCalls, 2, "Debug methods should be called twice")
	assert.Len(t, mock.infoCalls, 2, "Info methods should be called twice")
	assert.Len(t, mock.warnCalls, 2, "Warn methods should be called twice")
	assert.Len(t, mock.errorCalls, 2, "Error methods should be called twice")
	assert.Len(t, mock.panicCalls, 2, "Panic methods should be called twice")
	assert.Len(t, mock.fatalCalls, 2, "Fatal methods should be called twice")
}

func TestZapLoggerConfig(t *testing.T) {
	// Test that zapLoggerConfig is properly initialized
	assert.True(t, zapLoggerConfig.Development, "Development mode should be enabled")
	assert.Equal(t, "console", zapLoggerConfig.Encoding, "Encoding should be 'console'")
	assert.NotEmpty(t, zapLoggerConfig.OutputPaths, "OutputPaths should not be empty")
	assert.NotEmpty(t, zapLoggerConfig.ErrorOutputPaths, "ErrorOutputPaths should not be empty")
	assert.Contains(t, zapLoggerConfig.OutputPaths, "stderr", "OutputPaths should contain stderr")
	assert.Contains(t, zapLoggerConfig.ErrorOutputPaths, "stderr", "ErrorOutputPaths should contain stderr")
}

func TestZapLoggerEncoderConfig(t *testing.T) {
	// Test that zapLoggerEncoderConfig is properly initialized
	assert.Equal(t, "time", zapLoggerEncoderConfig.TimeKey, "TimeKey should be 'time'")
	assert.Equal(t, "level", zapLoggerEncoderConfig.LevelKey, "LevelKey should be 'level'")
	assert.Equal(t, "message", zapLoggerEncoderConfig.MessageKey, "MessageKey should be 'message'")
	assert.Equal(t, "logger", zapLoggerEncoderConfig.NameKey, "NameKey should be 'logger'")
	assert.Equal(t, "caller", zapLoggerEncoderConfig.CallerKey, "CallerKey should be 'caller'")
	assert.Equal(t, "stacktrace", zapLoggerEncoderConfig.StacktraceKey, "StacktraceKey should be 'stacktrace'")
	assert.NotNil(t, zapLoggerEncoderConfig.EncodeTime, "EncodeTime should not be nil")
	assert.NotNil(t, zapLoggerEncoderConfig.EncodeCaller, "EncodeCaller should not be nil")
	assert.NotNil(t, zapLoggerEncoderConfig.EncodeLevel, "EncodeLevel should not be nil")
	assert.NotNil(t, zapLoggerEncoderConfig.EncodeDuration, "EncodeDuration should not be nil")
}

func TestLogTimeFmt(t *testing.T) {
	// Test that the time format constant is correct
	assert.Equal(t, "2006-01-02 15:04:05.000 ", logTmFmt, "logTmFmt should match expected format")

	// Test formatting with the constant
	testTime := time.Date(2024, 12, 25, 13, 45, 30, 999999999, time.UTC)
	formatted := testTime.Format(logTmFmt)

	assert.True(t, strings.HasPrefix(formatted, "2024-12-25 13:45:30.999"), "Time format should be correct")
}

func TestErrorRecovery(t *testing.T) {
	setupTest(t)

	// Create a mock logger that panics
	mock := newMockLogger()
	mock.shouldPanic = true
	SetLogger(mock)

	// Test that Error function recovers from panic
	assert.NotPanics(t, func() {
		Error("test error")
	}, "Error() should recover from panic")

	// Test that Errorf function recovers from panic
	assert.NotPanics(t, func() {
		Errorf("test %s", "error")
	}, "Errorf() should recover from panic")
}

func TestAllLogLevelsWithMock(t *testing.T) {
	setupTest(t)

	mock := newMockLogger()
	SetLogger(mock)

	// Test all log levels
	Debug("debug msg")
	Debugf("debug %s", "formatted")
	Info("info msg")
	Infof("info %s", "formatted")
	Warn("warn msg")
	Warnf("warn %s", "formatted")
	Error("error msg")
	Errorf("error %s", "formatted")
	Panic("panic msg")
	Panicf("panic %s", "formatted")
	Fatal("fatal msg")
	Fatalf("fatal %s", "formatted")

	// Verify all calls
	assert.Equal(t, []string{"debug msg", "debug formatted"}, mock.debugCalls)
	assert.Equal(t, []string{"info msg", "info formatted"}, mock.infoCalls)
	assert.Equal(t, []string{"warn msg", "warn formatted"}, mock.warnCalls)
	assert.Equal(t, []string{"error msg", "error formatted"}, mock.errorCalls)
	assert.Equal(t, []string{"panic msg", "panic formatted"}, mock.panicCalls)
	assert.Equal(t, []string{"fatal msg", "fatal formatted"}, mock.fatalCalls)
}
