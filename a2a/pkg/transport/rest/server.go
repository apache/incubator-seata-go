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

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"seata-go-ai-a2a/pkg/handler"
	"seata-go-ai-a2a/pkg/types"
)

// HandlerFunc represents a REST API handler function
type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// Server represents a RESTful HTTP server for A2A protocol
type Server struct {
	mux                     *http.ServeMux
	agentCard               *types.AgentCard
	taskManager             types.TaskManagerInterface
	pushNotificationConfigs map[string]*types.TaskPushNotificationConfig
	pushConfigMu            sync.RWMutex
}

// NewServer creates a new REST server
func NewServer() *Server {
	return &Server{
		mux:                     http.NewServeMux(),
		pushNotificationConfigs: make(map[string]*types.TaskPushNotificationConfig),
	}
}

// RegisterA2AHandlers registers REST handlers for A2A protocol endpoints
func (s *Server) RegisterA2AHandlers(agentCard *types.AgentCard, taskManager types.TaskManagerInterface) {
	s.agentCard = agentCard
	s.taskManager = taskManager

	// Register A2A REST endpoints according to specification
	s.mux.HandleFunc("/api/v1/messages", s.handleMessages)
	s.mux.HandleFunc("/api/v1/tasks", s.handleTasks)
	s.mux.HandleFunc("/api/v1/tasks/", s.handleTaskOperations)
	s.mux.HandleFunc("/api/v1/agent-card", s.handleAgentCard)
	s.mux.HandleFunc("/api/v1/push-notification-configs", s.handlePushNotificationConfigs)
	s.mux.HandleFunc("/api/v1/push-notification-configs/", s.handlePushNotificationConfigOperations)
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight OPTIONS requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set common headers
	w.Header().Set("Content-Type", "application/json")

	s.mux.ServeHTTP(w, r)
}

// handleMessages handles /api/v1/messages endpoint
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleSendMessage(w, r)
	default:
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only POST is supported for messages endpoint")
	}
}

// handleSendMessage handles POST /api/v1/messages (message/send)
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Parse SendMessageRequest
	var req types.SendMessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate the request
	if req.Request == nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing message in request", "Request must contain a message")
		return
	}

	// Extract context ID from message or generate a new one
	contextID := req.Request.ContextID
	if contextID == "" {
		contextID = fmt.Sprintf("ctx_%s", uuid.New().String())
	}

	// Set context ID in the message if not already set
	if req.Request.ContextID == "" {
		req.Request.ContextID = contextID
	}

	// Check if task manager has handler interfaces
	if handlerTaskManager, ok := interface{}(s.taskManager).(interface {
		ProcessMessage(ctx context.Context, req *handler.MessageRequest) (*handler.MessageResponse, error)
	}); ok {
		// Use handler interface for processing
		s.handleSendMessageWithHandler(w, r, &req, handlerTaskManager)
		return
	}

	// Fallback to legacy direct task creation
	task, err := s.taskManager.CreateTask(r.Context(), contextID, req.Request, req.Metadata)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create task", err.Error())
		return
	}

	// If blocking mode is requested and the task is still submitted,
	// update it to working state for demonstration
	if req.Configuration != nil && req.Configuration.Blocking {
		err = s.taskManager.UpdateTaskStatus(r.Context(), task.ID, types.TaskStateWorking, nil)
		if err != nil {
			// Log error but don't fail the request
		}
	}

	response := &types.TaskSendMessageResponse{Task: task}
	s.writeJSONResponse(w, http.StatusOK, response)
}

// handleTasks handles /api/v1/tasks endpoint
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListTasks(w, r)
	default:
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only GET is supported for tasks endpoint")
	}
}

// handleListTasks handles GET /api/v1/tasks (tasks/list)
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	contextID := r.URL.Query().Get("contextId")
	states := r.URL.Query()["state"]

	// Convert state strings to TaskState enums
	var stateEnums []types.TaskState
	for _, stateStr := range states {
		if state, err := types.ParseTaskState(stateStr); err == nil {
			stateEnums = append(stateEnums, state)
		}
	}

	req := &types.ListTasksRequest{
		ContextID: contextID,
		States:    stateEnums,
	}

	// List tasks using the task manager
	response, err := s.taskManager.ListTasks(r.Context(), req)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list tasks", err.Error())
		return
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// handleTaskOperations handles /api/v1/tasks/{taskId} endpoint
func (s *Server) handleTaskOperations(w http.ResponseWriter, r *http.Request) {
	// Extract task ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	taskID := strings.Split(path, "/")[0]

	if taskID == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing task ID", "Task ID is required in the URL path")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetTask(w, r, taskID)
	case http.MethodDelete:
		s.handleCancelTask(w, r, taskID)
	default:
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only GET and DELETE are supported for task operations")
	}
}

// handleGetTask handles GET /api/v1/tasks/{taskId} (tasks/get)
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, taskID string) {
	// Get task using the task manager
	task, err := s.taskManager.GetTask(r.Context(), taskID)
	if err != nil {
		// Check if it's an A2A error
		if a2aErr, ok := err.(*types.A2AError); ok {
			switch a2aErr.Code {
			case types.TaskNotFoundError:
				s.writeErrorResponse(w, http.StatusNotFound, "Task not found", a2aErr.Message)
			default:
				s.writeErrorResponse(w, http.StatusBadRequest, "Task error", a2aErr.Message)
			}
			return
		}
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get task", err.Error())
		return
	}

	s.writeJSONResponse(w, http.StatusOK, task)
}

// handleCancelTask handles DELETE /api/v1/tasks/{taskId} (tasks/cancel)
func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request, taskID string) {
	// Cancel task using the task manager
	task, err := s.taskManager.CancelTask(r.Context(), taskID)
	if err != nil {
		// Check if it's an A2A error
		if a2aErr, ok := err.(*types.A2AError); ok {
			switch a2aErr.Code {
			case types.TaskNotFoundError:
				s.writeErrorResponse(w, http.StatusNotFound, "Task not found", a2aErr.Message)
			case types.TaskNotCancelableError:
				s.writeErrorResponse(w, http.StatusConflict, "Task not cancelable", a2aErr.Message)
			default:
				s.writeErrorResponse(w, http.StatusBadRequest, "Task error", a2aErr.Message)
			}
			return
		}
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to cancel task", err.Error())
		return
	}

	s.writeJSONResponse(w, http.StatusOK, task)
}

// handleAgentCard handles /api/v1/agent-card endpoint
func (s *Server) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var agentCard *types.AgentCard

		// Check if task manager has agent card provider
		if cardProvider, ok := interface{}(s.taskManager).(interface {
			GetAgentCard(ctx context.Context) (*types.AgentCard, error)
		}); ok {
			// Use provider interface
			card, err := cardProvider.GetAgentCard(r.Context())
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get agent card from provider", err.Error())
				return
			}
			agentCard = card
		} else {
			// Fallback to static agent card
			agentCard = s.agentCard
		}

		if agentCard == nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, "No agent card available", "Agent card is not configured")
			return
		}

		s.writeJSONResponse(w, http.StatusOK, agentCard)
	default:
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only GET is supported for agent-card endpoint")
	}
}

// handlePushNotificationConfigs handles /api/v1/push-notification-configs endpoint
func (s *Server) handlePushNotificationConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreatePushNotificationConfig(w, r)
	case http.MethodGet:
		s.handleListPushNotificationConfigs(w, r)
	default:
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only POST and GET are supported for push-notification-configs endpoint")
	}
}

// handleCreatePushNotificationConfig handles POST /api/v1/push-notification-configs
func (s *Server) handleCreatePushNotificationConfig(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Parse CreateTaskPushNotificationConfigRequest
	var req types.CreateTaskPushNotificationConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate parent format (tasks/{task_id})
	if !strings.HasPrefix(req.Parent, "tasks/") {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid parent format", "Parent must be in format tasks/{task_id}")
		return
	}

	taskID := strings.TrimPrefix(req.Parent, "tasks/")
	if taskID == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing task ID", "Task ID is required in parent")
		return
	}

	// Verify task exists
	_, err = s.taskManager.GetTask(r.Context(), taskID)
	if err != nil {
		if a2aErr, ok := err.(*types.A2AError); ok && a2aErr.Code == types.TaskNotFoundError {
			s.writeErrorResponse(w, http.StatusNotFound, "Task not found", "The specified task does not exist")
		} else {
			s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to verify task", err.Error())
		}
		return
	}

	// Generate config ID if not provided
	configID := req.ConfigID
	if configID == "" {
		configID = uuid.New().String()
	}

	// Generate full name for the config
	configName := fmt.Sprintf("tasks/%s/pushNotificationConfigs/%s", taskID, configID)
	req.Config.Name = configName

	// Store the config
	s.pushConfigMu.Lock()
	s.pushNotificationConfigs[configName] = req.Config
	s.pushConfigMu.Unlock()

	s.writeJSONResponse(w, http.StatusCreated, req.Config)
}

// handleListPushNotificationConfigs handles GET /api/v1/push-notification-configs
func (s *Server) handleListPushNotificationConfigs(w http.ResponseWriter, r *http.Request) {
	// Get parent (task ID) from query parameters
	parent := r.URL.Query().Get("parent")
	if parent == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing parent parameter", "Parent parameter is required")
		return
	}

	// Parse task ID from parent
	if !strings.HasPrefix(parent, "tasks/") {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid parent format", "Parent must be in format tasks/{task_id}")
		return
	}

	taskID := strings.TrimPrefix(parent, "tasks/")
	if taskID == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing task ID", "Task ID is required in parent")
		return
	}

	// Verify task exists
	_, err := s.taskManager.GetTask(r.Context(), taskID)
	if err != nil {
		if a2aErr, ok := err.(*types.A2AError); ok && a2aErr.Code == types.TaskNotFoundError {
			s.writeErrorResponse(w, http.StatusNotFound, "Task not found", "The specified task does not exist")
		} else {
			s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to verify task", err.Error())
		}
		return
	}

	// Find all configs for this task
	s.pushConfigMu.RLock()
	var configs []*types.TaskPushNotificationConfig
	prefix := fmt.Sprintf("tasks/%s/pushNotificationConfigs/", taskID)

	for name, config := range s.pushNotificationConfigs {
		if strings.HasPrefix(name, prefix) {
			configs = append(configs, config)
		}
	}
	s.pushConfigMu.RUnlock()

	response := &types.ListTaskPushNotificationConfigResponse{
		Configs: configs,
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// handlePushNotificationConfigOperations handles /api/v1/push-notification-configs/{configId} endpoint
func (s *Server) handlePushNotificationConfigOperations(w http.ResponseWriter, r *http.Request) {
	// Extract config ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/push-notification-configs/")
	configID := strings.Split(path, "/")[0]

	if configID == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing config ID", "Config ID is required in the URL path")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPushNotificationConfig(w, r, configID)
	case http.MethodDelete:
		s.handleDeletePushNotificationConfig(w, r, configID)
	default:
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only GET and DELETE are supported for push notification config operations")
	}
}

// handleGetPushNotificationConfig handles GET /api/v1/push-notification-configs/{configId}
func (s *Server) handleGetPushNotificationConfig(w http.ResponseWriter, r *http.Request, configID string) {
	// Get task ID from query parameter (required since we need the full resource path)
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing task_id parameter", "task_id query parameter is required")
		return
	}

	// Construct full config name
	configName := fmt.Sprintf("tasks/%s/pushNotificationConfigs/%s", taskID, configID)

	// Get the config from storage
	s.pushConfigMu.RLock()
	config, exists := s.pushNotificationConfigs[configName]
	s.pushConfigMu.RUnlock()

	if !exists {
		s.writeErrorResponse(w, http.StatusNotFound, "Push notification config not found", "The specified config does not exist")
		return
	}

	s.writeJSONResponse(w, http.StatusOK, config)
}

// handleDeletePushNotificationConfig handles DELETE /api/v1/push-notification-configs/{configId}
func (s *Server) handleDeletePushNotificationConfig(w http.ResponseWriter, r *http.Request, configID string) {
	// Get task ID from query parameter (required since we need the full resource path)
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing task_id parameter", "task_id query parameter is required")
		return
	}

	// Construct full config name
	configName := fmt.Sprintf("tasks/%s/pushNotificationConfigs/%s", taskID, configID)

	// Check if config exists and delete it
	s.pushConfigMu.Lock()
	defer s.pushConfigMu.Unlock()

	_, exists := s.pushNotificationConfigs[configName]
	if !exists {
		s.writeErrorResponse(w, http.StatusNotFound, "Push notification config not found", "The specified config does not exist")
		return
	}

	// Delete the config
	delete(s.pushNotificationConfigs, configName)

	// Return empty success response
	w.WriteHeader(http.StatusNoContent)
}

// writeJSONResponse writes a JSON response with the given status code
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Fallback error response if encoding fails
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to encode response"}`))
	}
}

// writeErrorResponse writes an error response in A2A error format
func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, details string) {
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    statusCode,
			"message": message,
			"details": details,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResp)
}

// Validate validates the server configuration
func (s *Server) Validate() error {
	if s.agentCard == nil {
		return fmt.Errorf("agent card is required")
	}

	return nil
}

// GetRegisteredEndpoints returns a list of registered REST endpoints
func (s *Server) GetRegisteredEndpoints() []string {
	return []string{
		"GET /api/v1/agent-card",
		"POST /api/v1/messages",
		"GET /api/v1/tasks",
		"GET /api/v1/tasks/{taskId}",
		"DELETE /api/v1/tasks/{taskId}",
		"POST /api/v1/push-notification-configs",
		"GET /api/v1/push-notification-configs",
		"GET /api/v1/push-notification-configs/{configId}",
		"DELETE /api/v1/push-notification-configs/{configId}",
	}
}

// SetAgentCard updates the agent card served by the REST server
func (s *Server) SetAgentCard(agentCard *types.AgentCard) {
	s.agentCard = agentCard
}

// SetTaskManager sets the task manager for handling task operations
func (s *Server) SetTaskManager(taskManager types.TaskManagerInterface) {
	s.taskManager = taskManager
}

// handleSendMessageWithHandler processes message through handler interface
func (s *Server) handleSendMessageWithHandler(w http.ResponseWriter, r *http.Request, sendReq *types.SendMessageRequest, handlerTaskManager interface {
	ProcessMessage(ctx context.Context, req *handler.MessageRequest) (*handler.MessageResponse, error)
}) {
	// Create task first to get task ID
	task, err := s.taskManager.CreateTask(r.Context(), sendReq.Request.ContextID, sendReq.Request, sendReq.Metadata)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create task", err.Error())
		return
	}

	// Create task operations for handler
	eventChan := make(chan types.StreamResponse, 100)
	taskOps := handler.NewTaskOperations(s.taskManager, task.ID, sendReq.Request.ContextID, eventChan)

	// Determine configuration from request
	config := &handler.MessageConfiguration{
		Blocking:  false, // REST is non-blocking by default
		Streaming: false,
		Timeout:   30 * time.Second,
	}

	if sendReq.Configuration != nil {
		config.Blocking = sendReq.Configuration.Blocking
		config.HistoryLength = int(sendReq.Configuration.HistoryLength)
		config.AcceptedOutputModes = sendReq.Configuration.AcceptedOutputModes
		config.PushNotification = sendReq.Configuration.PushNotification
	}

	// Build message request for handler
	messageReq := &handler.MessageRequest{
		Message:        sendReq.Request,
		TaskID:         task.ID,
		ContextID:      sendReq.Request.ContextID,
		Configuration:  config,
		History:        task.History,
		Metadata:       sendReq.Metadata,
		TaskOperations: taskOps,
	}

	// Process through handler
	response, err := handlerTaskManager.ProcessMessage(r.Context(), messageReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Handler processing failed", err.Error())
		return
	}

	// Handle response based on mode
	switch response.Mode {
	case handler.ResponseModeTask:
		// Return task response
		if response.Result != nil && response.Result.Data != nil {
			if taskData, ok := response.Result.Data.(*types.Task); ok {
				taskResponse := &types.TaskSendMessageResponse{Task: taskData}
				s.writeJSONResponse(w, http.StatusOK, taskResponse)
				return
			}
		}
	case handler.ResponseModeMessage:
		// Return message response
		if response.Result != nil && response.Result.Data != nil {
			if msgData, ok := response.Result.Data.(*types.Message); ok {
				msgResponse := &types.MessageSendMessageResponse{Message: msgData}
				s.writeJSONResponse(w, http.StatusOK, msgResponse)
				return
			}
		}
	case handler.ResponseModeStream:
		// For streaming mode, return the task and let client poll for updates
		if response.Result != nil && response.Result.Data != nil {
			if taskData, ok := response.Result.Data.(*types.Task); ok {
				taskResponse := &types.TaskSendMessageResponse{Task: taskData}
				s.writeJSONResponse(w, http.StatusOK, taskResponse)
				return
			}
		}
	}

	// Fallback - return original task
	taskResponse := &types.TaskSendMessageResponse{Task: task}
	s.writeJSONResponse(w, http.StatusOK, taskResponse)
}
