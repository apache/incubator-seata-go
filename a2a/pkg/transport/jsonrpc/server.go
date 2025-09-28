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

package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"seata-go-ai-a2a/pkg/handler"
	"seata-go-ai-a2a/pkg/types"
)

// HandlerFunc represents a JSON-RPC method handler
type HandlerFunc func(ctx context.Context, params interface{}) (interface{}, error)

// Server represents a JSON-RPC 2.0 server implementation for A2A protocol
type Server struct {
	handlers     map[string]HandlerFunc
	corsOrigins  []string
	allowAllCORS bool
}

// NewServer creates a new JSON-RPC server
func NewServer() *Server {
	return &Server{
		handlers:     make(map[string]HandlerFunc),
		allowAllCORS: true, // Default to allowing all origins
	}
}

// RegisterHandler registers a handler for a specific JSON-RPC method
func (s *Server) RegisterHandler(method string, handler HandlerFunc) {
	s.handlers[method] = handler
}

// RegisterA2AHandlers registers handlers for A2A protocol methods
func (s *Server) RegisterA2AHandlers(agentCard *types.AgentCard, taskManager types.TaskManagerInterface) {
	// Register message/send handler
	s.RegisterHandler("message/send", func(ctx context.Context, params interface{}) (interface{}, error) {
		// Parse SendMessageRequest from params
		var req types.SendMessageRequest
		if err := parseParams(params, &req); err != nil {
			return nil, NewError(InvalidParams, "Invalid parameters for message/send", err.Error())
		}

		// Validate the request
		if req.Request == nil {
			return nil, NewError(InvalidParams, "Missing message in request", nil)
		}

		// Extract context ID from message or generate a new one
		contextID := req.Request.ContextID
		if contextID == "" {
			contextID = fmt.Sprintf("ctx_%s", generateUUID())
		}

		// Set context ID in the message if not already set
		if req.Request.ContextID == "" {
			req.Request.ContextID = contextID
		}

		// Check if task manager has handler interfaces
		if handlerTaskManager, ok := interface{}(taskManager).(interface {
			ProcessMessage(ctx context.Context, req *handler.MessageRequest) (*handler.MessageResponse, error)
		}); ok {
			// Use handler interface for processing
			return s.handleSendMessageWithHandler(ctx, &req, handlerTaskManager, taskManager)
		}

		// Fallback to legacy direct task creation
		task, err := taskManager.CreateTask(ctx, contextID, req.Request, req.Metadata)
		if err != nil {
			return nil, NewError(InternalError, "Failed to create task", err.Error())
		}

		// If blocking mode is requested and the task is still submitted,
		// update it to working state for demonstration
		if req.Configuration != nil && req.Configuration.Blocking {
			err = taskManager.UpdateTaskStatus(ctx, task.ID, types.TaskStateWorking, nil)
			if err != nil {
				// Log error but don't fail the request
			}
		}

		return &types.TaskSendMessageResponse{Task: task}, nil
	})

	// Register agentCard/get handler
	s.RegisterHandler("agentCard/get", func(ctx context.Context, params interface{}) (interface{}, error) {
		// Check if task manager has agent card provider
		if cardProvider, ok := interface{}(taskManager).(interface {
			GetAgentCard(ctx context.Context) (*types.AgentCard, error)
		}); ok {
			// Use provider interface
			card, err := cardProvider.GetAgentCard(ctx)
			if err != nil {
				return nil, NewError(InternalError, "Failed to get agent card from provider", err.Error())
			}
			return card, nil
		}

		// Fallback to static agent card
		if agentCard == nil {
			return nil, NewError(InternalError, "No agent card available", nil)
		}
		return agentCard, nil
	})

	// Register tasks/get handler
	s.RegisterHandler("tasks/get", func(ctx context.Context, params interface{}) (interface{}, error) {
		var req types.GetTaskRequest
		if err := parseParams(params, &req); err != nil {
			return nil, NewError(InvalidParams, "Invalid parameters for tasks/get", err.Error())
		}

		// Parse task ID from name (format: tasks/{task_id})
		taskID := parseTaskIDFromName(req.Name)
		if taskID == "" {
			return nil, NewError(InvalidParams, "Invalid task name format, expected tasks/{task_id}", req.Name)
		}

		// Get task using the task manager
		task, err := taskManager.GetTask(ctx, taskID)
		if err != nil {
			// Check if it's an A2A error
			if a2aErr, ok := err.(*types.A2AError); ok {
				return nil, ConvertA2AErrorToJSONRPCError(a2aErr)
			}
			return nil, NewError(InternalError, "Failed to get task", err.Error())
		}

		return task, nil
	})

	// Register tasks/cancel handler
	s.RegisterHandler("tasks/cancel", func(ctx context.Context, params interface{}) (interface{}, error) {
		var req types.CancelTaskRequest
		if err := parseParams(params, &req); err != nil {
			return nil, NewError(InvalidParams, "Invalid parameters for tasks/cancel", err.Error())
		}

		// Parse task ID from name (format: tasks/{task_id})
		taskID := parseTaskIDFromName(req.Name)
		if taskID == "" {
			return nil, NewError(InvalidParams, "Invalid task name format, expected tasks/{task_id}", req.Name)
		}

		// Cancel task using the task manager
		task, err := taskManager.CancelTask(ctx, taskID)
		if err != nil {
			// Check if it's an A2A error
			if a2aErr, ok := err.(*types.A2AError); ok {
				return nil, ConvertA2AErrorToJSONRPCError(a2aErr)
			}
			return nil, NewError(InternalError, "Failed to cancel task", err.Error())
		}

		return task, nil
	})

	// Register tasks/list handler
	s.RegisterHandler("tasks/list", func(ctx context.Context, params interface{}) (interface{}, error) {
		var req types.ListTasksRequest
		if err := parseParams(params, &req); err != nil {
			return nil, NewError(InvalidParams, "Invalid parameters for tasks/list", err.Error())
		}

		// List tasks using the task manager
		response, err := taskManager.ListTasks(ctx, &req)
		if err != nil {
			return nil, NewError(InternalError, "Failed to list tasks", err.Error())
		}

		return response, nil
	})

	// Add more handlers as needed...
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	s.setCORSHeaders(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight OPTIONS requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST requests
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, NewError(InternalError, "Failed to read request body", err.Error()), nil)
		return
	}

	// Parse JSON-RPC request
	req, err := ParseRequest(body)
	if err != nil {
		if jsonErr, ok := err.(*Error); ok {
			s.writeErrorResponse(w, jsonErr, nil)
		} else {
			s.writeErrorResponse(w, NewError(ParseError, "Parse error", err.Error()), nil)
		}
		return
	}

	// Handle the request
	s.handleRequest(w, r, req)
}

// handleRequest handles a parsed JSON-RPC request
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request, req *Request) {
	// Check if handler exists
	handler, exists := s.handlers[req.Method]
	if !exists {
		s.writeErrorResponse(w, NewError(MethodNotFound, "Method not found", req.Method), req.ID)
		return
	}

	// Call the handler
	result, err := handler(r.Context(), req.Params)
	if err != nil {
		// Convert various error types to JSON-RPC errors
		var jsonErr *Error

		switch e := err.(type) {
		case *Error:
			jsonErr = e
		case *types.A2AError:
			jsonErr = ConvertA2AErrorToJSONRPCError(e)
		default:
			jsonErr = NewError(InternalError, "Internal error", e.Error())
		}

		s.writeErrorResponse(w, jsonErr, req.ID)
		return
	}

	// Write success response
	resp := NewSuccessResponse(result, req.ID)
	s.writeResponse(w, resp)
}

// writeResponse writes a JSON-RPC response
func (s *Server) writeResponse(w http.ResponseWriter, resp *Response) {
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Fallback error response if encoding fails
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":null}`))
	}
}

// writeErrorResponse writes a JSON-RPC error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, err *Error, id interface{}) {
	resp := NewErrorResponse(err, id)

	// Set appropriate HTTP status code based on JSON-RPC error code
	switch err.Code {
	case ParseError, InvalidRequest, InvalidParams:
		w.WriteHeader(http.StatusBadRequest)
	case MethodNotFound:
		w.WriteHeader(http.StatusNotFound)
	case TaskNotFoundError:
		w.WriteHeader(http.StatusNotFound)
	case UnsupportedOperationError:
		w.WriteHeader(http.StatusNotImplemented)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(resp)
}

// parseParams parses JSON-RPC params into a target struct
func parseParams(params interface{}, target interface{}) error {
	if params == nil {
		return nil
	}

	// Convert params to JSON and then unmarshal into target
	jsonData, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal params: %w", err)
	}

	return nil
}

// GetRegisteredMethods returns a list of registered method names
func (s *Server) GetRegisteredMethods() []string {
	methods := make([]string, 0, len(s.handlers))
	for method := range s.handlers {
		methods = append(methods, method)
	}
	return methods
}

// HasMethod checks if a method is registered
func (s *Server) HasMethod(method string) bool {
	_, exists := s.handlers[method]
	return exists
}

// GetSupportedMethods returns the A2A methods supported by this server
func (s *Server) GetSupportedMethods() []string {
	supported := make([]string, 0, len(MethodMapping))
	for a2aMethod, jsonRPCMethod := range MethodMapping {
		if s.HasMethod(jsonRPCMethod) {
			supported = append(supported, a2aMethod)
		}
	}
	return supported
}

// SetCORSOrigins sets allowed CORS origins
func (s *Server) SetCORSOrigins(origins []string) {
	s.corsOrigins = make([]string, len(origins))
	copy(s.corsOrigins, origins)
	s.allowAllCORS = len(origins) == 0
}

// Validate validates the server configuration
func (s *Server) Validate() error {
	if len(s.handlers) == 0 {
		return fmt.Errorf("no handlers registered")
	}

	// Check that essential A2A methods are implemented
	requiredMethods := []string{"agentCard/get"}
	var missing []string

	for _, method := range requiredMethods {
		if !s.HasMethod(method) {
			missing = append(missing, method)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required methods: %s", strings.Join(missing, ", "))
	}

	return nil
}

// parseTaskIDFromName extracts task ID from resource name format (tasks/{task_id})
func parseTaskIDFromName(name string) string {
	const prefix = "tasks/"
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	return name[len(prefix):]
}

// generateUUID generates a new UUID string
func generateUUID() string {
	return uuid.New().String()
}

// setCORSHeaders sets CORS headers based on configured origins
func (s *Server) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	if s.allowAllCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}

	// Check if origin is in allowed list
	for _, allowedOrigin := range s.corsOrigins {
		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			return
		}
	}

	// Origin not allowed, don't set CORS headers
}

// handleSendMessageWithHandler processes message through handler interface for JSON-RPC
func (s *Server) handleSendMessageWithHandler(ctx context.Context, sendReq *types.SendMessageRequest, handlerTaskManager interface {
	ProcessMessage(ctx context.Context, req *handler.MessageRequest) (*handler.MessageResponse, error)
}, taskManager types.TaskManagerInterface) (interface{}, error) {
	// Create task first to get task ID
	task, err := taskManager.CreateTask(ctx, sendReq.Request.ContextID, sendReq.Request, sendReq.Metadata)
	if err != nil {
		return nil, NewError(InternalError, "Failed to create task", err.Error())
	}

	// Create task operations for handler
	eventChan := make(chan types.StreamResponse, 100)
	taskOps := handler.NewTaskOperations(taskManager, task.ID, sendReq.Request.ContextID, eventChan)

	// Determine configuration from request
	config := &handler.MessageConfiguration{
		Blocking:  false, // JSON-RPC is non-blocking by default
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
	response, err := handlerTaskManager.ProcessMessage(ctx, messageReq)
	if err != nil {
		return nil, NewError(InternalError, "Handler processing failed", err.Error())
	}

	// Handle response based on mode
	switch response.Mode {
	case handler.ResponseModeTask:
		// Return task response
		if response.Result != nil && response.Result.Data != nil {
			if taskData, ok := response.Result.Data.(*types.Task); ok {
				return &types.TaskSendMessageResponse{Task: taskData}, nil
			}
		}
	case handler.ResponseModeMessage:
		// Return message response
		if response.Result != nil && response.Result.Data != nil {
			if msgData, ok := response.Result.Data.(*types.Message); ok {
				return &types.MessageSendMessageResponse{Message: msgData}, nil
			}
		}
	}

	// Fallback - return original task
	return &types.TaskSendMessageResponse{Task: task}, nil
}
