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

package internal

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"seata-go-ai-a2a/pkg/types"
)

// AuthAttemptEvent represents an authentication attempt
type AuthAttemptEvent struct {
	Timestamp  time.Time         `json:"timestamp"`
	Method     string            `json:"method"`
	UserID     string            `json:"user_id,omitempty"`
	RemoteAddr string            `json:"remote_addr"`
	UserAgent  string            `json:"user_agent,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// AuthFailureEvent represents a failed authentication
type AuthFailureEvent struct {
	AuthAttemptEvent
	Reason string `json:"reason"`
	Error  string `json:"error,omitempty"`
}

// AuthSuccessEvent represents a successful authentication
type AuthSuccessEvent struct {
	AuthAttemptEvent
	Scopes []string `json:"scopes,omitempty"`
}

// DefaultAuditLogger provides a comprehensive audit logging implementation
type DefaultAuditLogger struct {
	logger    types.Logger
	config    *AuditLoggerConfig
	eventChan chan *AuditLogEntry
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// AuditLoggerConfig configures the audit logger
type AuditLoggerConfig struct {
	Enabled           bool          `json:"enabled"`
	BufferSize        int           `json:"buffer_size"`
	FlushInterval     time.Duration `json:"flush_interval"`
	LogLevel          string        `json:"log_level"`
	IncludeMetadata   bool          `json:"include_metadata"`
	IncludeStackTrace bool          `json:"include_stack_trace"`
}

// AuditLogEntry represents a complete audit log entry
type AuditLogEntry struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Type       string                 `json:"type"` // "attempt", "success", "failure"
	Event      interface{}            `json:"event"`
	Context    map[string]interface{} `json:"context,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
}

// DefaultAuditLoggerConfig returns a default configuration
func DefaultAuditLoggerConfig() *AuditLoggerConfig {
	return &AuditLoggerConfig{
		Enabled:           true,
		BufferSize:        1000,
		FlushInterval:     time.Second * 10,
		LogLevel:          "INFO",
		IncludeMetadata:   true,
		IncludeStackTrace: false,
	}
}

// NewDefaultAuditLogger creates a new audit logger
func NewDefaultAuditLogger(logger types.Logger, config *AuditLoggerConfig) *DefaultAuditLogger {
	if config == nil {
		config = DefaultAuditLoggerConfig()
	}
	if logger == nil {
		logger = &types.DefaultLogger{}
	}

	al := &DefaultAuditLogger{
		logger:    logger,
		config:    config,
		eventChan: make(chan *AuditLogEntry, config.BufferSize),
		stopChan:  make(chan struct{}),
	}

	// Start background logging goroutine
	if config.Enabled {
		al.wg.Add(1)
		go al.processEvents()
	}

	return al
}

// LogAuthAttempt logs an authentication attempt
func (al *DefaultAuditLogger) LogAuthAttempt(ctx context.Context, event interface{}) {
	if !al.config.Enabled {
		return
	}

	entry := &AuditLogEntry{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		Type:      "attempt",
		Event:     event,
	}

	if al.config.IncludeMetadata {
		entry.Context = al.extractContextInfo(ctx)
	}

	select {
	case al.eventChan <- entry:
	default:
		// Channel is full, log directly to avoid blocking
		al.logEntry(entry)
	}
}

// LogAuthFailure logs an authentication failure
func (al *DefaultAuditLogger) LogAuthFailure(ctx context.Context, event interface{}) {
	if !al.config.Enabled {
		return
	}

	entry := &AuditLogEntry{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		Type:      "failure",
		Event:     event,
	}

	if al.config.IncludeMetadata {
		entry.Context = al.extractContextInfo(ctx)
	}

	if al.config.IncludeStackTrace {
		entry.StackTrace = getStackTrace()
	}

	select {
	case al.eventChan <- entry:
	default:
		// Channel is full, log directly to avoid blocking
		al.logEntry(entry)
	}
}

// LogAuthSuccess logs a successful authentication
func (al *DefaultAuditLogger) LogAuthSuccess(ctx context.Context, event interface{}) {
	if !al.config.Enabled {
		return
	}

	entry := &AuditLogEntry{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		Type:      "success",
		Event:     event,
	}

	if al.config.IncludeMetadata {
		entry.Context = al.extractContextInfo(ctx)
	}

	select {
	case al.eventChan <- entry:
	default:
		// Channel is full, log directly to avoid blocking
		al.logEntry(entry)
	}
}

// Close cleans up resources
func (al *DefaultAuditLogger) Close() error {
	if !al.config.Enabled {
		return nil
	}

	close(al.stopChan)
	al.wg.Wait()
	close(al.eventChan)

	// Process any remaining events
	for entry := range al.eventChan {
		al.logEntry(entry)
	}

	return nil
}

// processEvents processes audit log events in background
func (al *DefaultAuditLogger) processEvents() {
	defer al.wg.Done()

	ticker := time.NewTicker(al.config.FlushInterval)
	defer ticker.Stop()

	batch := make([]*AuditLogEntry, 0, 100)

	for {
		select {
		case entry, ok := <-al.eventChan:
			if !ok {
				// Channel closed, process remaining batch
				if len(batch) > 0 {
					al.processBatch(batch)
				}
				return
			}

			batch = append(batch, entry)

			// Process batch if full
			if len(batch) >= 100 {
				al.processBatch(batch)
				batch = batch[:0] // Reset slice
			}

		case <-ticker.C:
			// Flush on interval
			if len(batch) > 0 {
				al.processBatch(batch)
				batch = batch[:0] // Reset slice
			}

		case <-al.stopChan:
			// Process remaining entries
			if len(batch) > 0 {
				al.processBatch(batch)
			}
			return
		}
	}
}

// processBatch processes a batch of audit log entries
func (al *DefaultAuditLogger) processBatch(entries []*AuditLogEntry) {
	for _, entry := range entries {
		al.logEntry(entry)
	}
}

// logEntry logs a single audit log entry
func (al *DefaultAuditLogger) logEntry(entry *AuditLogEntry) {
	// Convert entry to JSON for structured logging
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		al.logger.Error(context.Background(), "Failed to marshal audit log entry", "error", err)
		return
	}

	// Log based on type
	switch entry.Type {
	case "failure":
		al.logger.Warn(context.Background(), "Authentication failure", "audit_entry", string(entryJSON))
	case "success":
		al.logger.Info(context.Background(), "Authentication success", "audit_entry", string(entryJSON))
	case "attempt":
		al.logger.Debug(context.Background(), "Authentication attempt", "audit_entry", string(entryJSON))
	default:
		al.logger.Info(context.Background(), "Authentication event", "audit_entry", string(entryJSON))
	}
}

// extractContextInfo extracts relevant information from context
func (al *DefaultAuditLogger) extractContextInfo(ctx context.Context) map[string]interface{} {
	info := make(map[string]interface{})

	// Extract request ID if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		info["request_id"] = requestID
	}

	// Extract trace ID if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		info["trace_id"] = traceID
	}

	// Extract user agent if available
	if userAgent := ctx.Value("user_agent"); userAgent != nil {
		info["user_agent"] = userAgent
	}

	// Extract IP address if available
	if remoteAddr := ctx.Value("remote_addr"); remoteAddr != nil {
		info["remote_addr"] = remoteAddr
	}

	return info
}

// generateEventID generates a unique event ID
func generateEventID() string {
	// Simple timestamp-based ID - in production you might want UUIDs
	return time.Now().Format("20060102150405.000000")
}

// getStackTrace returns the current stack trace
func getStackTrace() string {
	// Placeholder - in production you might use runtime.Stack() or similar
	return "stack trace not implemented"
}

// FileAuditLogger provides file-based audit logging
type FileAuditLogger struct {
	file     *os.File
	encoder  *json.Encoder
	mu       sync.Mutex
	filePath string
}

// NewFileAuditLogger creates a new file-based audit logger
func NewFileAuditLogger(filePath string) (*FileAuditLogger, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}

	return &FileAuditLogger{
		file:     file,
		encoder:  json.NewEncoder(file),
		filePath: filePath,
	}, nil
}

// LogAuthAttempt logs an authentication attempt to file
func (fal *FileAuditLogger) LogAuthAttempt(ctx context.Context, event interface{}) {
	fal.mu.Lock()
	defer fal.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now(),
		"type":      "attempt",
		"event":     event,
	}

	fal.encoder.Encode(entry)
}

// LogAuthFailure logs an authentication failure to file
func (fal *FileAuditLogger) LogAuthFailure(ctx context.Context, event interface{}) {
	fal.mu.Lock()
	defer fal.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now(),
		"type":      "failure",
		"event":     event,
	}

	fal.encoder.Encode(entry)
}

// LogAuthSuccess logs a successful authentication to file
func (fal *FileAuditLogger) LogAuthSuccess(ctx context.Context, event interface{}) {
	fal.mu.Lock()
	defer fal.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now(),
		"type":      "success",
		"event":     event,
	}

	fal.encoder.Encode(entry)
}

// Close closes the file
func (fal *FileAuditLogger) Close() error {
	fal.mu.Lock()
	defer fal.mu.Unlock()

	return fal.file.Close()
}
