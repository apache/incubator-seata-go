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

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"seata-go-ai-workflow-agent/internal/orchestration"
	"seata-go-ai-workflow-agent/pkg/logger"
	"seata-go-ai-workflow-agent/pkg/session"
)

// StreamingHandler handles session-based streaming requests
type StreamingHandler struct {
	streamingOrchestrator *orchestration.StreamingOrchestrator
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler(streamingOrchestrator *orchestration.StreamingOrchestrator) *StreamingHandler {
	return &StreamingHandler{
		streamingOrchestrator: streamingOrchestrator,
	}
}

// CreateSessionRequest represents request to create a new session
type CreateSessionRequest struct {
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// CreateSessionResponse represents response for session creation
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// SessionStatusResponse represents session status
type SessionStatusResponse struct {
	ID          string                `json:"id"`
	Description string                `json:"description"`
	Status      session.SessionStatus `json:"status"`
	CreatedAt   string                `json:"created_at"`
	UpdatedAt   string                `json:"updated_at"`
	EventCount  int                   `json:"event_count"`
}

// HandleCreateSession creates a new orchestration session
func (sh *StreamingHandler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	if req.Description == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Description is required",
		})
		return
	}

	// Create session
	sessionMgr := sh.streamingOrchestrator.GetSessionManager()
	sess := sessionMgr.CreateSession(req.Description, req.Context)

	logger.WithField("session_id", sess.ID).Info("session created")

	// Start orchestration in background
	go func() {
		// Use context.Background() instead of r.Context() to avoid cancellation
		// when the HTTP request completes
		ctx := context.Background()

		scenarioReq := orchestration.ScenarioRequest{
			Description: req.Description,
			Context:     req.Context,
		}

		_, err := sh.streamingOrchestrator.OrchestrateWithSession(ctx, sess, scenarioReq)
		if err != nil {
			logger.WithField("session_id", sess.ID).WithField("error", err).Error("orchestration failed")
		}
	}()

	respondJSON(w, http.StatusCreated, CreateSessionResponse{
		SessionID: sess.ID,
		Message:   "Session created and orchestration started",
	})
}

// HandleSessionStatus returns the current status of a session
func (sh *StreamingHandler) HandleSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Session ID is required",
		})
		return
	}

	sessionMgr := sh.streamingOrchestrator.GetSessionManager()
	sess, exists := sessionMgr.GetSession(sessionID)
	if !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "Session not found",
		})
		return
	}

	snapshot := sess.GetSnapshot()
	respondJSON(w, http.StatusOK, SessionStatusResponse{
		ID:          snapshot.ID,
		Description: snapshot.Description,
		Status:      snapshot.Status,
		CreatedAt:   snapshot.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   snapshot.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		EventCount:  snapshot.EventCount,
	})
}

// HandleListSessions returns all active sessions
func (sh *StreamingHandler) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	sessionMgr := sh.streamingOrchestrator.GetSessionManager()
	sessions := sessionMgr.ListSessions()

	snapshots := make([]SessionStatusResponse, 0, len(sessions))
	for _, sess := range sessions {
		snapshot := sess.GetSnapshot()
		snapshots = append(snapshots, SessionStatusResponse{
			ID:          snapshot.ID,
			Description: snapshot.Description,
			Status:      snapshot.Status,
			CreatedAt:   snapshot.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   snapshot.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			EventCount:  snapshot.EventCount,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": snapshots,
		"count":    len(snapshots),
	})
}

// HandleStreamProgress streams progress events via Server-Sent Events
func (sh *StreamingHandler) HandleStreamProgress(w http.ResponseWriter, r *http.Request) {
	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Session ID is required",
		})
		return
	}

	sessionMgr := sh.streamingOrchestrator.GetSessionManager()
	sess, exists := sessionMgr.GetSession(sessionID)
	if !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "Session not found",
		})
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to progress events
	subscriberID, eventChan := sess.Subscribe()
	defer sess.Unsubscribe(subscriberID)

	logger.WithField("session_id", sessionID).WithField("subscriber_id", subscriberID).Info("SSE client connected")

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"message\": \"Connected to session %s\"}\n\n", sessionID)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream events
	for {
		select {
		case <-r.Context().Done():
			logger.WithField("session_id", sessionID).Info("SSE client disconnected")
			return
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed
				fmt.Fprintf(w, "event: close\ndata: {\"message\": \"Session closed\"}\n\n")
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				return
			}

			// Marshal event to JSON
			eventData, err := json.Marshal(event)
			if err != nil {
				logger.WithField("error", err).Error("failed to marshal event")
				continue
			}

			// Send SSE event
			fmt.Fprintf(w, "event: progress\ndata: %s\n\n", string(eventData))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			// If completed or failed, send close event
			if event.Status == session.StatusCompleted || event.Status == session.StatusFailed {
				logger.WithField("session_id", sessionID).WithField("status", event.Status).Info("session finished")
				// Keep connection open for a bit to ensure client receives final event
				// Client will close on receiving completed/failed status
			}
		}
	}
}

// HandleSessionResult returns the final result of a completed session
func (sh *StreamingHandler) HandleSessionResult(w http.ResponseWriter, r *http.Request) {
	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Session ID is required",
		})
		return
	}

	sessionMgr := sh.streamingOrchestrator.GetSessionManager()
	sess, exists := sessionMgr.GetSession(sessionID)
	if !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "Session not found",
		})
		return
	}

	// Get session result
	resultInterface := sess.GetResult()

	if resultInterface == nil {
		snapshot := sess.GetSnapshot()
		respondJSON(w, http.StatusAccepted, map[string]string{
			"message": "Session is still in progress",
			"status":  string(snapshot.Status),
		})
		return
	}

	// Type assert to orchestration result
	result, ok := resultInterface.(*orchestration.OrchestrationResult)
	if !ok {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Invalid result type",
		})
		return
	}

	// Return result
	resp := OrchestrationResponse{
		Success:      result.Success,
		Message:      result.Message,
		RetryCount:   result.RetryCount,
		Capabilities: result.Capabilities,
	}

	if result.SeataWorkflow != nil && result.ReactFlow != nil {
		resp.Workflow = &WorkflowResult{
			SeataWorkflow: result.SeataWorkflow,
			ReactFlow:     result.ReactFlow,
		}
	}

	if !result.Success {
		resp.Error = result.Message
	}

	respondJSON(w, http.StatusOK, resp)
}

// HandleSessionHistory returns complete session history including all events
func (sh *StreamingHandler) HandleSessionHistory(w http.ResponseWriter, r *http.Request) {
	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Session ID is required",
		})
		return
	}

	sessionMgr := sh.streamingOrchestrator.GetSessionManager()
	sess, exists := sessionMgr.GetSession(sessionID)
	if !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "Session not found",
		})
		return
	}

	// Get complete session history
	history := sess.GetCompleteHistory()

	respondJSON(w, http.StatusOK, history)
}

// extractSessionID extracts session ID from URL path
// Expected format: /api/v1/sessions/{session_id}/...
func extractSessionID(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "sessions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
